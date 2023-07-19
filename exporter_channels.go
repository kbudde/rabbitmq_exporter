package main

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	RegisterExporter("channels", newExporterChannels)
}

var (
	channelLabelsStateMetric = []string{"cluster", "vhost", "node", "name", "user", "state", "confirm", "transactional", "self"}
	channelLabelKeys         = []string{"vhost", "node", "name", "user", "state", "confirm", "transactional", "node"}
)

type exporterChannels struct {
	stateMetric *prometheus.GaugeVec
}

func newExporterChannels() Exporter {
	return exporterChannels{
		stateMetric: newGaugeVec("channel_status", "Number of channels in a certain state aggregated per label combination.", channelLabelsStateMetric),
	}
}

func (e exporterChannels) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	rabbitChannelResponses, err := getStatsInfo(config, "channels", channelLabelKeys)

	if err != nil {
		return err
	}

	e.stateMetric.Reset()

	selfNode := ""
	if n, ok := ctx.Value(nodeName).(string); ok {
		selfNode = n
	}
	cluster := ""
	if n, ok := ctx.Value(clusterName).(string); ok {
		cluster = n
	}

	for _, connD := range rabbitChannelResponses {
		self := selfLabel(config, connD.labels["node"] == selfNode)
		e.stateMetric.WithLabelValues(
			cluster,
			connD.labels["vhost"],
			connD.labels["node"],
			connD.labels["name"],
			connD.labels["user"],
			connD.labels["state"],
			connD.labels["confirm"],
			connD.labels["transactional"],
			self).Add(1)
	}

	e.stateMetric.Collect(ch)
	return nil
}

func (e exporterChannels) Describe(ch chan<- *prometheus.Desc) {
	e.stateMetric.Describe(ch)
}
