package main

import "github.com/prometheus/client_golang/prometheus"

const (
	namespace = "rabbitmq"
)

var (
	queueLabels    = []string{"vhost", "queue", "durable", "policy"}
	exchangeLabels = []string{"vhost", "exchange"}
	nodeLabels     = []string{"vhost", "node"}

	upMetricDescription = newGauge("up", "Was the last scrape of rabbitmq successful.")

	overviewMetricDescription = map[string]prometheus.Gauge{
		"object_totals.channels":    newGauge("channelsTotal", "Total number of open channels."),
		"object_totals.connections": newGauge("connectionsTotal", "Total number of open connections."),
		"object_totals.consumers":   newGauge("consumersTotal", "Total number of message consumers."),
		"object_totals.queues":      newGauge("queuesTotal", "Total number of queues in use."),
		"object_totals.exchanges":   newGauge("exchangesTotal", "Total number of exchanges in use."),
	}

	queueGaugeVec = map[string]*prometheus.GaugeVec{
		"messages_ready":               newGaugeVec("queue_messages_ready", "Number of messages ready to be delivered to clients.", queueLabels),
		"messages_unacknowledged":      newGaugeVec("queue_messages_unacknowledged", "Number of messages delivered to clients but not yet acknowledged.", queueLabels),
		"messages":                     newGaugeVec("queue_messages", "Sum of ready and unacknowledged messages (queue depth).", queueLabels),
		"messages_ready_ram":           newGaugeVec("queue_messages_ready_ram", "Number of messages from messages_ready which are resident in ram.", queueLabels),
		"messages_unacknowledged_ram":  newGaugeVec("queue_messages_unacknowledged_ram", "Number of messages from messages_unacknowledged which are resident in ram.", queueLabels),
		"messages_ram":                 newGaugeVec("queue_messages_ram", "Total number of messages which are resident in ram.", queueLabels),
		"messages_persistent":          newGaugeVec("queue_messages_persistent", "Total number of persistent messages in the queue (will always be 0 for transient queues).", queueLabels),
		"message_bytes":                newGaugeVec("queue_message_bytes", "Sum of the size of all message bodies in the queue. This does not include the message properties (including headers) or any overhead.", queueLabels),
		"message_bytes_ready":          newGaugeVec("queue_message_bytes_ready", "Like message_bytes but counting only those messages ready to be delivered to clients.", queueLabels),
		"message_bytes_unacknowledged": newGaugeVec("queue_message_bytes_unacknowledged", "Like message_bytes but counting only those messages delivered to clients but not yet acknowledged.", queueLabels),
		"message_bytes_ram":            newGaugeVec("queue_message_bytes_ram", "Like message_bytes but counting only those messages which are in RAM.", queueLabels),
		"message_bytes_persistent":     newGaugeVec("queue_message_bytes_persistent", "Like message_bytes but counting only those messages which are persistent.", queueLabels),
		"consumers":                    newGaugeVec("queue_consumers", "Number of consumers.", queueLabels),
		"consumer_utilisation":         newGaugeVec("queue_consumer_utilisation", "Fraction of the time (between 0.0 and 1.0) that the queue is able to immediately deliver messages to consumers. This can be less than 1.0 if consumers are limited by network congestion or prefetch count.", queueLabels),
		"memory":                       newGaugeVec("queue_memory", "Bytes of memory consumed by the Erlang process associated with the queue, including stack, heap and internal structures.", queueLabels),
		"head_message_timestamp":       newGaugeVec("queue_head_message_timestamp", "The timestamp property of the first message in the queue, if present. Timestamps of messages only appear when they are in the paged-in state.", queueLabels), //https://github.com/rabbitmq/rabbitmq-server/pull/54
	}

	queueCounterVec = map[string]*prometheus.Desc{
		"disk_reads":                  newDesc("queue_disk_reads", "Total number of times messages have been read from disk by this queue since it started.", queueLabels),
		"disk_writes":                 newDesc("queue_disk_writes", "Total number of times messages have been written to disk by this queue since it started.", queueLabels),
		"message_stats.publish":       newDesc("queue_messages_published_total", "Count of messages published.", queueLabels),
		"message_stats.confirm":       newDesc("queue_messages_confirmed_total", "Count of messages confirmed. ", queueLabels),
		"message_stats.deliver":       newDesc("queue_messages_delivered_total", "Count of messages delivered in acknowledgement mode to consumers.", queueLabels),
		"message_stats.deliver_noack": newDesc("queue_messages_delivered_noack_total", "Count of messages delivered in no-acknowledgement mode to consumers. ", queueLabels),
		"message_stats.get":           newDesc("queue_messages_get_total", "Count of messages delivered in acknowledgement mode in response to basic.get.", queueLabels),
		"message_stats.get_noack":     newDesc("queue_messages_get_noack_total", "Count of messages delivered in no-acknowledgement mode in response to basic.get.", queueLabels),
		"message_stats.redeliver":     newDesc("queue_messages_redelivered_total", "Count of subset of messages in deliver_get which had the redelivered flag set.", queueLabels),
		"message_stats.return":        newDesc("queue_messages_returned_total", "Count of messages returned to publisher as unroutable.", queueLabels),
		"message_stats.ack":           newDesc("queue_messages_ack_total", "Count of messages delivered in acknowledgement mode in response to basic.get.", queueLabels),
	}

	nodeCounterVec = map[string]*prometheus.Desc{
		"running":         newDesc("running", "test", nodeLabels),
		"mem_used":        newDesc("node_mem_used", "Memory used in bytes", nodeLabels),
		"mem_limit":       newDesc("node_mem_limit", "Point at which the memory alarm will go off", nodeLabels),
		"mem_alarm":       newDesc("node_mem_alarm", "Whether the memory alarm has gone off", nodeLabels),
		"disk_free":       newDesc("node_disk_free", "Disk free space in bytes.", nodeLabels),
		"disk_free_alarm": newDesc("node_disk_free_alarm", "Whether the disk alarm has gone off.", nodeLabels),
		"disk_free_limit": newDesc("node_disk_free_limit", "Point at which the disk alarm will go off.", nodeLabels),
		"fd_used":         newDesc("fd_used", "Used File descriptors", nodeLabels),
		"fd_total":        newDesc("fd_total", "File descriptors available", nodeLabels),
		"sockets_used":    newDesc("sockets_used", "File descriptors used as sockets.", nodeLabels),
		"sockets_total":   newDesc("sockets_total", "File descriptors available for use as sockets", nodeLabels),
	}

	exchangeCounterVec = map[string]*prometheus.Desc{
		"message_stats.publish":           newDesc("exchange_messages_published_total", "Count of messages published.", exchangeLabels),
		"message_stats.publish_in":        newDesc("exchange_messages_published_in_total", "Count of messages published in to an exchange, i.e. not taking account of routing.", exchangeLabels),
		"message_stats.publish_out":       newDesc("exchange_messages_published_out_total", "Count of messages published out of an exchange, i.e. taking account of routing.", exchangeLabels),
		"message_stats.confirm":           newDesc("exchange_messages_confirmed_total", "Count of messages confirmed. ", exchangeLabels),
		"message_stats.deliver":           newDesc("exchange_messages_delivered_total", "Count of messages delivered in acknowledgement mode to consumers.", exchangeLabels),
		"message_stats.deliver_noack":     newDesc("exchange_messages_delivered_noack_total", "Count of messages delivered in no-acknowledgement mode to consumers. ", exchangeLabels),
		"message_stats.get":               newDesc("exchange_messages_get_total", "Count of messages delivered in acknowledgement mode in response to basic.get.", exchangeLabels),
		"message_stats.get_noack":         newDesc("exchange_messages_get_noack_total", "Count of messages delivered in no-acknowledgement mode in response to basic.get.", exchangeLabels),
		"message_stats.ack":               newDesc("exchange_messages_ack_total", "Count of messages delivered in acknowledgement mode in response to basic.get.", exchangeLabels),
		"message_stats.redeliver":         newDesc("exchange_messages_redelivered_total", "Count of subset of messages in deliver_get which had the redelivered flag set.", exchangeLabels),
		"message_stats.return_unroutable": newDesc("exchange_messages_returned_total", "Count of messages returned to publisher as unroutable.", exchangeLabels),
	}
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

func newGauge(metricName string, docString string) prometheus.Gauge {
	return prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      metricName,
			Help:      docString,
		},
	)
}

func newDesc(metricName string, docString string, labels []string) *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", metricName),
		docString,
		labels,
		nil)
}
