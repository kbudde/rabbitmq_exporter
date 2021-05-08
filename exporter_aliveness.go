package main

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

func init() {
	RegisterExporter("aliveness", newExporterAliveness)
}

var (
	alivenessLabels = []string{"vhost"}

	alivenessGaugeVec = map[string]*prometheus.GaugeVec{
		"vhost.aliveness": newGaugeVec("aliveness_test", "vhost aliveness test", alivenessLabels),
	}

	rabbitmqAlivenessMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rabbitmq_aliveness_info",
			Help: "A metric with value 1 status:ok else 0 labeled by aliveness test status, error, reason",
		},
		[]string{"status", "error", "reason"},
	)
)

type exporterAliveness struct {
	alivenessMetrics map[string]*prometheus.GaugeVec
	alivenessInfo    AlivenessInfo
}

type AlivenessInfo struct {
	Status string
	Error  string
	Reason string
}

func newExporterAliveness() Exporter {
	alivenessGaugeVecActual := alivenessGaugeVec

	if len(config.ExcludeMetrics) > 0 {
		for _, metric := range config.ExcludeMetrics {
			if alivenessGaugeVecActual[metric] != nil {
				delete(alivenessGaugeVecActual, metric)
			}
		}
	}

	return &exporterAliveness{
		alivenessMetrics: alivenessGaugeVecActual,
		alivenessInfo:    AlivenessInfo{},
	}
}

func (e *exporterAliveness) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	body, contentType, err := apiRequest(config, "aliveness-test")
	if err != nil {
		return err
	}

	reply, err := MakeReply(contentType, body)
	if err != nil {
		return err
	}

	rabbitMqAlivenessData := reply.MakeMap()

	e.alivenessInfo.Status, _ = reply.GetString("status")
	e.alivenessInfo.Error, _ = reply.GetString("error")
	e.alivenessInfo.Reason, _ = reply.GetString("reason")

	rabbitmqAlivenessMetric.Reset()
	var flag float64 = 0
	if e.alivenessInfo.Status == "ok" {
		flag = 1
	}
	rabbitmqAlivenessMetric.WithLabelValues(e.alivenessInfo.Status, e.alivenessInfo.Error, e.alivenessInfo.Reason).Set(flag)

	log.WithField("alivenesswData", rabbitMqAlivenessData).Debug("Aliveness data")
	for key, gauge := range e.alivenessMetrics {
		if value, ok := rabbitMqAlivenessData[key]; ok {
			log.WithFields(log.Fields{"key": key, "value": value}).Debug("Set aliveness metric for key")
			gauge.WithLabelValues(e.alivenessInfo.Status).Set(value)
		}
	}

	if ch != nil {
		rabbitmqAlivenessMetric.Collect(ch)
		for _, gauge := range e.alivenessMetrics {
			gauge.Collect(ch)
		}
	}
	return nil
}

func (e exporterAliveness) Describe(ch chan<- *prometheus.Desc) {
	rabbitmqVersionMetric.Describe(ch)
	for _, gauge := range e.alivenessMetrics {
		gauge.Describe(ch)
	}
}
