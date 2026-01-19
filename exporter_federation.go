package main

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	RegisterExporter("federation", newExporterFederation)
}

var (
	federationLabels     = []string{"cluster", "vhost", "node", "queue", "exchange", "self", "status", "upstream", "upstream_exchange", "upstream_queue", "uri"}
	federationLabelsKeys = []string{"vhost", "status", "node", "queue", "exchange", "upstream", "upstream_exchange", "upstream_queue", "uri"}
)

type exporterFederation struct {
	stateMetric *prometheus.GaugeVec
}

func newExporterFederation() Exporter {
	return exporterFederation{
		stateMetric: newGaugeVec("federation_state", "A metric with a value of constant '1' for each federation in a certain state", federationLabels),
	}
}

func (e exporterFederation) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	e.stateMetric.Reset()

	federationData, err := getStatsInfo(config, "federation-links", federationLabelsKeys)
	if err != nil {
		return err
	}

	cluster := ""
	if n, ok := ctx.Value(clusterName).(string); ok {
		cluster = n
	}
	selfNode := ""
	if n, ok := ctx.Value(nodeName).(string); ok {
		selfNode = n
	}

	for _, federation := range federationData {
		self := selfLabel(config, federation.labels["node"] == selfNode)
		e.stateMetric.WithLabelValues(cluster, federation.labels["vhost"], federation.labels["node"], federation.labels["queue"], federation.labels["exchange"], self, federation.labels["status"], federation.labels["upstream"], federation.labels["upstream_exchange"], federation.labels["upstream_queue"], federation.labels["uri"]).Set(1)
	}

	e.stateMetric.Collect(ch)
	return nil
}

func (e exporterFederation) Describe(ch chan<- *prometheus.Desc) {
	e.stateMetric.Describe(ch)
}
