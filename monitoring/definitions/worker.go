package definitions

import (
	"fmt"
	"time"

	"github.com/sourcegraph/sourcegraph/monitoring/definitions/shared"
	"github.com/sourcegraph/sourcegraph/monitoring/monitoring"
)

func Worker() *monitoring.Container {
	const containerName = "worker"

	var workerJobs = []struct {
		Name  string
		Owner monitoring.ObservableOwner
	}{
		{Name: "codeintel-janitor", Owner: monitoring.ObservableOwnerCodeIntel},
		{Name: "codeintel-commitgraph", Owner: monitoring.ObservableOwnerCodeIntel},
		{Name: "codeintel-auto-indexing", Owner: monitoring.ObservableOwnerCodeIntel},
	}

	var activeJobObservables []monitoring.Observable
	for _, job := range workerJobs {
		activeJobObservables = append(activeJobObservables, monitoring.Observable{
			Name:          fmt.Sprintf("worker_job_%s_count", job.Name),
			Description:   fmt.Sprintf("number of worker instances running the %s job", job.Name),
			Query:         fmt.Sprintf(`sum (src_worker_jobs{job="worker", job_name="%s"})`, job.Name),
			Panel:         monitoring.Panel().LegendFormat(fmt.Sprintf("instances running %s", job.Name)),
			DataMustExist: true,
			Warning:       monitoring.Alert().Less(1, nil).For(1 * time.Minute),
			Critical:      monitoring.Alert().Less(1, nil).For(5 * time.Minute),
			Owner:         job.Owner,
			PossibleSolutions: fmt.Sprintf(`
				- Ensure your instance defines a worker container such that:
					- `+"`"+`WORKER_JOB_ALLOWLIST`+"`"+` contains "%[1]s" (or "all"), and
					- `+"`"+`WORKER_JOB_BLOCKLIST`+"`"+` does not contain "%[1]s"
				- Ensure that such a container is not failing to start or stay active
			`, job.Name),
		})
	}

	panelsPerRow := 4
	if rem := len(activeJobObservables) % panelsPerRow; rem == 1 || rem == 2 {
		// If we'd leave one or two panels on the only/last row, then reduce
		// the number of panels in previous rows so that we have less of a width
		// difference at the end
		panelsPerRow = 3
	}

	var activeJobRows []monitoring.Row
	for _, observable := range activeJobObservables {
		if n := len(activeJobRows); n == 0 || len(activeJobRows[n-1]) >= panelsPerRow {
			activeJobRows = append(activeJobRows, nil)
		}

		n := len(activeJobRows)
		activeJobRows[n-1] = append(activeJobRows[n-1], observable)
	}

	activeJobsGroup := monitoring.Group{
		Title: "Active jobs",
		Rows: append(
			[]monitoring.Row{
				{
					{
						Name:        "worker_job_count",
						Description: "number of worker instances running each job",
						Query:       `sum by (job_name) (src_worker_jobs{job="worker"})`,
						Panel:       monitoring.Panel().LegendFormat("instances running {{job_name}}"),
						NoAlert:     true,
						Interpretation: `
						The number of worker instances running each job type.
						It is necessary for each job type to be managed by at least one worker instance.
					`,
					},
				},
			},
			activeJobRows...,
		),
	}

	codeintelJanitorStatsGroup := monitoring.Group{
		Title:  "[codeintel] Janitor stats",
		Hidden: true,
		Rows: []monitoring.Row{
			{
				{
					Name:           "codeintel_upload_records_removed",
					Description:    "upload records expired or deleted every 5m",
					Query:          fmt.Sprintf(`sum(increase(src_codeintel_background_upload_records_removed_total{job=~"%s"}[5m]))`, containerName),
					Panel:          monitoring.Panel().LegendFormat("uploads removed"),
					Owner:          monitoring.ObservableOwnerCodeIntel,
					NoAlert:        true,
					Interpretation: "none",
				},
				{
					Name:           "codeintel_index_records_removed",
					Description:    "index records expired or deleted every 5m",
					Query:          fmt.Sprintf(`sum(increase(src_codeintel_background_index_records_removed_total{job=~"%s"}[5m]))`, containerName),
					Panel:          monitoring.Panel().LegendFormat("indexes removed"),
					Owner:          monitoring.ObservableOwnerCodeIntel,
					NoAlert:        true,
					Interpretation: "none",
				},
				{
					Name:        "codeintel_lsif_data_removed",
					Description: "data for unreferenced upload records removed every 5m",
					Query:       fmt.Sprintf(`sum(increase(src_codeintel_background_uploads_purged_total{job=~"%s"}[5m]))`, containerName),

					Panel:          monitoring.Panel().LegendFormat("uploads purged"),
					Owner:          monitoring.ObservableOwnerCodeIntel,
					NoAlert:        true,
					Interpretation: "none",
				},
				{
					Name:              "codeintel_janitor_errors",
					Description:       "janitor errors every 5m",
					Query:             fmt.Sprintf(`sum(increase(src_codeintel_background_errors_total{job=~"%s"}[5m]))`, containerName),
					Panel:             monitoring.Panel().LegendFormat("errors"),
					Owner:             monitoring.ObservableOwnerCodeIntel,
					Warning:           monitoring.Alert().GreaterOrEqual(20, nil),
					PossibleSolutions: "none",
				},
			},
		},
	}

	return &monitoring.Container{
		Name:        "worker",
		Title:       "Worker",
		Description: "Manages background processes.",
		Groups: []monitoring.Group{
			// src_worker_jobs
			activeJobsGroup,

			// src_codeintel_commit_graph_total
			// src_codeintel_commit_graph_processor_total
			shared.NewQueueSizeGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.QueueSizeGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "repositories with stale commit graph",
					MetricName:        "codeintel_commit_graph",
					MetricDescription: "TODO",
				},
			}),

			// src_codeintel_dependency_index_total
			// src_codeintel_dependency_index_processor_total
			shared.NewQueueSizeGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.QueueSizeGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "dependency indexes",
					MetricName:        "codeintel_dependency_index",
					MetricDescription: "TODO",
				},
			}),

			// src_codeintel_commit_graph_processor_total
			// src_codeintel_commit_graph_processor_duration_seconds_bucket
			// src_codeintel_commit_graph_processor_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "commit graph updater",
					MetricName:        "codeintel_commit_graph_processor",
					MetricDescription: "TODO",
					Hidden:            true,
				},
			}),

			// src_codeintel_dependency_index_processor_total
			// src_codeintel_dependency_index_processor_duration_seconds_bucket
			// src_codeintel_dependency_index_processor_errors_total
			// src_codeintel_dependency_index_processor_handlers
			shared.NewWorkerutilGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.WorkerutilGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "dependency indexes",
					MetricName:        "codeintel_dependency_index",
					MetricDescription: "TODO",
					Hidden:            true,
				},
			}),

			// src_codeintel_background_upload_records_removed_total
			// src_codeintel_background_index_records_removed_total
			// src_codeintel_background_uploads_purged_total
			// src_codeintel_background_errors_total
			codeintelJanitorStatsGroup,

			// src_codeintel_background_upload_resets_total
			// src_codeintel_background_upload_reset_failures_total
			// src_codeintel_background_upload_errors_total
			shared.NewResetterGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ResetterGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "codeintel upload record resetter",
					MetricName:        "codeintel_background_upload",
					MetricDescription: "TODO",
					Hidden:            true,
				},
			}),

			// src_codeintel_background_indexes_resets_total
			// src_codeintel_background_indexes_reset_failures_total
			// src_codeintel_background_indexes_errors_total
			shared.NewResetterGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ResetterGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "codeintel index record resetter",
					MetricName:        "codeintel_background_indexes",
					MetricDescription: "TODO",
					Hidden:            true,
				},
			}),

			// src_codeintel_background_dependency_indexes_resets_total
			// src_codeintel_background_dependency_indexes_reset_failures_total
			// src_codeintel_background_dependency_indexes_errors_total
			shared.NewResetterGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ResetterGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "codeintel dependency index record resetter",
					MetricName:        "codeintel_background_dependency_indexes",
					MetricDescription: "TODO",
					Hidden:            true,
				},
			}),

			// src_codeintel_index_scheduler_total
			// src_codeintel_index_scheduler_duration_seconds_bucket
			// src_codeintel_index_scheduler_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "index scheduler",
					MetricName:        "codeintel_index_scheduler",
					MetricDescription: "TODO",
					Hidden:            true,
				},
			}),

			// src_codeintel_autoindex_enqueuer_total
			// src_codeintel_autoindex_enqueuer_duration_seconds_bucket
			// src_codeintel_autoindex_enqueuer_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "autoindex enqueuer",
					MetricName:        "codeintel_autoindex_enqueuer",
					MetricDescription: "TODO",
					Hidden:            true,
				},
			}),

			// src_codeintel_dbstore_total
			// src_codeintel_dbstore_duration_seconds_bucket
			// src_codeintel_dbstore_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "database",
					MetricName:        "codeintel_dbstore",
					MetricDescription: "TODO",
					Hidden:            true,
				},
			}),

			// src_codeintel_lsifstore_total
			// src_codeintel_lsifstore_duration_seconds_bucket
			// src_codeintel_lsifstore_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "codeintel-db",
					MetricName:        "codeintel_lsifstore",
					MetricDescription: "TODO",
					Hidden:            true,
				},
			}),

			// src_codeintel_gitserver_total
			// src_codeintel_gitserver_duration_seconds_bucket
			// src_codeintel_gitserver_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "gitserver client",
					MetricName:        "codeintel_gitserver",
					MetricDescription: "TODO",
					Hidden:            true,
				},
			}),

			// Resource monitoring
			shared.NewFrontendInternalAPIErrorResponseMonitoringGroup(containerName, monitoring.ObservableOwnerCodeIntel, nil),
			shared.NewDatabaseConnectionsMonitoringGroup(containerName),
			shared.NewContainerMonitoringGroup(containerName, monitoring.ObservableOwnerCodeIntel, nil),
			shared.NewProvisioningIndicatorsGroup(containerName, monitoring.ObservableOwnerCodeIntel, nil),
			shared.NewGolangMonitoringGroup(containerName, monitoring.ObservableOwnerCodeIntel, nil),
			shared.NewKubernetesMonitoringGroup(containerName, monitoring.ObservableOwnerCodeIntel, nil),
		},
	}
}
