package main

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus/exporter-toolkit/web"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"

)

func initLogger() {
	logHandlerOptions := slog.HandlerOptions{}
	envLogLevel := os.Getenv("LOG_LEVEL")
	if envLogLevel != "" {
		logLevel := slog.Level.Level(slog.LevelInfo)
		error := logLevel.UnmarshalText([]byte(envLogLevel))
		if error != nil {
			slog.Error("Allowed case-independent log level values: debug, info, warn, error.", "LOG_LEVEL", envLogLevel)
			panic(error)
		}
		logHandlerOptions = slog.HandlerOptions{Level: logLevel}
	}
	if strings.ToUpper(config.OutputFormat) == "JSON" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &logHandlerOptions)))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &logHandlerOptions)))
	}
}

func main() {
	var checkURL = kingpin.Flag("check-url", "Curl url and return exit code (http: 200 => 0, otherwise 1)").Default("").String()
	var configFile = kingpin.Flag("config-file", "path to json config").Default("conf/rabbitmq.conf").String()
	webFlags := webflag.AddFlags(kingpin.CommandLine, ":9417")
	kingpin.Parse()

	if *checkURL != "" { // do a single http get request. Used in docker healthckecks as curl is not inside the image
		curl(*checkURL)
		return
	}

	err := initConfigFromFile(*configFile)                  //Try parsing config file
	if _, isPathError := err.(*os.PathError); isPathError { // No file => use environment variables
		initConfig()
	} else if err != nil {
		panic(err)
	}

	initLogger()
	initClient()
	exporter := newExporter()
	prometheus.MustRegister(exporter)

	slog.Info("Starting RabbitMQ exporter", 
			  "VERSION",    Version, 
			  "REVISION",   Revision, 
			  "BRANCH",     Branch,
			  "BUILD_DATE", BuildDate)

	slog.Info("Active Configuration",
			  "RABBIT_URL",          config.RabbitURL,
			  "RABBIT_USER",         config.RabbitUsername,
			  "RABBIT_CONNECTION",   config.RabbitConnection,
			  "OUTPUT_FORMAT",       config.OutputFormat,
			  "RABBIT_CAPABILITIES", formatCapabilities(config.RabbitCapabilities),
			  "RABBIT_EXPORTERS",    config.EnabledExporters,
			  "CAFILE",              config.CAFile,
			  "CERTFILE",            config.CertFile,
			  "KEYFILE",             config.KeyFile,
			  "SKIPVERIFY",          config.InsecureSkipVerify,
			  "EXCLUDE_METRICS",     config.ExcludeMetrics,
			  "SKIP_EXCHANGES",      config.SkipExchanges.String(),
			  "INCLUDE_EXCHANGES",   config.IncludeExchanges.String(),
			  "SKIP_QUEUES",         config.SkipQueues.String(),
			  "INCLUDE_QUEUES",      config.IncludeQueues.String(),
			  "SKIP_VHOST",          config.SkipVHost.String(),
			  "INCLUDE_VHOST",       config.IncludeVHost.String(),
			  "RABBIT_TIMEOUT",      config.Timeout,
			  "MAX_QUEUES",          config.MaxQueues,

	)

	http.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
             <head><title>RabbitMQ Exporter</title></head>
             <body>
             <h1>RabbitMQ Exporter</h1>
             <p><a href='/metrics'>Metrics</a></p>
             </body>
             </html>`))
	})
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if exporter.LastScrapeOK() {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusGatewayTimeout)
		}
	})

	server := &http.Server{}

	go func() {
		if err := web.ListenAndServe(server, webFlags, slog.Default()); err != nil {
			slog.Any("fatal error", err)
			panic(err)
		}
	}()

	<-runService()
	slog.Info("Shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := server.Shutdown(ctx); err != nil {
		slog.Any("fatal error", err)
		panic(err)
	}
	cancel()
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
