package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
	"github.com/tkanos/gonfig"
)

var (
	config        rabbitExporterConfig
	defaultConfig = rabbitExporterConfig{
		RabbitURL:          "http://127.0.0.1:15672",
		RabbitUsername:     "guest",
		RabbitPassword:     "guest",
		RabbitConnection:   "direct",
		PublishPort:        "9419",
		PublishAddr:        "",
		OutputFormat:       "TTY", //JSON
		CAFile:             "ca.pem",
		CertFile:           "client-cert.pem",
		KeyFile:            "client-key.pem",
		InsecureSkipVerify: false,
		ExcludeMetrics:     []string{},
		SkipExchanges:      regexp.MustCompile("^$"),
		IncludeExchanges:   regexp.MustCompile(".*"),
		SkipQueues:         regexp.MustCompile("^$"),
		IncludeQueues:      regexp.MustCompile(".*"),
		SkipVHost:          regexp.MustCompile("^$"),
		IncludeVHost:       regexp.MustCompile(".*"),
		RabbitCapabilities: parseCapabilities("no_sort,bert"),
		AlivenessVhost:     "/",
		EnabledExporters:   []string{"exchange", "node", "overview", "queue"},
		Timeout:            30,
		MaxQueues:          0,
		UseVault:           false,
		VaultAddress:       "http://127.0.0.1:8200",
		VaultRoleID:        "test",
		VaultSecretID:      "test",
		VaultSecretPath:    "deployments/unit/dev/user/passwords_yml",
		VaultSecretMountPath: "secret_v2",
		CredentialNameInVaultSecret: "rabbitmq_monitoring_password",
	}
)

type rabbitExporterConfig struct {
	RabbitURL                string              `json:"rabbit_url"`
	RabbitUsername           string              `json:"rabbit_user"`
	RabbitPassword           string              `json:"rabbit_pass"`
	RabbitConnection         string              `json:"rabbit_connection"`
	PublishPort              string              `json:"publish_port"`
	PublishAddr              string              `json:"publish_addr"`
	OutputFormat             string              `json:"output_format"`
	CAFile                   string              `json:"ca_file"`
	CertFile                 string              `json:"cert_file"`
	KeyFile                  string              `json:"key_file"`
	InsecureSkipVerify       bool                `json:"insecure_skip_verify"`
	ExcludeMetrics           []string            `json:"exlude_metrics"`
	SkipExchanges            *regexp.Regexp      `json:"-"`
	IncludeExchanges         *regexp.Regexp      `json:"-"`
	SkipQueues               *regexp.Regexp      `json:"-"`
	IncludeQueues            *regexp.Regexp      `json:"-"`
	SkipVHost                *regexp.Regexp      `json:"-"`
	IncludeVHost             *regexp.Regexp      `json:"-"`
	IncludeExchangesString   string              `json:"include_exchanges"`
	SkipExchangesString      string              `json:"skip_exchanges"`
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
	UseVault                 bool                `json:"use_vault"`
	VaultAddress             string              `json:"vault_address"`
	VaultRoleID              string              `json:"vault_role_id"`
	VaultSecretID            string              `json:"vault_secret_id"`
	VaultSecretPath          string              `json:"vault_secret_path"`
	VaultSecretMountPath     string              `json:"vault_secret_mount_path"`
	CredentialNameInVaultSecret string           `json:"credential_name_in_vault_secret"`
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

	config.SkipExchanges = regexp.MustCompile(config.SkipExchangesString)
	config.IncludeExchanges = regexp.MustCompile(config.IncludeExchangesString)
	config.SkipQueues = regexp.MustCompile(config.SkipQueuesString)
	config.IncludeQueues = regexp.MustCompile(config.IncludeQueuesString)
	config.SkipVHost = regexp.MustCompile(config.SkipVHostString)
	config.IncludeVHost = regexp.MustCompile(config.IncludeVHostString)
	config.RabbitCapabilities = parseCapabilities(config.RabbitCapabilitiesString)

	SetPasswordIfVaultIsUsed()

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

	if connection := os.Getenv("RABBIT_CONNECTION"); connection != "" {
		if valid, _ := regexp.MatchString("(direct|loadbalancer)", connection); valid {
			config.RabbitConnection = connection
		} else {
			panic(fmt.Errorf("rabbit connection must be direct or loadbalancer"))
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

	if SkipExchanges := os.Getenv("SKIP_EXCHANGES"); SkipExchanges != "" {
		config.SkipExchanges = regexp.MustCompile(SkipExchanges)
	}

	if IncludeExchanges := os.Getenv("INCLUDE_EXCHANGES"); IncludeExchanges != "" {
		config.IncludeExchanges = regexp.MustCompile(IncludeExchanges)
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

func selfLabel(config rabbitExporterConfig, isSelf bool) string {
	if config.RabbitConnection == "loadbalancer" {
        return "lb"
    } else if isSelf {
		return "1"
	} else {
		return "0"
	}
}

func SetPasswordIfVaultIsUsed() {
	if !config.UseVault {
		return
	}
	client, err := vault.New(vault.WithAddress(config.VaultAddress),)
	if err != nil {
		panic(fmt.Errorf("failed to create vault client: %v", err))
	}
	ctx := context.Background()
	resp, err := client.Auth.AppRoleLogin(
		ctx,
		schema.AppRoleLoginRequest{
			RoleId:   config.VaultRoleID,
			SecretId: config.VaultSecretID,
		},
	)
	if err != nil {
		panic(fmt.Errorf("failed to login to vault: %v", err))
	}
	if err := client.SetToken(resp.Auth.ClientToken); err != nil {
		panic(fmt.Errorf("failed to set vault token: %v", err))
	}
	data, err := client.Secrets.KvV2Read(
		ctx,
		config.VaultSecretPath,
		vault.WithMountPath(config.VaultSecretMountPath),
	)
	if err != nil {
		panic(fmt.Errorf("failed to get Vault secret: %v", err))
	}

	config.RabbitPassword = data.Data.Data[config.CredentialNameInVaultSecret].(string)
}
