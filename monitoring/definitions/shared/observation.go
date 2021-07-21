package shared

import (
	"fmt"

	"github.com/sourcegraph/sourcegraph/monitoring/monitoring"
)

var (
	ObservationTotal observableConstructor = func(options ObservableOptions) sharedObservable {
		return func(containerName string, owner monitoring.ObservableOwner) Observable {
			filters := makeFilters(containerName, options.Filters...)
			by, legendPrefix := makeBy(options.By...)

			return Observable{
				Name:           fmt.Sprintf("%s_total", options.MetricName),
				Description:    fmt.Sprintf("%s operations every 5m", options.MetricDescription),
				Query:          fmt.Sprintf(`sum%s(increase(src_%s_total{%s}[5m]))`, by, options.MetricName, filters),
				Panel:          monitoring.Panel().LegendFormat(fmt.Sprintf("%s operations", legendPrefix)),
				Owner:          owner,
				NoAlert:        true,
				Interpretation: "none",
			}
		}
	}

	ObservationDuration observableConstructor = func(options ObservableOptions) sharedObservable {
		return func(containerName string, owner monitoring.ObservableOwner) Observable {
			filters := makeFilters(containerName, options.Filters...)
			by, _ := makeBy(append([]string{"le"}, options.By...)...)
			_, legendPrefix := makeBy(options.By...)

			return Observable{
				Name:           fmt.Sprintf("%s_99th_percentile_duration", options.MetricName),
				Description:    fmt.Sprintf("99th percentile successful %s operation duration over 5m", options.MetricDescription),
				Query:          fmt.Sprintf(`histogram_quantile(0.99, sum %s(rate(src_%s_duration_seconds_bucket{%s}[5m])))`, by, options.MetricName, filters),
				Panel:          monitoring.Panel().LegendFormat(fmt.Sprintf("%s operations", legendPrefix)).Unit(monitoring.Seconds),
				Owner:          owner,
				NoAlert:        true,
				Interpretation: "none",
			}
		}
	}

	ObservationErrors observableConstructor = func(options ObservableOptions) sharedObservable {
		return func(containerName string, owner monitoring.ObservableOwner) Observable {
			filters := makeFilters(containerName, options.Filters...)
			by, legendPrefix := makeBy(options.By...)

			return Observable{
				Name:           fmt.Sprintf("%s_errors_total", options.MetricName),
				Description:    fmt.Sprintf("%s errors every 5m", options.MetricDescription),
				Query:          fmt.Sprintf(`sum%s(increase(src_%s_errors_total{%s}[5m]))`, by, options.MetricName, filters),
				Panel:          monitoring.Panel().LegendFormat(fmt.Sprintf("%s errors", legendPrefix)),
				Owner:          owner,
				NoAlert:        true,
				Interpretation: "none",
			}
		}
	}
)

type ObservationGroupOptions struct {
	ObservableOptions

	// Total transforms the default observable used to construct the operation count panel.
	Total ObservableOption

	// Duration transforms the default observable used to construct the duration histogram panel.
	Duration ObservableOption

	// Errors transforms the default observable used to construct the error rate panel.
	Errors ObservableOption
}

// NewObservationGroup creates a group containing panels displaying the total number of operations,
// operation duration histogram, and number of errors for the given observable within the given
// container.
func NewObservationGroup(containerName string, owner monitoring.ObservableOwner, options ObservationGroupOptions) monitoring.Group {
	return monitoring.Group{
		Title:  fmt.Sprintf("[%s] Observable: %s", options.Namespace, options.GroupDescription),
		Hidden: options.Hidden,
		Rows: []monitoring.Row{
			{
				options.Total.safeApply(ObservationTotal(options.ObservableOptions)(containerName, owner)).Observable(),
				options.Duration.safeApply(ObservationDuration(options.ObservableOptions)(containerName, owner)).Observable(),
				options.Errors.safeApply(ObservationErrors(options.ObservableOptions)(containerName, owner)).Observable(),
			},
		},
	}
}
