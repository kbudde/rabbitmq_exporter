package main

import (
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"
)

type exporter struct {
	mutex               sync.RWMutex
	queueMetricsGauge   map[string]*prometheus.GaugeVec
	queueMetricsCounter map[string]*prometheus.CounterVec
	overviewMetrics     map[string]prometheus.Gauge
	upMetric            prometheus.Gauge
	exchangeMetrics     map[string]*prometheus.CounterVec
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

func (e *exporter) fetchRabbit() {
	rabbitMqOverviewData, overviewError := getMetricMap(config, "overview")
	rabbitMqQueueData, queueError := getStatsInfo(config, "queues")
	exchangeData, exchangeError := getStatsInfo(config, "exchanges")

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
				log.WithFields(log.Fields{"vhost": queue.vhost, "queue": queue.name, "key": key, "value": value}).Debug("Set queue metric for key")
				gaugevec.WithLabelValues(queue.vhost, queue.name).Set(value)
			} else {
				//log.WithFields(log.Fields{"queue": queue, "key": key}).Warn("Queue data not found")
			}
		}
	}
	for key, countvec := range e.queueMetricsCounter {
		for _, queue := range rabbitMqQueueData {
			if value, ok := queue.metrics[key]; ok {
				log.WithFields(log.Fields{"vhost": queue.vhost, "queue": queue.name, "key": key, "value": value}).Debug("Set queue metric for key")
				countvec.WithLabelValues(queue.vhost, queue.name).Set(value)
			} else {
				//log.WithFields(log.Fields{"queue": queue, "key": key}).Warn("Queue data not found")
			}
		}
	}

	for key, countvec := range e.exchangeMetrics {
		for _, exchange := range exchangeData {
			if value, ok := exchange.metrics[key]; ok {
				log.WithFields(log.Fields{"vhost": exchange.vhost, "exchange": exchange.name, "key": key, "value": value}).Debug("Set exchange metric for key")
				countvec.WithLabelValues(exchange.vhost, exchange.name).Set(value)
			} else {
				//log.WithFields(log.Fields{"queue": queue, "key": key}).Warn("Queue data not found")
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
		countervec.Describe(ch)
	}
	for _, exchangeMetric := range e.exchangeMetrics {
		exchangeMetric.Describe(ch)
	}
}

func (e *exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock() // To protect metrics from concurrent collects.
	defer e.mutex.Unlock()

	for _, gaugevec := range e.queueMetricsGauge {
		gaugevec.Reset()
	}
	for _, countvec := range e.queueMetricsCounter {
		countvec.Reset()
	}
	for _, exchangeMetric := range e.exchangeMetrics {
		exchangeMetric.Reset()
	}

	e.fetchRabbit()

	e.upMetric.Collect(ch)

	for _, gauge := range e.overviewMetrics {
		gauge.Collect(ch)
	}

	for _, gaugevec := range e.queueMetricsGauge {
		gaugevec.Collect(ch)
	}
	for _, countervec := range e.queueMetricsCounter {
		countervec.Collect(ch)
	}

	for _, exchangeMetric := range e.exchangeMetrics {
		exchangeMetric.Collect(ch)
	}

	BuildInfo.Collect(ch)
}
