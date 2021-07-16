package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/tkanos/gonfig"
)

var (
	config        rabbitExporterConfig
	defaultConfig = rabbitExporterConfig{
		RabbitURL:          "http://127.0.0.1:15672",
		RabbitUsername:     "guest",
		RabbitPassword:     "guest",
		PublishPort:        "9419",
		PublishAddr:        "",
		OutputFormat:       "TTY", //JSON
		CAFile:             "ca.pem",
		CertFile:           "client-cert.pem",
		KeyFile:            "client-key.pem",
		InsecureSkipVerify: false,
		ExcludeMetrics:     []string{},
		SkipQueues:         regexp.MustCompile("^$"),
		IncludeQueues:      regexp.MustCompile(".*"),
		SkipVHost:          regexp.MustCompile("^$"),
		IncludeVHost:       regexp.MustCompile(".*"),
		RabbitCapabilities: parseCapabilities("no_sort,bert"),
		AlivenessVhost:     "/",
		EnabledExporters:   []string{"exchange", "node", "overview", "queue"},
		Timeout:            30,
		MaxQueues:          0,
	}
)

type rabbitExporterConfig struct {
	RabbitURL                string              `json:"rabbit_url"`
	RabbitUsername           string              `json:"rabbit_user"`
	RabbitPassword           string              `json:"rabbit_pass"`
	PublishPort              string              `json:"publish_port"`
	PublishAddr              string              `json:"publish_addr"`
	OutputFormat             string              `json:"output_format"`
	CAFile                   string              `json:"ca_file"`
	CertFile                 string              `json:"cert_file"`
	KeyFile                  string              `json:"key_file"`
	InsecureSkipVerify       bool                `json:"insecure_skip_verify"`
	ExcludeMetrics           []string            `json:"exlude_metrics"`
	SkipQueues               *regexp.Regexp      `json:"-"`
	IncludeQueues            *regexp.Regexp      `json:"-"`
	SkipVHost                *regexp.Regexp      `json:"-"`
	IncludeVHost             *regexp.Regexp      `json:"-"`
	IncludeQueuesString      string              `json:"include_queues"`
	SkipQueuesString         string              `json:"skip_queues"`
	SkipVHostString          string              `json:"skip_vhost"`
	IncludeVHostString       string              `json:"include_vhost"`
	RabbitCapabilitiesString string              `json:"rabbit_capabilities"`
	RabbitCapabilities       rabbitCapabilitySet `json:"-"`
	AlivenessVhost           string              `json:"aliveness_vhost"`
	EnabledExporters         []string            `json:"enabled_exporters"`
	Timeout                  int                 `json:"timeout"`
	MaxQueues                int                 `json:"max_queues"`
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

func initConfigFromFile(configFile string) error {
	config = rabbitExporterConfig{}
	err := gonfig.GetConf(configFile, &config)
	if err != nil {
		return err
	}

	if url := config.RabbitURL; url != "" {
		if valid, _ := regexp.MatchString("https?://[a-zA-Z.0-9]+", strings.ToLower(url)); !valid {
			panic(fmt.Errorf("rabbit URL must start with http:// or https://"))
		}
	}

	config.SkipQueues = regexp.MustCompile(config.SkipQueuesString)
	config.IncludeQueues = regexp.MustCompile(config.IncludeQueuesString)
	config.SkipVHost = regexp.MustCompile(config.SkipVHostString)
	config.IncludeVHost = regexp.MustCompile(config.IncludeVHostString)
	config.RabbitCapabilities = parseCapabilities(config.RabbitCapabilitiesString)
	return nil
}

func initConfig() {
	config = defaultConfig
	if url := os.Getenv("RABBIT_URL"); url != "" {
		if valid, _ := regexp.MatchString("https?://[a-zA-Z.0-9]+", strings.ToLower(url)); valid {
			config.RabbitURL = url
		} else {
			panic(fmt.Errorf("rabbit URL must start with http:// or https://"))
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
		} else {
			panic(fmt.Errorf("the configured port is not a valid number: %v", port))
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
	if certfile := os.Getenv("CERTFILE"); certfile != "" {
		config.CertFile = certfile
	}
	if keyfile := os.Getenv("KEYFILE"); keyfile != "" {
		config.KeyFile = keyfile
	}
	if insecureSkipVerify := os.Getenv("SKIPVERIFY"); insecureSkipVerify == "true" || insecureSkipVerify == "1" || insecureSkipVerify == "TRUE" {
		config.InsecureSkipVerify = true
	}

	if ExcludeMetrics := os.Getenv("EXCLUDE_METRICS"); ExcludeMetrics != "" {
		config.ExcludeMetrics = strings.Split(ExcludeMetrics, ",")
	}

	if SkipQueues := os.Getenv("SKIP_QUEUES"); SkipQueues != "" {
		config.SkipQueues = regexp.MustCompile(SkipQueues)
	}

	if IncludeQueues := os.Getenv("INCLUDE_QUEUES"); IncludeQueues != "" {
		config.IncludeQueues = regexp.MustCompile(IncludeQueues)
	}

	if SkipVHost := os.Getenv("SKIP_VHOST"); SkipVHost != "" {
		config.SkipVHost = regexp.MustCompile(SkipVHost)
	}

	if IncludeVHost := os.Getenv("INCLUDE_VHOST"); IncludeVHost != "" {
		config.IncludeVHost = regexp.MustCompile(IncludeVHost)
	}

	if rawCapabilities := os.Getenv("RABBIT_CAPABILITIES"); rawCapabilities != "" {
		config.RabbitCapabilities = parseCapabilities(rawCapabilities)
	}

	if enabledExporters := os.Getenv("RABBIT_EXPORTERS"); enabledExporters != "" {
		config.EnabledExporters = strings.Split(enabledExporters, ",")
	}

	if alivenessVhost := os.Getenv("ALIVENESS_VHOST"); alivenessVhost != "" {
		config.AlivenessVhost = alivenessVhost
	}

	if timeout := os.Getenv("RABBIT_TIMEOUT"); timeout != "" {
		t, err := strconv.Atoi(timeout)
		if err != nil {
			panic(fmt.Errorf("timeout is not a number: %v", err))
		}
		config.Timeout = t
	}

	if maxQueues := os.Getenv("MAX_QUEUES"); maxQueues != "" {
		m, err := strconv.Atoi(maxQueues)
		if err != nil {
			panic(fmt.Errorf("maxQueues is not a number: %v", err))
		}
		config.MaxQueues = m
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
