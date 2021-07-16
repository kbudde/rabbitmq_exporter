package main

import (
	"context"
	"errors"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

func init() {
	RegisterExporter("queue", newExporterQueue)
}

var (
	queueLabels    = []string{"cluster", "vhost", "queue", "durable", "policy", "self"}
	queueLabelKeys = []string{"vhost", "name", "durable", "policy", "state", "node", "idle_since"}

	queueGaugeVec = map[string]*prometheus.GaugeVec{
		"messages_ready":                            newGaugeVec("queue_messages_ready", "Number of messages ready to be delivered to clients.", queueLabels),
		"messages_unacknowledged":                   newGaugeVec("queue_messages_unacknowledged", "Number of messages delivered to clients but not yet acknowledged.", queueLabels),
		"messages":                                  newGaugeVec("queue_messages", "Sum of ready and unacknowledged messages (queue depth).", queueLabels),
		"messages_ready_ram":                        newGaugeVec("queue_messages_ready_ram", "Number of messages from messages_ready which are resident in ram.", queueLabels),
		"messages_unacknowledged_ram":               newGaugeVec("queue_messages_unacknowledged_ram", "Number of messages from messages_unacknowledged which are resident in ram.", queueLabels),
		"messages_ram":                              newGaugeVec("queue_messages_ram", "Total number of messages which are resident in ram.", queueLabels),
		"messages_persistent":                       newGaugeVec("queue_messages_persistent", "Total number of persistent messages in the queue (will always be 0 for transient queues).", queueLabels),
		"message_bytes":                             newGaugeVec("queue_message_bytes", "Sum of the size of all message bodies in the queue. This does not include the message properties (including headers) or any overhead.", queueLabels),
		"message_bytes_ready":                       newGaugeVec("queue_message_bytes_ready", "Like message_bytes but counting only those messages ready to be delivered to clients.", queueLabels),
		"message_bytes_unacknowledged":              newGaugeVec("queue_message_bytes_unacknowledged", "Like message_bytes but counting only those messages delivered to clients but not yet acknowledged.", queueLabels),
		"message_bytes_ram":                         newGaugeVec("queue_message_bytes_ram", "Like message_bytes but counting only those messages which are in RAM.", queueLabels),
		"message_bytes_persistent":                  newGaugeVec("queue_message_bytes_persistent", "Like message_bytes but counting only those messages which are persistent.", queueLabels),
		"consumers":                                 newGaugeVec("queue_consumers", "Number of consumers.", queueLabels),
		"consumer_utilisation":                      newGaugeVec("queue_consumer_utilisation", "Fraction of the time (between 0.0 and 1.0) that the queue is able to immediately deliver messages to consumers. This can be less than 1.0 if consumers are limited by network congestion or prefetch count.", queueLabels),
		"memory":                                    newGaugeVec("queue_memory", "Bytes of memory consumed by the Erlang process associated with the queue, including stack, heap and internal structures.", queueLabels),
		"head_message_timestamp":                    newGaugeVec("queue_head_message_timestamp", "The timestamp property of the first message in the queue, if present. Timestamps of messages only appear when they are in the paged-in state.", queueLabels), //https://github.com/rabbitmq/rabbitmq-server/pull/54
		"arguments.x-max-length-bytes":              newGaugeVec("queue_max_length_bytes", "Total body size for ready messages a queue can contain before it starts to drop them from its head.", queueLabels),
		"arguments.x-max-length":                    newGaugeVec("queue_max_length", "How many (ready) messages a queue can contain before it starts to drop them from its head.", queueLabels),
		"garbage_collection.min_heap_size":          newGaugeVec("queue_gc_min_heap", "Minimum heap size in words", queueLabels),
		"garbage_collection.min_bin_vheap_size":     newGaugeVec("queue_gc_min_vheap", "Minimum binary virtual heap size in words", queueLabels),
		"garbage_collection.fullsweep_after":        newGaugeVec("queue_gc_collections_before_fullsweep", "Maximum generational collections before fullsweep", queueLabels),
		"slave_nodes_len":                           newGaugeVec("queue_slaves_nodes_len", "Number of slave nodes attached to the queue", queueLabels),
		"synchronised_slave_nodes_len":              newGaugeVec("queue_synchronised_slave_nodes_len", "Number of slave nodes in sync to the queue", queueLabels),
		"message_stats.publish_details.rate":        newGaugeVec("queue_messages_publish_rate", "Rate at which messages are entering the server.", queueLabels),
		"message_stats.deliver_no_ack_details.rate": newGaugeVec("queue_messages_deliver_no_ack_rate", "Rate at which messages are delivered to consumers that use automatic acknowledgements.", queueLabels),
		"message_stats.deliver_details.rate":        newGaugeVec("queue_messages_deliver_rate", "Rate at which messages are delivered to consumers that use manual acknowledgements.", queueLabels),
	}

	queueCounterVec = map[string]*prometheus.Desc{
		"disk_reads":                   newDesc("queue_disk_reads_total", "Total number of times messages have been read from disk by this queue since it started.", queueLabels),
		"disk_writes":                  newDesc("queue_disk_writes_total", "Total number of times messages have been written to disk by this queue since it started.", queueLabels),
		"message_stats.publish":        newDesc("queue_messages_published_total", "Count of messages published.", queueLabels),
		"message_stats.confirm":        newDesc("queue_messages_confirmed_total", "Count of messages confirmed. ", queueLabels),
		"message_stats.deliver":        newDesc("queue_messages_delivered_total", "Count of messages delivered in acknowledgement mode to consumers.", queueLabels),
		"message_stats.deliver_no_ack": newDesc("queue_messages_delivered_noack_total", "Count of messages delivered in no-acknowledgement mode to consumers. ", queueLabels),
		"message_stats.get":            newDesc("queue_messages_get_total", "Count of messages delivered in acknowledgement mode in response to basic.get.", queueLabels),
		"message_stats.get_no_ack":     newDesc("queue_messages_get_noack_total", "Count of messages delivered in no-acknowledgement mode in response to basic.get.", queueLabels),
		"message_stats.redeliver":      newDesc("queue_messages_redelivered_total", "Count of subset of messages in deliver_get which had the redelivered flag set.", queueLabels),
		"message_stats.return":         newDesc("queue_messages_returned_total", "Count of messages returned to publisher as unroutable.", queueLabels),
		"message_stats.ack":            newDesc("queue_messages_ack_total", "Count of messages delivered in acknowledgement mode in response to basic.get.", queueLabels),
		"reductions":                   newDesc("queue_reductions_total", "Count of  reductions which take place on this process. .", queueLabels),
		"garbage_collection.minor_gcs": newDesc("queue_gc_minor_collections_total", "Number of minor GCs", queueLabels),
	}
)

type exporterQueue struct {
	queueMetricsGauge   map[string]*prometheus.GaugeVec
	queueMetricsCounter map[string]*prometheus.Desc
	stateMetric         *prometheus.GaugeVec
	idleSinceMetric     *prometheus.GaugeVec
}

func newExporterQueue() Exporter {
	queueGaugeVecActual := queueGaugeVec
	queueCounterVecActual := queueCounterVec

	if len(config.ExcludeMetrics) > 0 {
		for _, metric := range config.ExcludeMetrics {
			if queueGaugeVecActual[metric] != nil {
				delete(queueGaugeVecActual, metric)
			}
			if queueCounterVecActual[metric] != nil {
				delete(queueCounterVecActual, metric)
			}
		}
	}

	return exporterQueue{
		queueMetricsGauge:   queueGaugeVecActual,
		queueMetricsCounter: queueCounterVecActual,
		stateMetric:         newGaugeVec("queue_state", "A metric with a value of constant '1' if the queue is in a certain state", append(queueLabels, "state")),
		idleSinceMetric:     newGaugeVec("queue_idle_since_seconds", "starttime where the queue switched to idle state; in seconds since epoch (1970).", queueLabels),
	}
}

func (e exporterQueue) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	for _, gaugevec := range e.queueMetricsGauge {
		gaugevec.Reset()
	}
	e.stateMetric.Reset()
	e.idleSinceMetric.Reset()

	if config.MaxQueues > 0 {
		// Get overview info to check total queues
		totalQueues, ok := ctx.Value(totalQueues).(int)
		if !ok {
			return errors.New("total Queue counter missing")
		}

		if totalQueues > config.MaxQueues {
			log.WithFields(log.Fields{
				"MaxQueues":   config.MaxQueues,
				"TotalQueues": totalQueues,
			}).Debug("MaxQueues exceeded.")
			return nil
		}
	}

	selfNode := ""
	if n, ok := ctx.Value(nodeName).(string); ok {
		selfNode = n
	}
	cluster := ""
	if n, ok := ctx.Value(clusterName).(string); ok {
		cluster = n
	}

	rabbitMqQueueData, err := getStatsInfo(config, "queues", queueLabelKeys)

	if err != nil {
		return err
	}

	log.WithField("queueData", rabbitMqQueueData).Debug("Queue data")
	for key, gaugevec := range e.queueMetricsGauge {
		for _, queue := range rabbitMqQueueData {
			qname := queue.labels["name"]
			vname := queue.labels["vhost"]
			if value, ok := queue.metrics[key]; ok {

				if matchVhost := config.IncludeVHost.MatchString(vname); matchVhost {
					if skipVhost := config.SkipVHost.MatchString(vname); !skipVhost {
						if matchInclude := config.IncludeQueues.MatchString(qname); matchInclude {
							if matchSkip := config.SkipQueues.MatchString(qname); !matchSkip {
								self := "0"
								if queue.labels["node"] == selfNode {
									self = "1"
								}
								// log.WithFields(log.Fields{"vhost": queue.labels["vhost"], "queue": queue.labels["name"], "key": key, "value": value}).Info("Set queue metric for key")
								gaugevec.WithLabelValues(cluster, queue.labels["vhost"], queue.labels["name"], queue.labels["durable"], queue.labels["policy"], self).Set(value)
							}
						}
					}
				}
			}
		}
	}

	for _, queue := range rabbitMqQueueData {
		qname := queue.labels["name"]
		vname := queue.labels["vhost"]

		if vhostIncluded := config.IncludeVHost.MatchString(vname); !vhostIncluded {
			continue
		}
		if skipVhost := config.SkipVHost.MatchString(vname); skipVhost {
			continue
		}
		if queueIncluded := config.IncludeQueues.MatchString(qname); !queueIncluded {
			continue
		}
		if queueSkipped := config.SkipQueues.MatchString(qname); queueSkipped {
			continue
		}

		self := "0"
		if queue.labels["node"] == selfNode {
			self = "1"
		}

		idleSince, exists := queue.labels["idle_since"]
		if exists && idleSince != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", idleSince); err == nil {
				unixSeconds := float64(t.UnixNano()) / 1e9
				state := queue.labels["state"]
				if state == "running" { //replace running state with idle if idle_since time is provided. Other states (flow, etc.) are not replaced
					state = "idle"
				}
				e.idleSinceMetric.WithLabelValues(cluster, queue.labels["vhost"], queue.labels["name"], queue.labels["durable"], queue.labels["policy"], self).Set(unixSeconds)
				e.stateMetric.WithLabelValues(cluster, queue.labels["vhost"], queue.labels["name"], queue.labels["durable"], queue.labels["policy"], self, state).Set(1)
			} else {
				log.WithError(err).WithField("idle_since", idleSince).Warn("error parsing idle since time")
			}
		} else {
			e.stateMetric.WithLabelValues(cluster, queue.labels["vhost"], queue.labels["name"], queue.labels["durable"], queue.labels["policy"], self, queue.labels["state"]).Set(1)
		}
	}

	for key, countvec := range e.queueMetricsCounter {
		for _, queue := range rabbitMqQueueData {
			qname := queue.labels["name"]
			vname := queue.labels["vhost"]

			if matchVhost := config.IncludeVHost.MatchString(vname); matchVhost {
				if skipVhost := config.SkipVHost.MatchString(vname); !skipVhost {
					if matchInclude := config.IncludeQueues.MatchString(qname); matchInclude {
						if matchSkip := config.SkipQueues.MatchString(qname); !matchSkip {
							self := "0"
							if queue.labels["node"] == selfNode {
								self = "1"
							}
							if value, ok := queue.metrics[key]; ok {
								ch <- prometheus.MustNewConstMetric(countvec, prometheus.CounterValue, value, cluster, queue.labels["vhost"], queue.labels["name"], queue.labels["durable"], queue.labels["policy"], self)
							} else {
								ch <- prometheus.MustNewConstMetric(countvec, prometheus.CounterValue, 0, cluster, queue.labels["vhost"], queue.labels["name"], queue.labels["durable"], queue.labels["policy"], self)
							}
						}
					}
				}
			}
		}
	}

	for _, gaugevec := range e.queueMetricsGauge {
		gaugevec.Collect(ch)
	}
	e.stateMetric.Collect(ch)
	e.idleSinceMetric.Collect(ch)

	return nil
}

func (e exporterQueue) Describe(ch chan<- *prometheus.Desc) {
	for _, gaugevec := range e.queueMetricsGauge {
		gaugevec.Describe(ch)
	}
	e.stateMetric.Describe(ch)
	e.idleSinceMetric.Describe(ch)
	for _, countervec := range e.queueMetricsCounter {
		ch <- countervec
	}
}
