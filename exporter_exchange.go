package main

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	RegisterExporter("exchange", newExporterExchange)
}

var (
	exchangeLabels    = []string{"cluster", "vhost", "exchange"}
	exchangeLabelKeys = []string{"vhost", "name"}

	exchangeCounterVec = map[string]*prometheus.Desc{
		"message_stats.publish":           newDesc("exchange_messages_published_total", "Count of messages published.", exchangeLabels),
		"message_stats.publish_in":        newDesc("exchange_messages_published_in_total", "Count of messages published in to an exchange, i.e. not taking account of routing.", exchangeLabels),
		"message_stats.publish_out":       newDesc("exchange_messages_published_out_total", "Count of messages published out of an exchange, i.e. taking account of routing.", exchangeLabels),
		"message_stats.confirm":           newDesc("exchange_messages_confirmed_total", "Count of messages confirmed. ", exchangeLabels),
		"message_stats.deliver":           newDesc("exchange_messages_delivered_total", "Count of messages delivered in acknowledgement mode to consumers.", exchangeLabels),
		"message_stats.deliver_no_ack":    newDesc("exchange_messages_delivered_noack_total", "Count of messages delivered in no-acknowledgement mode to consumers. ", exchangeLabels),
		"message_stats.get":               newDesc("exchange_messages_get_total", "Count of messages delivered in acknowledgement mode in response to basic.get.", exchangeLabels),
		"message_stats.get_no_ack":        newDesc("exchange_messages_get_noack_total", "Count of messages delivered in no-acknowledgement mode in response to basic.get.", exchangeLabels),
		"message_stats.ack":               newDesc("exchange_messages_ack_total", "Count of messages delivered in acknowledgement mode in response to basic.get.", exchangeLabels),
		"message_stats.redeliver":         newDesc("exchange_messages_redelivered_total", "Count of subset of messages in deliver_get which had the redelivered flag set.", exchangeLabels),
		"message_stats.return_unroutable": newDesc("exchange_messages_returned_total", "Count of messages returned to publisher as unroutable.", exchangeLabels),
	}
)

type exporterExchange struct {
	exchangeMetrics map[string]*prometheus.Desc
}

func newExporterExchange() Exporter {
	exchangeCounterVecActual := exchangeCounterVec

	if len(config.ExcludeMetrics) > 0 {
		for _, metric := range config.ExcludeMetrics {
			if exchangeCounterVecActual[metric] != nil {
				delete(exchangeCounterVecActual, metric)
			}
		}
	}

	return exporterExchange{
		exchangeMetrics: exchangeCounterVecActual,
	}
}

func (e exporterExchange) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	exchangeData, err := getStatsInfo(config, "exchanges", exchangeLabelKeys)

	if err != nil {
		return err
	}
	cluster := ""
	if n, ok := ctx.Value(clusterName).(string); ok {
		cluster = n
	}

	for key, countvec := range e.exchangeMetrics {
		for _, exchange := range exchangeData {
			ename := exchange.labels["name"]
			vname := exchange.labels["vhost"]
			if vhostIncluded := config.IncludeVHost.MatchString(vname); !vhostIncluded {
				continue
			}
			if skipVhost := config.SkipVHost.MatchString(vname); skipVhost {
				continue
			}
			if exchangeIncluded := config.IncludeExchanges.MatchString(ename); !exchangeIncluded {
				continue
			}
			if exchangeSkipped := config.SkipExchanges.MatchString(ename); exchangeSkipped {
				continue
			}
			if value, ok := exchange.metrics[key]; ok {
				ch <- prometheus.MustNewConstMetric(countvec, prometheus.CounterValue, value, cluster, exchange.labels["vhost"], exchange.labels["name"])
			}
		}
	}

	return nil
}

func (e exporterExchange) Describe(ch chan<- *prometheus.Desc) {
	for _, exchangeMetric := range e.exchangeMetrics {
		ch <- exchangeMetric
	}
}
