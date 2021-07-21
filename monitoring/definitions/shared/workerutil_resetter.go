package shared

import (
	"fmt"

	"github.com/sourcegraph/sourcegraph/monitoring/monitoring"
)

var (
	ResetterRecordResets observableConstructor = func(options ObservableOptions) sharedObservable {
		return func(containerName string, owner monitoring.ObservableOwner) Observable {
			filters := makeFilters(containerName, options.Filters...)
			by, legendPrefix := makeBy(options.By...)

			return Observable{
				Name:           fmt.Sprintf("%s_record_resets", options.MetricName),
				Description:    fmt.Sprintf("%s records reset to queued state every 5m", options.MetricDescription),
				Query:          fmt.Sprintf(`sum%s(increase(src_%s_resets_total{%s}[5m]))`, by, options.MetricName, filters),
				Panel:          monitoring.Panel().LegendFormat(fmt.Sprintf("%srecords", legendPrefix)),
				Owner:          owner,
				NoAlert:        true,
				Interpretation: "none",
			}
		}
	}

	ResetterRecordResetFailures observableConstructor = func(options ObservableOptions) sharedObservable {
		return func(containerName string, owner monitoring.ObservableOwner) Observable {
			filters := makeFilters(containerName, options.Filters...)
			by, legendPrefix := makeBy(options.By...)

			return Observable{
				Name:           fmt.Sprintf("%s_record_reset_failures", options.MetricName),
				Description:    fmt.Sprintf("%s records reset to errored state every 5m", options.MetricDescription),
				Query:          fmt.Sprintf(`sum%s(increase(src_%s_reset_failures_total{%s}[5m]))`, by, options.MetricName, filters),
				Panel:          monitoring.Panel().LegendFormat(fmt.Sprintf("%srecords", legendPrefix)),
				Owner:          owner,
				NoAlert:        true,
				Interpretation: "none",
			}
		}
	}
)

type ResetterGroupOptions struct {
	ObservableOptions

	// Total transforms the default observable used to construct the reset count panel.
	RecordResets ObservableOption

	// Duration transforms the default observable used to construct the reset failure count panel.
	RecordResetFailures ObservableOption

	// Errors transforms the default observable used to construct the resetter error rate panel.
	Errors ObservableOption
}

// NewResetterGroup creates a group containing panels displaying the total number of records
// reset, the number of records moved to errored, and the error rate of the resetter operating
// within the given container.
func NewResetterGroup(containerName string, owner monitoring.ObservableOwner, options ResetterGroupOptions) monitoring.Group {
	return monitoring.Group{
		Title:  fmt.Sprintf("[%s] Workerutil resetter: %s", options.Namespace, options.GroupDescription),
		Hidden: options.Hidden,
		Rows: []monitoring.Row{
			{
				options.RecordResets.safeApply(ResetterRecordResets(options.ObservableOptions)(containerName, owner)).Observable(),
				options.RecordResetFailures.safeApply(ResetterRecordResetFailures(options.ObservableOptions)(containerName, owner)).Observable(),
				options.Errors.safeApply(ObservationErrors(options.ObservableOptions)(containerName, owner)).Observable(),
			},
		},
	}
}
