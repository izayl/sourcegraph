package definitions

import (
	"github.com/sourcegraph/sourcegraph/monitoring/definitions/shared"
	"github.com/sourcegraph/sourcegraph/monitoring/monitoring"
)

func PreciseCodeIntelWorker() *monitoring.Container {
	const containerName = "precise-code-intel-worker"

	return &monitoring.Container{
		Name:        "precise-code-intel-worker",
		Title:       "Precise Code Intel Worker",
		Description: "Handles conversion of uploaded precise code intelligence bundles.",
		Groups: []monitoring.Group{
			// src_codeintel_upload_total
			// src_codeintel_upload_processor_total
			shared.NewQueueSizeGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.QueueSizeGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "lsif upload",
					MetricName:        "codeintel_upload",
					MetricDescription: "TODO",
				},
			}),

			// src_codeintel_upload_processor_total
			// src_codeintel_upload_processor_duration_seconds_bucket
			// src_codeintel_upload_processor_errors_total
			// src_codeintel_upload_processor_handlers
			shared.NewWorkerutilGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.WorkerutilGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "lsif upload",
					MetricName:        "codeintel_upload",
					MetricDescription: "TODO",
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

			// src_workerutil_dbworker_store_codeintel_upload_total
			// src_workerutil_dbworker_store_codeintel_upload_duration_seconds_bucket
			// src_workerutil_dbworker_store_codeintel_upload_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "dbworker store (polling lsif_uploads)",
					MetricName:        "workerutil_dbworker_store_codeintel_upload",
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

			// src_codeintel_uploadstore_total
			// src_codeintel_uploadstore_duration_seconds_bucket
			// src_codeintel_uploadstore_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "upload store (S3, GCS, or MinIO)",
					MetricName:        "codeintel_uploadstore",
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
