package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	RegisterExporter("node", newExporterNode)
}

var (
	nodeLabels    = []string{"node"}
	nodeLabelKeys = []string{"name"}

	nodeGaugeVec = map[string]*prometheus.GaugeVec{
		"running":         newGaugeVec("running", "number of running nodes", nodeLabels),
		"mem_used":        newGaugeVec("node_mem_used", "Memory used in bytes", nodeLabels),
		"mem_limit":       newGaugeVec("node_mem_limit", "Point at which the memory alarm will go off", nodeLabels),
		"mem_alarm":       newGaugeVec("node_mem_alarm", "Whether the memory alarm has gone off", nodeLabels),
		"disk_free":       newGaugeVec("node_disk_free", "Disk free space in bytes.", nodeLabels),
		"disk_free_alarm": newGaugeVec("node_disk_free_alarm", "Whether the disk alarm has gone off.", nodeLabels),
		"disk_free_limit": newGaugeVec("node_disk_free_limit", "Point at which the disk alarm will go off.", nodeLabels),
		"fd_used":         newGaugeVec("fd_used", "Used File descriptors", nodeLabels),
		"fd_total":        newGaugeVec("fd_total", "File descriptors available", nodeLabels),
		"sockets_used":    newGaugeVec("sockets_used", "File descriptors used as sockets.", nodeLabels),
		"sockets_total":   newGaugeVec("sockets_total", "File descriptors available for use as sockets", nodeLabels),
		"partitions_len":  newGaugeVec("partitions", "Current Number of network partitions. 0 is ok. If the cluster is splitted the value is at least 2", nodeLabels),
	}
)

type exporterNode struct {
	nodeMetricsGauge map[string]*prometheus.GaugeVec
}

func newExporterNode() Exporter {
	return exporterNode{
		nodeMetricsGauge: nodeGaugeVec,
	}
}

func (e exporterNode) String() string {
	return "Exporter node"
}

func (e exporterNode) Collect(ch chan<- prometheus.Metric) error {
	nodeData, err := getStatsInfo(config, "nodes", nodeLabelKeys)

	if err != nil {
		return err
	}

	for key, gauge := range e.nodeMetricsGauge {
		for _, node := range nodeData {
			if value, ok := node.metrics[key]; ok {
				gauge.WithLabelValues(node.labels["name"]).Set(value)
			}
		}
	}

	for _, gauge := range e.nodeMetricsGauge {
		gauge.Collect(ch)
	}
	return nil
}

func (e exporterNode) Describe(ch chan<- *prometheus.Desc) {
	for _, nodeMetric := range e.nodeMetricsGauge {
		nodeMetric.Describe(ch)
	}

}
