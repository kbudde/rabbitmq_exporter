package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"log/slog"
)

var client = &http.Client{Timeout: 15 * time.Second} //default client for test. Client is initialized in initClient()

func initClient() {
	var roots *x509.CertPool

	if data, err := os.ReadFile(config.CAFile); err == nil {
		roots = x509.NewCertPool()
		if !roots.AppendCertsFromPEM(data) {
			slog.Error("Adding certificate to rootCAs failed", "filename", config.CAFile)
		}
	} else {
		var err error
		slog.Info("Using default certificate pool")
		roots, err = x509.SystemCertPool()
		if err != nil {
			slog.Error("retriving system cert pool failed")
			slog.Any("error", err)
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
		slog.Info("Using client certificate: " + config.CertFile + " and key: " + config.KeyFile)
		if cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile); err == nil {
			tr.TLSClientConfig.ClientAuth = tls.RequireAndVerifyClientCert
			tr.TLSClientConfig.Certificates = []tls.Certificate{cert}
		} else {
			slog.Error("Loading client certificate and key failed.", 
			           "error", err, 
					   "certFile", config.CertFile,
					   "keyFile", config.KeyFile)
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
		slog.Error("Error while constructing rabbitHost request", "error", err, "host", config.RabbitURL)
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
		slog.Error("Error while retrieving data from rabbitHost", "error", err, "host", config.RabbitURL, "statusCode", status)
		return nil, "", errors.New("Error while retrieving data from rabbitHost")
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	content := resp.Header.Get("Content-type")
	if err != nil {
		return nil, "", err
	}
	slog.Debug("Metrics loaded", "body", string(body), "endpoint", endpoint)

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
