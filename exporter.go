package main

import (
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"
	"regexp"
	"strings"
)

type exporter struct {
	mutex               sync.RWMutex
	queueMetricsGauge   map[string]*prometheus.GaugeVec
	queueMetricsCounter map[string]*prometheus.Desc
	overviewMetrics     map[string]prometheus.Gauge
	upMetric            prometheus.Gauge
	exchangeMetrics     map[string]*prometheus.Desc
	overviewFetched     bool
	queuesFetched       bool
	exchangesFetched    bool
}

func newExporter() *exporter {
	return &exporter{
		queueMetricsGauge:   queueGaugeVec,
		queueMetricsCounter: queueCounterVec,
		overviewMetrics:     overviewMetricDescription,
		upMetric:            upMetricDescription,
		exchangeMetrics:     exchangeCounterVec,
	}
}

func (e *exporter) fetchRabbit(ch chan<- prometheus.Metric) {
	rabbitMqOverviewData, overviewError := getMetricMap(config, "overview")
	rabbitMqQueueData, queueError := getStatsInfo(config, "queues")
	exchangeData, exchangeError := getStatsInfo(config, "exchanges")

	e.overviewFetched = overviewError == nil
	e.queuesFetched = queueError == nil
	e.exchangesFetched = exchangeError == nil

	if overviewError != nil || queueError != nil || exchangeError != nil {
		e.upMetric.Set(0)
	} else {
		e.upMetric.Set(1)
	}

	log.WithField("overviewData", rabbitMqOverviewData).Debug("Overview data")
	for key, gauge := range e.overviewMetrics {
		if value, ok := rabbitMqOverviewData[key]; ok {
			log.WithFields(log.Fields{"key": key, "value": value}).Debug("Set overview metric for key")
			gauge.Set(value)
		} else {
			log.WithFields(log.Fields{"key": key}).Warn("Overview data not found")
		}
	}

	log.WithField("queueData", rabbitMqQueueData).Debug("Queue data")
	for key, gaugevec := range e.queueMetricsGauge {
		for _, queue := range rabbitMqQueueData {
			if value, ok := queue.metrics[key]; ok {
				if match_include, _ := regexp.MatchString(config.IncludeQueues, strings.ToLower(queue.name)); match_include {
					if match_skip, _ := regexp.MatchString(config.SkipQueues, strings.ToLower(queue.name)); !match_skip {

						log.WithFields(log.Fields{"vhost": queue.vhost, "queue": queue.name, "key": key, "value": value}).Debug("Set queue metric for key")
						gaugevec.WithLabelValues(queue.vhost, queue.name).Set(value)
					}
				}
			}
		}
	}
	for key, countvec := range e.queueMetricsCounter {
		for _, queue := range rabbitMqQueueData {
			if match_include, _ := regexp.MatchString(config.IncludeQueues, strings.ToLower(queue.name)); match_include {
				if match_skip, _ := regexp.MatchString(config.SkipQueues, strings.ToLower(queue.name)); !match_skip {

					if value, ok := queue.metrics[key]; ok {
						log.WithFields(log.Fields{"vhost": queue.vhost, "queue": queue.name, "key": key, "value": value}).Debug("Set queue metric for key")
						ch <- prometheus.MustNewConstMetric(countvec, prometheus.CounterValue, value, queue.vhost, queue.name)
					} else {
						ch <- prometheus.MustNewConstMetric(countvec, prometheus.CounterValue, 0, queue.vhost, queue.name)
					}
				}
			}
		}
	}

	for key, countvec := range e.exchangeMetrics {
		for _, exchange := range exchangeData {
			if value, ok := exchange.metrics[key]; ok {
				log.WithFields(log.Fields{"vhost": exchange.vhost, "exchange": exchange.name, "key": key, "value": value}).Debug("Set exchange metric for key")
				ch <- prometheus.MustNewConstMetric(countvec, prometheus.CounterValue, value, exchange.vhost, exchange.name)
			}
		}
	}

	log.Info("Metrics updated successfully.")
}

func (e *exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, gauge := range e.overviewMetrics {
		gauge.Describe(ch)
	}

	for _, gaugevec := range e.queueMetricsGauge {
		gaugevec.Describe(ch)
	}
	for _, countervec := range e.queueMetricsCounter {
		ch <- countervec
	}
	for _, exchangeMetric := range e.exchangeMetrics {
		ch <- exchangeMetric
	}

	e.upMetric.Describe(ch)
	BuildInfo.Describe(ch)
}

func (e *exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock() // To protect metrics from concurrent collects.
	defer e.mutex.Unlock()

	for _, gaugevec := range e.queueMetricsGauge {
		gaugevec.Reset()
	}

	e.fetchRabbit(ch)

	e.upMetric.Collect(ch)

	if e.overviewFetched {
		for _, gauge := range e.overviewMetrics {
			gauge.Collect(ch)
		}
	}

	if e.queuesFetched {
		for _, gaugevec := range e.queueMetricsGauge {
			gaugevec.Collect(ch)
		}
	}

	BuildInfo.Collect(ch)
}
