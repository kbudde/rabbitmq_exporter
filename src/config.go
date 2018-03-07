package main

import (
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	config        rabbitExporterConfig
	defaultConfig = rabbitExporterConfig{
		RabbitURL:          "http://localhost:15672",
		RabbitUsername:     "guest",
		RabbitPassword:     "guest",
		PublishPort:        "9090",
		PublishAddr:        "",
		OutputFormat:       "TTY", //JSON
		CAFile:             "ca.pem",
		InsecureSkipVerify: false,
		SkipQueues:         regexp.MustCompile("^$"),
		IncludeQueues:      regexp.MustCompile(".*"),
		RabbitCapabilities: make(rabbitCapabilitySet),
		EnabledExporters:   []string{"exchange", "node", "overview", "queue"},
	}
)

type rabbitExporterConfig struct {
	RabbitURL          string
	RabbitUsername     string
	RabbitPassword     string
	PublishPort        string
	PublishAddr        string
	OutputFormat       string
	CAFile             string
	InsecureSkipVerify bool
	SkipQueues         *regexp.Regexp
	IncludeQueues      *regexp.Regexp
	RabbitCapabilities rabbitCapabilitySet
	EnabledExporters   []string
}

type rabbitCapability string
type rabbitCapabilitySet map[rabbitCapability]bool

const (
	rabbitCapNoSort rabbitCapability = "no_sort"
	rabbitCapBert   rabbitCapability = "bert"
)

var allRabbitCapabilities = rabbitCapabilitySet{
	rabbitCapNoSort: true,
	rabbitCapBert:   true,
}

func initConfig() {
	config = defaultConfig
	if url := os.Getenv("RABBIT_URL"); url != "" {
		if valid, _ := regexp.MatchString("https?://[a-zA-Z.0-9]+", strings.ToLower(url)); valid {
			config.RabbitURL = url
		}
	}

	var user string
	var pass string

	if len(os.Getenv("RABBIT_USER_FILE")) != 0 {
		fileContents, err := ioutil.ReadFile(os.Getenv("RABBIT_USER_FILE"))
		if err != nil {
			panic(err)
		}
		user = strings.TrimSpace(string(fileContents))
	} else {
		user = os.Getenv("RABBIT_USER")
	}

	if user != "" {
		config.RabbitUsername = user
	}

	if len(os.Getenv("RABBIT_PASSWORD_FILE")) != 0 {
		fileContents, err := ioutil.ReadFile(os.Getenv("RABBIT_PASSWORD_FILE"))
		if err != nil {
			panic(err)
		}
		pass = strings.TrimSpace(string(fileContents))
	} else {
		pass = os.Getenv("RABBIT_PASSWORD")
	}
	if pass != "" {
		config.RabbitPassword = pass
	}

	if port := os.Getenv("PUBLISH_PORT"); port != "" {
		if _, err := strconv.Atoi(port); err == nil {
			config.PublishPort = port
		}

	}

	if addr := os.Getenv("PUBLISH_ADDR"); addr != "" {
		config.PublishAddr = addr
	}

	if output := os.Getenv("OUTPUT_FORMAT"); output != "" {
		config.OutputFormat = output
	}

	if cafile := os.Getenv("CAFILE"); cafile != "" {
		config.CAFile = cafile
	}
	if insecureSkipVerify := os.Getenv("SKIPVERIFY"); insecureSkipVerify == "true" || insecureSkipVerify == "1" {
		config.InsecureSkipVerify = true
	}

	if SkipQueues := os.Getenv("SKIP_QUEUES"); SkipQueues != "" {
		config.SkipQueues = regexp.MustCompile(SkipQueues)
	}

	if IncludeQueues := os.Getenv("INCLUDE_QUEUES"); IncludeQueues != "" {
		config.IncludeQueues = regexp.MustCompile(IncludeQueues)
	}

	if rawCapabilities := os.Getenv("RABBIT_CAPABILITIES"); rawCapabilities != "" {
		config.RabbitCapabilities = parseCapabilities(rawCapabilities)
	}

	if enabledExporters := os.Getenv("RABBIT_EXPORTERS"); enabledExporters != "" {
		config.EnabledExporters = strings.Split(enabledExporters, ",")
	}
}

func parseCapabilities(raw string) rabbitCapabilitySet {
	result := make(rabbitCapabilitySet)
	candidates := strings.Split(raw, ",")
	for _, maybeCapStr := range candidates {
		maybeCap := rabbitCapability(strings.TrimSpace(maybeCapStr))
		enabled, present := allRabbitCapabilities[maybeCap]
		if enabled && present {
			result[maybeCap] = true
		}
	}
	return result
}

func isCapEnabled(config rabbitExporterConfig, cap rabbitCapability) bool {
	exists, enabled := config.RabbitCapabilities[cap]
	return exists && enabled
}
