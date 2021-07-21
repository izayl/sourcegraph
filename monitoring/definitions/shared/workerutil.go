package shared

import (
	"fmt"

	"github.com/sourcegraph/sourcegraph/monitoring/monitoring"
)

var (
	WorkerutilProcessorTotal observableConstructor = func(options ObservableOptions) sharedObservable {
		options.MetricName = fmt.Sprintf("%s_processor", options.MetricName)
		return ObservationTotal(options)
	}

	WorkerutilProcessorDuration observableConstructor = func(options ObservableOptions) sharedObservable {
		options.MetricName = fmt.Sprintf("%s_processor", options.MetricName)
		return ObservationDuration(options)
	}

	WorkerutilProcessorErrors observableConstructor = func(options ObservableOptions) sharedObservable {
		options.MetricName = fmt.Sprintf("%s_processor", options.MetricName)
		return ObservationDuration(options)
	}

	WorkerutilProcessorHandlers observableConstructor = func(options ObservableOptions) sharedObservable {
		return func(containerName string, owner monitoring.ObservableOwner) Observable {
			filters := makeFilters(containerName, options.Filters...)
			by, legendPrefix := makeBy(options.By...)

			return Observable{
				Name:           fmt.Sprintf("%s_handlers", options.MetricName),
				Description:    fmt.Sprintf("%s active handlers", options.MetricDescription),
				Query:          fmt.Sprintf(`sum%s(src_%s_processor_handlers{%s})`, by, options.MetricName, filters),
				Panel:          monitoring.Panel().LegendFormat(fmt.Sprintf("%shandlers", legendPrefix)),
				Owner:          owner,
				NoAlert:        true,
				Interpretation: "none",
			}
		}
	}
)

type WorkerutilGroupOptions struct {
	ObservableOptions

	// Total transforms the default observable used to construct the processor operation count panel.
	Total ObservableOption

	// Duration transforms the default observable used to construct the processor duration histogram panel.
	Duration ObservableOption

	// Errors transforms the default observable used to construct the processor error rate panel.
	Errors ObservableOption

	// Handlers transforms the default observable used to construct the processor count panel.
	Handlers ObservableOption
}

// NewWorkerutilGroup creates a group containing panels displaying the total number of jobs,
// duration of processing, error rate, and number of workers operating on the queue for the
// given worker observable within the given container.
func NewWorkerutilGroup(containerName string, owner monitoring.ObservableOwner, options WorkerutilGroupOptions) monitoring.Group {
	return monitoring.Group{
		Title:  fmt.Sprintf("[%s] Workerutil processor: %s", options.Namespace, options.GroupDescription),
		Hidden: options.Hidden,
		Rows: []monitoring.Row{
			{
				options.Total.safeApply(WorkerutilProcessorTotal(options.ObservableOptions)(containerName, owner)).Observable(),
				options.Duration.safeApply(WorkerutilProcessorDuration(options.ObservableOptions)(containerName, owner)).Observable(),
				options.Errors.safeApply(WorkerutilProcessorErrors(options.ObservableOptions)(containerName, owner)).Observable(),
				options.Handlers.safeApply(WorkerutilProcessorErrors(options.ObservableOptions)(containerName, owner)).Observable(),
			},
		},
	}
}
