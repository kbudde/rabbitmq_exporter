package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	RegisterExporter("connections", newExporterConnections)
}

var (
	connectionLabels = []string{"vhost", "node", "peer_host", "user"}

	connectionGaugeVec = map[string]*prometheus.GaugeVec{
		"channels":  newGaugeVec("connection_channels", "number of channels in use", connectionLabels),
		"recv_oct":  newGaugeVec("connection_received_bytes", "received bytes", connectionLabels),
		"recv_cnt":  newGaugeVec("connection_received_packets", "received packets", connectionLabels),
		"send_oct":  newGaugeVec("connection_send_bytes", "send bytes", connectionLabels),
		"send_cnt":  newGaugeVec("connection_send_packets", "send packets", connectionLabels),
		"send_pend": newGaugeVec("connection_send_pending", "Send queue size", connectionLabels),
	}
)

type exporterConnections struct {
	metricsGV map[string]*prometheus.GaugeVec
}

func newExporterConnections() Exporter {
	return exporterConnections{
		metricsGV: connectionGaugeVec,
	}
}

func (e exporterConnections) String() string {
	return "Exporter connections"
}

func (e exporterConnections) Collect(ch chan<- prometheus.Metric) error {
	connectionData, err := getStatsInfo(config, "connections", connectionLabels)

	if err != nil {
		return err
	}
	for _, gauge := range e.metricsGV {
		gauge.Reset()
	}

	for key, gauge := range e.metricsGV {
		for _, connD := range connectionData {
			if value, ok := connD.metrics[key]; ok {
				gauge.WithLabelValues(connD.labels["vhost"], connD.labels["node"], connD.labels["peer_host"], connD.labels["user"]).Add(value)
			}
		}
	}

	for _, gauge := range e.metricsGV {
		gauge.Collect(ch)
	}
	return nil
}

func (e exporterConnections) Describe(ch chan<- *prometheus.Desc) {
	for _, nodeMetric := range e.metricsGV {
		nodeMetric.Describe(ch)
	}

}
