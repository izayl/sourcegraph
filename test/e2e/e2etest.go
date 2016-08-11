package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/context"

	"sourcegraph.com/sourcegraph/go-selenium"
	"sourcegraph.com/sourcegraph/sourcegraph/api/sourcegraph"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/auth/idkey"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/auth/sharedsecret"
	"sourcegraph.com/sourcegraph/sourcegraph/test/e2e/e2etestuser"

	"github.com/jpillora/backoff"
	"github.com/neelance/parallel"
	"github.com/nlopes/slack"
	"github.com/prometheus/client_golang/prometheus"
)

// T is passed as context into all tests. It provides generic helper methods to
// make life during testing easier.
type T struct {
	selenium.WebDriverT

	// Target is the target Sourcegraph server to test, e.g. https://sourcegraph.com
	Target *url.URL

	// TestLogin is a username prefixed with e2etestuser.Prefix which is unique
	// for this test. In specific it is e2etestuser.Prefix + Test.Name.
	TestLogin string

	// TestEmail is a email address prefixed with e2etestuser.Prefix which is
	// unique for this test. In specific it is e2etestuser.Prefix + Test.Name + "@sourcegraph.com".
	TestEmail string

	// WebDriver is the underlying selenium web driver. Useful if you want to
	// handle errors yourself (the embedded WebDriverT handles them for you by
	// calling Fatalf).
	WebDriver selenium.WebDriver

	// testingT provides a Logf implementation
	testingT TestingT

	tr *testRunner
}

// Minimal interface for what testing.T provides
type TestingT interface {
	Logf(fmt string, v ...interface{})
}

type internalError struct {
	err error
}

func (e *internalError) Error() string {
	return e.err.Error()
}

type fatalError struct {
	err string
}

func (e *fatalError) Error() string {
	return e.err
}

// Fatalf implements the TestingT and the selenium.TestingT interface.
func (t *T) Fatalf(fmtStr string, v ...interface{}) {
	// We throw a panic here and it is always recovered by runTest and
	// subsequently handled. This isn't idiomatic, but it lets us write out
	// screenshots after a go test has failed.
	panic(&fatalError{err: fmt.Sprintf(fmtStr, v...)})
}

// Logf implements TestingT
func (t *T) Logf(fmtStr string, v ...interface{}) {
	t.testingT.Logf(fmtStr, v...)
}

// Endpoint returns an absolute URL given one relative to the target instance
// root. For example, if t.Target == "https://sourcegraph.com", Endpoint("/login")
// will return "https://sourcegraph.com/login"
func (t *T) Endpoint(e string) string {
	// Manually join the URL components in order to retain query parameters,
	// etc. that might be specified in e. This works as long as e is an
	// absolute path (aka e starts with a slash).
	u := t.Target.String()
	if strings.HasSuffix(u, "/") {
		u = strings.TrimSuffix(u, "/")
	}
	return u + e
}

// TargetGRPC returns the final gRPC target URL, e.g. https://grpc.sourcegraph.com.
func (t *T) TargetGRPC() *url.URL {
	cpy := *t.Target
	endpoint := &cpy

	// HACK: Because there is no gRPC service discovery system set up in
	//       Sourcegraph, we hard-code gRPC endpoints here so that we don't
	//       have to specify TARGET_GRPC every single time (one less thing
	//       to think about).
	switch endpoint.Host {
	case "sourcegraph.com":
		endpoint.Host = "grpc.sourcegraph.com"
	case "staging.sourcegraph.com":
		endpoint.Host = "grpc-staging.sourcegraph.com"
	case "staging2.sourcegraph.com":
		endpoint.Host = "grpc-staging2.sourcegraph.com"
	case "staging3.sourcegraph.com":
		endpoint.Host = "grpc-staging3.sourcegraph.com"
	case "staging4.sourcegraph.com":
		endpoint.Host = "grpc-staging4.sourcegraph.com"
	}

	if grpc := os.Getenv("TARGET_GRPC"); grpc != "" {
		var err error
		endpoint, err = url.Parse(grpc)
		if err != nil {
			t.Fatalf("could not parse TARGET_GRPC as url: %s", err)
		}
	}
	return endpoint
}

// GRPCClient returns a new authenticated Sourcegraph gRPC client. It uses the
// server's ID key, and thus has 100% unrestricted access. Use with caution!
func (t *T) GRPCClient() (context.Context, *sourcegraph.Client) {
	// Create context with gRPC endpoint and idKey credentials.
	ctx := context.Background()
	ctx = sourcegraph.WithGRPCEndpoint(ctx, t.TargetGRPC())
	ctx = sourcegraph.WithCredentials(ctx, sharedsecret.TokenSource(t.tr.idKey, "internal:e2etest"))

	// Create client.
	c, err := sourcegraph.NewClientFromContext(ctx)
	if err != nil {
		t.Fatalf("could not create gRPC client: %v", c)
	}
	return ctx, c
}

// WaitForCondition waits up to d for cond() to be true. After each time cond()
// is called, time.Sleep(optimisticD) is called. If timeout occurs, t.Fatalf is
// invoked.
func (t *T) WaitForCondition(d time.Duration, optimisticD time.Duration, cond func() bool, condName string) {
	start := time.Now()
	for time.Now().Sub(start) < d {
		time.Sleep(optimisticD)
		if cond() {
			return
		}
	}
	t.Fatalf("timed out waiting %v for condition %q to be met", d, condName)
}

// WaitForElement waits up to 20s for an element that matches the given selector and filters.
func (t *T) WaitForElement(by, value string, filters ...ElementFilter) selenium.WebElement {
	var element selenium.WebElement
	t.WaitForCondition(
		20*time.Second,
		100*time.Millisecond,
		func() bool {
			elements, err := t.WebDriver.FindElements(by, value)
			if err != nil {
				return false
			}
			if seleniumTrace {
				t.Logf("WaitForElement: %d matches for (%s, %q)", len(elements), by, value)
			}
			f := And(filters...)
			for _, e := range elements {
				if f(e) {
					element = e
					return true
				}
			}
			if seleniumTrace {
				t.Logf("WaitForElement: failed to find filter match")
			}
			return false
		},
		fmt.Sprintf("Wait for element to appear: %s %q", by, value),
	)
	return element
}

// Click waits up to 20s for an element that matches the given selector and
// filters to appear, and then clicks it.
//
// ChromeDriver has a race condition where if an element moves while an
// attempt to click on it happens, it throws an internal exception. This
// function will retry on errors from Click to workaround the issue. This may
// add an extra 20s on the wait time.
func (t *T) Click(by, value string, filters ...ElementFilter) {
	var err error
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		e := t.WaitForElement(by, value, filters...)
		err = e.Click()
		if err == nil {
			return
		}
	}
	t.Fatalf("Failed to click (%s, %s): %s", by, value, err)
}

type ElementFilter func(selenium.WebElement) bool

func And(filters ...ElementFilter) ElementFilter {
	return func(e selenium.WebElement) bool {
		for _, f := range filters {
			if !f(e) {
				return false
			}
		}
		return true
	}
}

func Or(filters ...ElementFilter) ElementFilter {
	return func(e selenium.WebElement) bool {
		for _, f := range filters {
			if f(e) {
				return true
			}
		}
		return false
	}
}

func MatchAttribute(attr, pattern string) ElementFilter {
	r := regexp.MustCompile(pattern)
	return func(e selenium.WebElement) bool {
		href, err := e.GetAttribute(attr)
		if err != nil {
			return false
		}
		return r.MatchString(href)
	}
}

// WaitForRedirect waits up to 20s for a redirect to the given URL (e.g.,
// "https://sourcegraph.com/login").
//
// Use t.Endpoint("/foo") to get an endpoint relative to $TARGET easily.
func (t *T) WaitForRedirect(url, description string) {
	t.WaitForCondition(
		20*time.Second,
		100*time.Millisecond,
		func() bool {
			currentURL, err := t.WebDriver.CurrentURL()
			if err != nil {
				return false
			}
			return currentURL == url
		},
		fmt.Sprintf("%s (%s)", description, url),
	)
}

// WaitForRedirectPrefix waits up to 20s for a redirect to a page with the
// given prefix (e.g., "https://github.com/login" matches if the URL is really
// "https://github.com/login?foo").
func (t *T) WaitForRedirectPrefix(prefix, description string) {
	t.WaitForCondition(
		20*time.Second,
		100*time.Millisecond,
		func() bool {
			currentURL, err := t.WebDriver.CurrentURL()
			if err != nil {
				return false
			}
			return strings.HasPrefix(currentURL, prefix)
		},
		fmt.Sprintf("%s (%s)", description, prefix),
	)
}

// WaitForRedirectSuffix waits up to 20s for a redirect to a page with the
// given suffix (e.g., "login" matches if the URL is really
// "https://github.com/login").
func (t *T) WaitForRedirectSuffix(suffix, description string) {
	t.WaitForCondition(
		20*time.Second,
		100*time.Millisecond,
		func() bool {
			currentURL, err := t.WebDriver.CurrentURL()
			if err != nil {
				return false
			}
			return strings.HasSuffix(currentURL, suffix)
		},
		fmt.Sprintf("%s (%s)", description, suffix),
	)
}

// Test represents a single E2E test.
type Test struct {
	// Name is the name of your test, which should be short and readable, e.g.,
	// "register_and_login".
	Name string

	// Description is a more verbose description of your test, e.g., "Registers a
	// new user account and logs in to it.".
	Description string

	// Func is called to perform the test. If an error is returned, the test is
	// considered failed.
	//
	// Tests must log all output to t.Log instead of via other logging packages.
	Func func(t *T) error

	// Quarantined tests are run as usual, but their failures are not
	// reported. This is useful for understanding the effectiveness of new
	// tests / temporarily disabling bad tests
	Quarantined bool
}

// Register should be called inside of an init function in order to register a
// new test as part of the testsuite.
func Register(t *Test) {
	tr.tests = append(tr.tests, t)
}

// testRunner is provided as input to each test and provides generic helper
// methods to make testing easier.
type testRunner struct {
	log      *log.Logger
	target   *url.URL
	tests    []*Test
	executor string
	idKey    *idkey.IDKey

	slack                             *slack.Client
	slackToken                        string
	slackChannel, slackWarningChannel *slack.Channel
	slackLogBuffer                    *bytes.Buffer
	slackSkipAtChannel                bool
}

const (
	typeWarning = iota
	typeNormal
)

func (t *testRunner) slackMessage(messageType int, msg, quoted string) {
	if t.slack == nil {
		return
	}
	if messageType == typeWarning && t.slackWarningChannel == nil {
		return
	}

	params := slack.PostMessageParameters{
		Username:  "e2etest",
		Parse:     "full",
		IconEmoji: ":shield:",
		Attachments: []slack.Attachment{
			slack.Attachment{
				Text: quoted,
			},
		},
	}
	id := t.slackChannel.ID
	if messageType == typeWarning {
		id = t.slackWarningChannel.ID
	}
	_, _, err := t.slack.PostMessage(id, msg, params)
	if err != nil {
		log.Println(err)
		return
	}
}

var runCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "src",
	Subsystem: "e2etest",
	Name:      "run",
	Help:      "Number of times the testsuite has run",
}, []string{"state"})
var testCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "src",
	Subsystem: "e2etest",
	Name:      "test",
	Help:      "Number of times an individual test has run",
}, []string{"name", "state"})

func init() {
	prometheus.MustRegister(runCounter)
	prometheus.MustRegister(testCounter)
}

// run runs the test suite over and over again against $TARGET, if $TARGET is set,
// otherwise it runs the test suite just once.
func (t *testRunner) run() {
	b := &backoff.Backoff{
		Min: time.Second,
		Max: 10 * time.Minute,
	}
	shouldLogSuccess := 0
	for {
		if t.runTests(shouldLogSuccess < 5) {
			b.Reset()
			shouldLogSuccess++
			if shouldLogSuccess == 5 {
				t.slackMessage(typeNormal, ":star: *Five consecutive successes!* (silencing output until next failure)", "")
			}
		} else {
			time.Sleep(b.Duration())
			shouldLogSuccess = 0
		}

		if *runOnce {
			break
		}
	}
}

// runTests runs all of the tests and handles failures. It returns whether or
// not all tests were successful.
func (t *testRunner) runTests(logSuccess bool) bool {
	// Execute the registered tests in parallel.
	var (
		failuresMu sync.Mutex
		failures   int
		start      = time.Now()
		run        = parallel.NewRun(1) // currently not running tests in parallel because of Selenium errors
		total      = 0
	)
	for _, testToCopy := range t.tests {
		// If they want to run specifically just one test, check for that now.
		if *runFlag != "" {
			if testToCopy.Name != *runFlag {
				continue
			}
		}
		total++

		test := testToCopy
		run.Acquire()
		go func() {
			defer run.Release()
			// Attempt the test a number of times, to weed out any flakiness that could occur.
			for attempt := 0; attempt < *retriesFlag; attempt++ {
				unitStart := time.Now()
				wouldAttemptAgain := attempt+1 < *retriesFlag

				// This error should not be bubbled up! That will cause the parallel.Run to short circuit,
				// but we want all tests to run regardless.
				err, screenshot := t.runTest(test, nil)
				if _, ok := err.(*internalError); ok {
					t.log.Printf("[warning] [%v] unable to establish a session: %v\n", test.Name, err)
					t.slackMessage(typeWarning, fmt.Sprintf("Test %v failed due to inability to establish a connection: %v", test.Name, err), "")
					testCounter.WithLabelValues(test.Name, "error").Inc()
					return
				}

				unitTime := time.Since(unitStart)
				if err != nil {
					// If we would attempt this test again, then just log warnings and retry.
					if !*runOnce && wouldAttemptAgain {
						msg := fmt.Sprintf("[warning] [attempt %v failed] [%v] [%v]: %v\n", attempt, test.Name, unitTime, err)
						t.log.Printf(msg)
						t.slackMessage(typeWarning, msg, "")
						if err := t.slackFileUpload(typeWarning, screenshot, test.Name+".png"); err != nil {
							t.log.Println("could not upload screenshot to Slack", test.Name, err)
						}
						testCounter.WithLabelValues(test.Name, "retry").Inc()
						continue
					}

					failureType := typeNormal
					if test.Quarantined {
						failureType = typeWarning
					}

					if failureType == typeNormal {
						failuresMu.Lock()
						failures++
						failuresMu.Unlock()
					}

					testCounter.WithLabelValues(test.Name, "failure").Inc()

					if err := t.slackFileUpload(failureType, screenshot, test.Name+".png"); err != nil {
						t.log.Println("could not upload screenshot to Slack", test.Name, err)
					}

					if t.slack != nil {
						t.log.Printf("[failure] [%v] [%v]: %v\n", test.Name, unitTime, err)
					} else {
						// Include the screenshot location when running without slack.
						t.log.Printf("[failure] [%v] [%v]: %v (see ./%s.png)\n", test.Name, unitTime, err, test.Name)
					}
					return
				}
				t.log.Printf("[success] [%v] [%v]\n", test.Name, unitTime)
				testCounter.WithLabelValues(test.Name, "success").Inc()
				return
			}
			panic("never here")
		}()
	}
	run.Wait()

	t.log.Printf("%v tests finished in %v [%v success] [%v failure]\n", total, time.Since(start), total-failures, failures)

	if failures == 0 {
		runCounter.WithLabelValues("success").Inc()
		t.slackSkipAtChannel = false // do @channel on next failure
		if logSuccess {
			t.slackMessage(typeNormal, fmt.Sprintf(":thumbsup: *Success! %v tests successful against %v!*", total, t.target.String()), "")
		}
	} else {
		runCounter.WithLabelValues("failure").Inc()

		// Only send @channel on the first failure, not all consecutive ones (that
		// would be very annoying).
		atChannel := ""
		if !t.slackSkipAtChannel {
			t.slackSkipAtChannel = true
			atChannel = " @channel"
		}
		t.slackMessage(
			typeNormal,
			fmt.Sprintf(":fire: *FAILURE! %v/%v tests failed against %v: *"+atChannel, failures, total, t.target.String()),
			t.slackLogBuffer.String(),
		)

		// Create an OpsGenie alert.
		err := t.alertOpsGenie(fmt.Sprintf("[e2etest]: %v/%v tests failed against %v", failures, total, t.target.String()))
		if err != nil {
			t.log.Printf("[WARNING] error while sending alert to opsgenie %s\n", err)
		}
	}
	t.slackLogBuffer.Reset()
	return failures == 0
}

var opsGenieClient = &http.Client{
	Timeout: 5 * time.Second,
}

func (t *testRunner) alertOpsGenie(msg string) error {
	key := os.Getenv("OPSGENIE_KEY")
	if key == "" {
		return nil
	}

	// We could use the OpsGenie Go SDK for this, but it's quite large and our
	// usage of their API here is very small, so we don't right now.
	payload := map[string]string{
		"apiKey":  key,
		"alias":   "e2etest", // using the same
		"message": msg,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", "https://api.opsgenie.com/v1/json/alert", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := opsGenieClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("opsgenie alert creation returned status=%d body=%q", resp.StatusCode, string(data))
	}
	return nil
}

// runTest runs a single test and recovers from panics, should they occur. If
// the test failed for any reason err != nil is returned. If it was possible to
// capture a screenshot of the error, screenshot will be the encoded PNG bytes.
//
// testingT is optional, and is set as T.testingT directly if specified.
func (t *testRunner) runTest(test *Test, testingT *testing.T) (err error, screenshot []byte) {
	wd, err := t.newWebDriver()
	if err != nil {
		return &internalError{err: err}, nil
	}

	// Handle things after the test has finished executing.
	defer func() {
		// Handle any panics that might occur because we are a single-process test
		// suite.
		if r := recover(); r != nil {
			if err == nil {
				err = errors.New(fmt.Sprint(r))
			}
		}

		// If there was an error, capture a screenshot of the problem.
		if err != nil {
			// Wrap the error with the current URL.
			currentURL, _ := wd.CurrentURL()
			err = fmt.Errorf("%s (on page %s)", err, currentURL)

			// Capture a screenshot of the problem.
			var err2 error
			screenshot, err2 = wd.Screenshot()
			if err2 != nil {
				t.log.Println("could not capture screenshot for", test.Name, err2)
				screenshot = nil
			}
		}
		wd.Quit()
	}()

	ctx := t.newT(test, wd)
	ctx.testingT = testingT
	return test.Func(ctx), nil
}

func (t *testRunner) newWebDriver() (selenium.WebDriver, error) {
	caps := selenium.Capabilities(map[string]interface{}{
		"browserName": "chrome",
		"chromeOptions": map[string]interface{}{
			"args": []string{"user-agent=" + e2etestuser.UserAgent},
		},
	})
	return selenium.NewRemote(caps, t.executor)
}

func (t *testRunner) newT(test *Test, wd selenium.WebDriver) *T {
	ctx := &T{
		Target:    t.target,
		TestLogin: e2etestuser.Prefix + test.Name,
		TestEmail: e2etestuser.Prefix + test.Name + "@sourcegraph.com",
		WebDriver: wd,
		testingT:  defaultTestingT{},
		tr:        t,
	}
	ctx.WebDriverT = ctx.WebDriver.T(ctx)
	return ctx
}

// slackFileUpload implements slack multipart file upload.
//
// TODO(slimsag): upstream this type of change to github.com/nlopes/slack.
func (t *testRunner) slackFileUpload(messageType int, f []byte, title string) error {
	if t.slack == nil {
		return ioutil.WriteFile(title, f, 0666)
	}
	b := &bytes.Buffer{}
	w := multipart.NewWriter(b)
	fw, err := w.CreateFormFile("file", title)
	if err != nil {
		return err
	}
	if _, err := io.Copy(fw, bytes.NewReader(f)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	// Write additional fields.
	channel := t.slackChannel.ID
	if messageType == typeWarning {
		if t.slackWarningChannel == nil {
			return nil
		}
		channel = t.slackWarningChannel.ID
	}
	fields := map[string]string{
		"channels": channel,
		"token":    t.slackToken,
	}
	for k, v := range fields {
		fw, err := w.CreateFormField(k)
		if err != nil {
			return err
		}
		if _, err := fw.Write([]byte(v)); err != nil {
			return err
		}
	}

	// Make the request.
	resp, err := http.Post("https://slack.com/api/files.upload", w.FormDataContentType(), b)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Decode the response and check ok field as the Slack API docs say to do.
	var slackResp = struct {
		Ok bool
		// TODO(slimsag): "file" object field
	}{}
	if err := json.Unmarshal(body, &slackResp); err != nil {
		return err
	}
	if !slackResp.Ok {
		return fmt.Errorf("slack file upload failed, response: %s", string(body))
	}
	return nil
}

var tr = &testRunner{
	slackLogBuffer: &bytes.Buffer{},
}

var (
	runOnce     = flag.Bool("once", true, "run the tests only once (true) or forever (false)")
	runFlag     = flag.String("run", "", "specify an exact test name to run (e.g. 'login_flow', 'register_flow')")
	retriesFlag = flag.Int("retries", 1, "maximum number of times to retry a test before considering it failed")

	seleniumTrace = os.Getenv("SELENIUM_TRACE") != ""
)

func parseEnv() error {
	// Determine which Selenium server to connect to.
	serverAddr := os.Getenv("SELENIUM_SERVER_IP")
	serverPort := os.Getenv("SELENIUM_SERVER_PORT")
	if serverAddr == "" {
		serverAddr = "localhost" // default to localhost
	}
	if serverPort == "" {
		serverPort = "4444" // default to standard Selenium port
	}

	if !seleniumTrace {
		selenium.Log = log.New(ioutil.Discard, "", 0)
	}

	if !strings.Contains(serverAddr, "://") {
		serverAddr = "http://" + serverAddr
	}

	u, err := url.Parse(fmt.Sprintf("%s:%s", serverAddr, serverPort))
	if err != nil {
		return err
	}

	u.Path = path.Join(u.Path, "wd/hub")

	tr.executor = u.String()

	// Determine the target Sourcegraph instance to test against.
	target := os.Getenv("TARGET")
	if target == "" {
		return errors.New("unable to get TARGET Sourcegraph instance from environment")
	}
	tr.target, err = url.Parse(target)
	if err != nil {
		return err
	}
	if tr.target.Scheme == "" {
		return errors.New("TARGET must specify scheme (http or https) prefix")
	}

	// Find server ID key information.
	if key := os.Getenv("ID_KEY_DATA"); key != "" {
		tr.idKey, err = idkey.FromString(key)
		if err != nil {
			return err
		}
	} else {
		sgpath := os.Getenv("SGPATH")
		if sgpath == "" {
			currentUser, err := user.Current()
			if err != nil {
				return err
			}
			sgpath = filepath.Join(currentUser.HomeDir, ".sourcegraph")
		}
		data, err := ioutil.ReadFile(filepath.Join(sgpath, "id.pem"))
		if err != nil {
			return err
		}
		tr.idKey, err = idkey.New(data)
		if err != nil {
			return err
		}
	}

	if token := os.Getenv("SLACK_API_TOKEN"); token != "" {
		tr.slack = slack.New(token)
		tr.slackToken = token

		// Determine which slack channel and warning channel we should use.
		// Find the channel IDs.
		channelName := os.Getenv("SLACK_CHANNEL")
		warningChannelName := os.Getenv("SLACK_WARNING_CHANNEL")
		if channelName == "" {
			channelName = "e2etest"
		}
		if warningChannelName == "" {
			log.Println("SLACK_WARNING_CHANNEL not configured, warnings will not appear on slack")
		}

		// Find the channel IDs.
		channels, err := tr.slack.GetChannels(true)
		if err != nil {
			return err
		}
		findChannel := func(name string) *slack.Channel {
			for _, c := range channels {
				if c.Name == name {
					return &c
				}
			}
			return nil
		}
		tr.slackChannel = findChannel(channelName)
		if tr.slackChannel == nil {
			log.Println("could not find slack channel", channelName)
			log.Println("disabling slack notifications")
			tr.slack = nil
		}
		if warningChannelName != "" {
			tr.slackWarningChannel = findChannel(warningChannelName)
			if tr.slackWarningChannel == nil {
				log.Printf("SLACK_WARNING_CHANNEL=%s does not exist, warnings will not appear on slack.", warningChannelName)
			}
		}
		if tr.slackChannel != nil {
			registeredTests := &bytes.Buffer{}
			for _, t := range tr.tests {
				fmt.Fprintf(registeredTests, "[%v]: %v\n", t.Name, t.Description)
			}
			tr.slackMessage(typeNormal, ":shield: *Ready and reporting for duty!* Registered tests:", registeredTests.String())
		}
	}

	if addr := os.Getenv("PROMETHEUS_IO_ADDR"); addr != "" {
		http.Handle("/metrics", prometheus.Handler())
		go func() {
			err := http.ListenAndServe(addr, nil)
			if err != nil {
				log.Fatal("Prometheus ListenAndServe:", err)
			}
		}()
	}

	return nil
}

func Main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, `
Environment:
  TARGET (required)
      target Sourcegraph server to test against (e.g. 'http://192.168.1.1:3080', use LAN IP due to Docker!)
  SELENIUM_SERVER_IP = "localhost"
      IP address of the Selenium server (consider consulting 'docker-machine ls' on certain Docker versions)
  SELENIUM_SERVER_PORT = "4444"
      port of the Selenium server
  SELENIUM_TRACE (optional)
      If specified, selenium actions will be verbosely logged. Useful when debugging selenium.
  WRITE_SCREENSHOTS (optional)
      If specified, screenshots of any failures will be written to the specified directory path
      relative to the test/e2e package directory. Only used by 'go test'.
  ID_KEY_DATA (optional)
      If specified, the Base64-encoded string is used in place of '$SGPATH/id.pem' for authenticating
  SLACK_API_TOKEN (optional)
      If specified, send information about tests to Slack.
  SLACK_CHANNEL = "e2etest"
      Slack channel to which test result output and test failure screenshots will be sent to.
  SLACK_WARNING_CHANNEL (optional)
      If specified, send warning (verbose) log messages to this channel instead of SLACK_CHANNEL.
  PROMETHEUS_IO_ADDR (optional)
      If specified, prometheus metric will be exported on this address (eg :6060)

Flags:
`)
		flag.PrintDefaults()
	}
	flag.Parse()

	// Prepare logging.
	tr.log = log.New(io.MultiWriter(os.Stderr, tr.slackLogBuffer), "", 0)

	err := parseEnv()
	if err != nil {
		log.Fatal(err)
	}

	tr.run()
}

type defaultTestingT struct{}

func (t defaultTestingT) Logf(fmtStr string, v ...interface{}) {
	log.Printf(fmtStr, v...)
}
