package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
)

var client = &http.Client{Timeout: 10 * time.Second}

func initClient() {
	roots := x509.NewCertPool()

	if data, err := ioutil.ReadFile(config.CAFile); err == nil {
		if !roots.AppendCertsFromPEM(data) {
			log.WithField("filename", config.CAFile).Error("Adding certificate to rootCAs failed")
		}
	} else {
		log.Info("Using default certificate pool")
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.InsecureSkipVerify,
			RootCAs:            roots,
		},
	}

	client = &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

}

func loadMetrics(config rabbitExporterConfig, endpoint string) (RabbitReply, error) {
	var args string
	enabled, exists := config.RabbitCapabilities[rabbitCapNoSort]
	if enabled && exists {
		args = "?sort="
	}

	req, err := http.NewRequest("GET", config.RabbitURL+"/api/"+endpoint+args, nil)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "host": config.RabbitURL}).Error("Error while constructing rabbitHost request")
		return nil, errors.New("Error while constructing rabbitHost request")
	}

	req.SetBasicAuth(config.RabbitUsername, config.RabbitPassword)
	req.Header.Add("Accept", acceptContentType(config))

	resp, err := client.Do(req)

	if err != nil || resp == nil || resp.StatusCode != 200 {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		log.WithFields(log.Fields{"error": err, "host": config.RabbitURL, "statusCode": status}).Error("Error while retrieving data from rabbitHost")
		return nil, errors.New("Error while retrieving data from rabbitHost")
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	log.WithFields(log.Fields{"body": string(body), "endpoint": endpoint}).Debug("Metrics loaded")

	return MakeReply(config, body)
}

func getStatsInfo(config rabbitExporterConfig, apiEndpoint string) ([]StatsInfo, error) {
	var q []StatsInfo

	reply, err := loadMetrics(config, apiEndpoint)
	if err != nil {
		return q, err
	}

	q = reply.MakeStatsInfo()

	return q, nil
}

func getMetricMap(config rabbitExporterConfig, apiEndpoint string) (MetricMap, error) {
	var overview MetricMap

	reply, err := loadMetrics(config, apiEndpoint)
	if err != nil {
		return overview, err
	}

	return reply.MakeMap(), nil
}

func acceptContentType(config rabbitExporterConfig) string {
	if isCapEnabled(config, rabbitCapBert) {
		return "application/bert"
	}
	return "application/json"
}
