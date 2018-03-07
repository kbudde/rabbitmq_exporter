package main

import (
	"bytes"
	"net/http"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"
)

const defaultLogLevel = log.InfoLevel

func initLogger() {
	log.SetLevel(getLogLevel())
	if strings.ToUpper(config.OutputFormat) == "JSON" {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		// The TextFormatter is default, you don't actually have to do this.
		log.SetFormatter(&log.TextFormatter{})
	}
}

func main() {
	initConfig()
	initLogger()
	initClient()
	exporter := newExporter()
	prometheus.MustRegister(exporter)

	http.Handle("/metrics", prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>RabbitMQ Exporter</title></head>
             <body>
             <h1>RabbitMQ Exporter</h1>
             <p><a href='/metrics'>Metrics</a></p>
             </body>
             </html>`))
	})

	log.WithFields(log.Fields{
		"VERSION":    Version,
		"REVISION":   Revision,
		"BRANCH":     Branch,
		"BUILD_DATE": BuildDate,
		//		"RABBIT_PASSWORD": config.RABBIT_PASSWORD,
	}).Info("Starting RabbitMQ exporter")

	log.WithFields(log.Fields{
		"PUBLISH_ADDR":        config.PublishAddr,
		"PUBLISH_PORT":        config.PublishPort,
		"RABBIT_URL":          config.RabbitURL,
		"RABBIT_USER":         config.RabbitUsername,
		"OUTPUT_FORMAT":       config.OutputFormat,
		"RABBIT_CAPABILITIES": formatCapabilities(config.RabbitCapabilities),
		"RABBIT_EXPORTERS":    config.EnabledExporters,
		//		"RABBIT_PASSWORD": config.RABBIT_PASSWORD,
	}).Info("Active Configuration")

	log.Fatal(http.ListenAndServe(config.PublishAddr+":"+config.PublishPort, nil))
}

func getLogLevel() log.Level {
	lvl := strings.ToLower(os.Getenv("LOG_LEVEL"))
	level, err := log.ParseLevel(lvl)
	if err != nil {
		level = defaultLogLevel
	}
	return level
}

func formatCapabilities(caps rabbitCapabilitySet) string {
	var buffer bytes.Buffer
	first := true
	for k := range caps {
		if !first {
			buffer.WriteString(",")
		}
		first = false
		buffer.WriteString(string(k))
	}
	return buffer.String()
}
