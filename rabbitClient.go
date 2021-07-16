package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

var client = &http.Client{Timeout: 15 * time.Second} //default client for test. Client is initialized in initClient()

func initClient() {
	var roots *x509.CertPool

	if data, err := ioutil.ReadFile(config.CAFile); err == nil {
		roots = x509.NewCertPool()
		if !roots.AppendCertsFromPEM(data) {
			log.WithField("filename", config.CAFile).Error("Adding certificate to rootCAs failed")
		}
	} else {
		var err error
		log.Info("Using default certificate pool")
		roots, err = x509.SystemCertPool()
		if err != nil {
			log.WithError(err).Error("retriving system cert pool failed")
		}
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.InsecureSkipVerify,
			RootCAs:            roots,
		},
	}

	_, errCertFile := os.Stat(config.CertFile)
	_, errKeyFile := os.Stat(config.KeyFile)
	if errCertFile == nil && errKeyFile == nil {
		log.Info("Using client certificate: " + config.CertFile + " and key: " + config.KeyFile)
		if cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile); err == nil {
			tr.TLSClientConfig.ClientAuth = tls.RequireAndVerifyClientCert
			tr.TLSClientConfig.Certificates = []tls.Certificate{cert}
		} else {
			log.WithField("certFile", config.CertFile).
				WithField("keyFile", config.KeyFile).
				Error("Loading client certificate and key failed: ", err)
		}
	}

	client = &http.Client{
		Transport: tr,
		Timeout:   time.Duration(config.Timeout) * time.Second,
	}

}

func apiRequest(config rabbitExporterConfig, endpoint string) ([]byte, string, error) {
	var args string
	enabled, exists := config.RabbitCapabilities[rabbitCapNoSort]
	if enabled && exists {
		args = "?sort="
	}
	if endpoint == "aliveness-test" {
		escapeAlivenessVhost := url.QueryEscape(config.AlivenessVhost)
		args = "/" + escapeAlivenessVhost
	}
	req, err := http.NewRequest("GET", config.RabbitURL+"/api/"+endpoint+args, nil)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "host": config.RabbitURL}).Error("Error while constructing rabbitHost request")
		return nil, "", errors.New("Error while constructing rabbitHost request")
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
		return nil, "", errors.New("Error while retrieving data from rabbitHost")
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	content := resp.Header.Get("Content-type")
	if err != nil {
		return nil, "", err
	}
	log.WithFields(log.Fields{"body": string(body), "endpoint": endpoint}).Debug("Metrics loaded")

	return body, content, nil
}

func loadMetrics(config rabbitExporterConfig, endpoint string) (RabbitReply, error) {
	body, content, err := apiRequest(config, endpoint)

	if err != nil {
		return nil, err
	}
	return MakeReply(content, body)
}

func getStatsInfo(config rabbitExporterConfig, apiEndpoint string, labels []string) ([]StatsInfo, error) {
	var q []StatsInfo

	reply, err := loadMetrics(config, apiEndpoint)
	if err != nil {
		return q, err
	}

	q = reply.MakeStatsInfo(labels)

	return q, nil
}

func getMetricMap(config rabbitExporterConfig, apiEndpoint string) (MetricMap, error) {
	var overview MetricMap

	body, content, err := apiRequest(config, apiEndpoint)
	if err != nil {
		return overview, err
	}

	reply, err := MakeReply(content, body)
	if err != nil {
		return overview, err
	}

	return reply.MakeMap(), nil
}

func acceptContentType(config rabbitExporterConfig) string {
	if isCapEnabled(config, rabbitCapBert) {
		return "application/bert, application/json;q=0.1"
	}
	return "application/json"
}
