package main

import "github.com/prometheus/client_golang/prometheus"

const (
	namespace = "rabbitmq"
)

func newGaugeVec(metricName string, docString string, labels []string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      metricName,
			Help:      docString,
		},
		labels,
	)
}

func newDesc(metricName string, docString string, labels []string) *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", metricName),
		docString,
		labels,
		nil)
}
