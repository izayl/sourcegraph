// Code generated by github.com/efritz/go-mockgen 0.1.0; DO NOT EDIT.

package discovery

import (
	"context"
	api "github.com/sourcegraph/sourcegraph/internal/api"
	"sync"
)

// MockSettingStore is a mock implementation of the SettingStore interface
// (from the package
// github.com/sourcegraph/sourcegraph/enterprise/internal/insights/discovery)
// used for unit testing.
type MockSettingStore struct {
	// GetLatestFunc is an instance of a mock function object controlling
	// the behavior of the method GetLatest.
	GetLatestFunc *SettingStoreGetLatestFunc
}

// NewMockSettingStore creates a new mock of the SettingStore interface. All
// methods return zero values for all results, unless overwritten.
func NewMockSettingStore() *MockSettingStore {
	return &MockSettingStore{
		GetLatestFunc: &SettingStoreGetLatestFunc{
			defaultHook: func(context.Context, api.SettingsSubject) (*api.Settings, error) {
				return nil, nil
			},
		},
	}
}

// NewMockSettingStoreFrom creates a new mock of the MockSettingStore
// interface. All methods delegate to the given implementation, unless
// overwritten.
func NewMockSettingStoreFrom(i SettingStore) *MockSettingStore {
	return &MockSettingStore{
		GetLatestFunc: &SettingStoreGetLatestFunc{
			defaultHook: i.GetLatest,
		},
	}
}

// SettingStoreGetLatestFunc describes the behavior when the GetLatest
// method of the parent MockSettingStore instance is invoked.
type SettingStoreGetLatestFunc struct {
	defaultHook func(context.Context, api.SettingsSubject) (*api.Settings, error)
	hooks       []func(context.Context, api.SettingsSubject) (*api.Settings, error)
	history     []SettingStoreGetLatestFuncCall
	mutex       sync.Mutex
}

// GetLatest delegates to the next hook function in the queue and stores the
// parameter and result values of this invocation.
func (m *MockSettingStore) GetLatest(v0 context.Context, v1 api.SettingsSubject) (*api.Settings, error) {
	r0, r1 := m.GetLatestFunc.nextHook()(v0, v1)
	m.GetLatestFunc.appendCall(SettingStoreGetLatestFuncCall{v0, v1, r0, r1})
	return r0, r1
}

// SetDefaultHook sets function that is called when the GetLatest method of
// the parent MockSettingStore instance is invoked and the hook queue is
// empty.
func (f *SettingStoreGetLatestFunc) SetDefaultHook(hook func(context.Context, api.SettingsSubject) (*api.Settings, error)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// GetLatest method of the parent MockSettingStore instance invokes the hook
// at the front of the queue and discards it. After the queue is empty, the
// default hook function is invoked for any future action.
func (f *SettingStoreGetLatestFunc) PushHook(hook func(context.Context, api.SettingsSubject) (*api.Settings, error)) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultDefaultHook with a function that returns
// the given values.
func (f *SettingStoreGetLatestFunc) SetDefaultReturn(r0 *api.Settings, r1 error) {
	f.SetDefaultHook(func(context.Context, api.SettingsSubject) (*api.Settings, error) {
		return r0, r1
	})
}

// PushReturn calls PushDefaultHook with a function that returns the given
// values.
func (f *SettingStoreGetLatestFunc) PushReturn(r0 *api.Settings, r1 error) {
	f.PushHook(func(context.Context, api.SettingsSubject) (*api.Settings, error) {
		return r0, r1
	})
}

func (f *SettingStoreGetLatestFunc) nextHook() func(context.Context, api.SettingsSubject) (*api.Settings, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *SettingStoreGetLatestFunc) appendCall(r0 SettingStoreGetLatestFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of SettingStoreGetLatestFuncCall objects
// describing the invocations of this function.
func (f *SettingStoreGetLatestFunc) History() []SettingStoreGetLatestFuncCall {
	f.mutex.Lock()
	history := make([]SettingStoreGetLatestFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// SettingStoreGetLatestFuncCall is an object that describes an invocation
// of method GetLatest on an instance of MockSettingStore.
type SettingStoreGetLatestFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 context.Context
	// Arg1 is the value of the 2nd argument passed to this method
	// invocation.
	Arg1 api.SettingsSubject
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 *api.Settings
	// Result1 is the value of the 2nd result returned from this method
	// invocation.
	Result1 error
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c SettingStoreGetLatestFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0, c.Arg1}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c SettingStoreGetLatestFuncCall) Results() []interface{} {
	return []interface{}{c.Result0, c.Result1}
}