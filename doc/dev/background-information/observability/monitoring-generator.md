# Sourcegraph monitoring generator

<p class="lead">
The monitoring generator manages converting monitoring definitions into integrations with Sourcegraph's monitoring ecosystem.
</p>

Its purpose is to help enable a [cohesive observability experience for site administrators](../../../admin/observability/index.md), codify [Sourcegraph's monitoring pillars](https://about.sourcegraph.com/handbook/engineering/observability/monitoring_pillars), and make it easy for [developers to add monitoring for their Sourcegraph services](../../how-to/add_monitoring.md) by generating integrations with Sourcegraph's monitoring ecosystem for free.

## Reference

- [Usage and development](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/monitoring/README.md) for developing the generator itself
- [Monitoring API](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/monitoring/monitoring/README.md) for interacting with the generator library
- [How to add monitoring definitions](../../how-to/add_monitoring.md) for developers looking to add monitoring for their services

## Features

### Documentation generation

The generator automatically creates documentation from monitoring definitions that customers and engineers can reference.
These include:

- [Alert solutions reference](https://docs.sourcegraph.com/admin/observability/alert_solutions)
- [Dashboards reference](https://docs.sourcegraph.com/admin/observability/dashboards)

Links to generated documentation can be provided in our other generated integrations - for example, [Slack alerts](https://docs.sourcegraph.com/admin/observability/alerting#setting-up-alerting) will provide a link to the appropriate alert solutions entry, and [Grafana panels](#grafana-integration) will link to the appropriate dashboards reference entry.

### Grafana integration

The generator automatically generates and ships dashboards from monitoring definitions within the [Sourcegraph Grafana distribution](https://about.sourcegraph.com/handbook/engineering/observability/monitoring_architecture#sourcegraph-grafana).

It also takes care of the following:

- Graphs within rows are sized appropriately
- Alerts visualization through the [`ObservableAlertDefinition` API](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/monitoring/monitoring/README.md#type-observablealertdefinition):
  - Overview graphs for alerts (both Sourcegraph-wide and per-service)
  - Threshold lines for alerts of all levels are rendered in graphs
- Formatting of units, labels, and more (using either the defaults, or the [`ObservablePanelOptions` API](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/monitoring/monitoring/README.md#type-observablepaneloptions))
- Maintaining a uniform look and feel across all dashboards
- Providing links to [generated documentation](#documentation-generation)

Links to generated documentation can be provided in our other generated integrations - for example, [Slack alerts](https://docs.sourcegraph.com/admin/observability/alerting#setting-up-alerting) will provide a link to the appropriate service's dashboard.

### Prometheus integration

The generator automatically generates and ships Prometheus recording rules and alerts within the [Sourcegraph Prometheus distribution](https://about.sourcegraph.com/handbook/engineering/observability/monitoring_architecture#sourcegraph-prometheus).
This include the following, all with appropriate and consistent labels:

- [`alert_count` recording rules](https://about.sourcegraph.com/handbook/engineering/observability/monitoring_architecture#alert-count-metrics)
- Native Prometheus alerts, leveraged by our [Alertmanager integration](#alertmanager-integration)

Generated Prometheus recording rules are leveraged by the [Grafana integration](#grafana-integration).

### Alertmanager integration

The generator's [Prometheus integration](#prometheus-integration) is a critical part of the [Sourcegraph's alerting capabilities](https://about.sourcegraph.com/handbook/engineering/observability/monitoring_architecture#alert-notifications), which handles alert routing by level and formatting of alert messages to include links to [documentation](#documentation-generation) and [dashboards](#grafana-integration).
Learn more about using Sourcegraph alerting in the [alerting documentation](https://docs.sourcegraph.com/admin/observability/alerting).
This is possible due to the labels generated by the [Prometheus integration](#prometheus-integration).

At Sourcegraph, extended routing based on team ownership (as defined by [`ObservableOwner`](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/monitoring/monitoring/README.md#type-observableowner)) is also used to route customer support requests and [on-call events through OpsGenie](https://about.sourcegraph.com/handbook/engineering/incidents/on_call).