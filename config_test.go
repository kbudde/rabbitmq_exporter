package main

import (
	"os"
	"reflect"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestEnvironmentSettingURL_HTTPS(t *testing.T) {
	newValue := "https://testURL"
	os.Setenv("RABBIT_URL", newValue)
	defer os.Unsetenv("RABBIT_URL")
	initConfig()
	if config.RabbitURL != newValue {
		t.Errorf("Expected config.RABBIT_URL to be modified. Found=%v, expected=%v", config.RabbitURL, newValue)
	}
}

func TestEnvironmentSettingURL_HTTP(t *testing.T) {
	newValue := "http://testURL"
	os.Setenv("RABBIT_URL", newValue)
	defer os.Unsetenv("RABBIT_URL")
	initConfig()
	if config.RabbitURL != newValue {
		t.Errorf("Expected config.RABBIT_URL to be modified. Found=%v, expected=%v", config.RabbitURL, newValue)
	}
}

func TestEnvironmentSettingUser(t *testing.T) {
	newValue := "username"
	os.Setenv("RABBIT_USER", newValue)
	defer os.Unsetenv("RABBIT_USER")
	initConfig()
	if config.RabbitUsername != newValue {
		t.Errorf("Expected config.RABBIT_USER to be modified. Found=%v, expected=%v", config.RabbitUsername, newValue)
	}
}

func TestEnvironmentSettingPassword(t *testing.T) {
	newValue := "password"
	os.Setenv("RABBIT_PASSWORD", newValue)
	defer os.Unsetenv("RABBIT_PASSWORD")
	initConfig()
	if config.RabbitPassword != newValue {
		t.Errorf("Expected config.RABBIT_PASSWORD to be modified. Found=%v, expected=%v", config.RabbitPassword, newValue)
	}
}

func TestEnvironmentSettingUserFile(t *testing.T) {
	fileValue := "./testdata/username_file"
	newValue := "username"
	os.Setenv("RABBIT_USER_FILE", fileValue)
	defer os.Unsetenv("RABBIT_USER_FILE")
	initConfig()
	if config.RabbitUsername != newValue {
		t.Errorf("Expected config.RABBIT_USER to be modified. Found=%v, expected=%v", config.RabbitUsername, newValue)
	}
}

func TestEnvironmentSettingPasswordFile(t *testing.T) {
	fileValue := "./testdata/password_file"
	newValue := "password"
	os.Setenv("RABBIT_PASSWORD_FILE", fileValue)
	defer os.Unsetenv("RABBIT_PASSWORD_FILE")
	initConfig()
	if config.RabbitPassword != newValue {
		t.Errorf("Expected config.RABBIT_PASSWORD to be modified. Found=%v, expected=%v", config.RabbitPassword, newValue)
	}
}

func TestEnvironmentSettingFormat(t *testing.T) {
	newValue := "json"
	os.Setenv("OUTPUT_FORMAT", newValue)
	defer os.Unsetenv("OUTPUT_FORMAT")
	initConfig()
	if config.OutputFormat != newValue {
		t.Errorf("Expected config.OUTPUT_FORMAT to be modified. Found=%v, expected=%v", config.OutputFormat, newValue)
	}
}

func TestConfig_Http_URL(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("initConfig should panic on invalid url config")
		}
	}()
	url := config.RabbitURL
	os.Setenv("RABBIT_URL", "ftp://test")
	defer os.Unsetenv("RABBIT_URL")
	initConfig()
	if config.RabbitURL != url {
		t.Errorf("Invalid URL. It should start with http(s)://. expected=%v,got=%v", url, config.RabbitURL)
	}
}

func TestConfig_Capabilities(t *testing.T) {
	defer os.Unsetenv("RABBIT_CAPABILITIES")

	os.Unsetenv("RABBIT_CAPABILITIES")
	initConfig()
	if !config.RabbitCapabilities[rabbitCapBert] {
		t.Error("Bert support should be enabled by default")
	}
	if !config.RabbitCapabilities[rabbitCapNoSort] {
		t.Error("No_sort support should be enabled by default")
	}

	var needToSupport = []rabbitCapability{"no_sort", "bert"}
	for _, cap := range needToSupport {
		os.Setenv("RABBIT_CAPABILITIES", "junk_cap, another_with_spaces_around ,  "+string(cap)+", done")
		initConfig()
		expected := rabbitCapabilitySet{cap: true}
		if !reflect.DeepEqual(config.RabbitCapabilities, expected) {
			t.Errorf("Capability '%s' wasn't properly detected from env", cap)
		}
	}
	//disable all capabilities
	os.Setenv("RABBIT_CAPABILITIES", " ")
	initConfig()
	expected := rabbitCapabilitySet{}
	if !reflect.DeepEqual(config.RabbitCapabilities, expected) {
		t.Errorf("Capabilities '%v' should be empty", config.RabbitCapabilities)
	}
}

func TestConfig_EnabledExporters(t *testing.T) {
	enabledExporters := []string{"overview", "connections"}
	os.Setenv("RABBIT_EXPORTERS", "overview,connections")
	defer os.Unsetenv("RABBIT_EXPORTERS")
	initConfig()
	if diff := pretty.Compare(config.EnabledExporters, enabledExporters); diff != "" {
		t.Errorf("Invalid Exporters list. diff\n%v", diff)
	}
}

func TestConfig_RabbitConnection_Default(t *testing.T) {
	defer os.Unsetenv("RABBIT_CONNECTION")

	os.Unsetenv("RABBIT_CONNECTION")
	initConfig()

	if config.RabbitConnection != "direct" {
		t.Errorf("RabbitConnection unspecified. It should default to direct. expected=%v,got=%v", "direct", config.RabbitConnection)
	}
}

func TestConfig_RabbitConnection_LoadBalaner(t *testing.T) {
    newValue := "loadbalancer"
	defer os.Unsetenv("RABBIT_CONNECTION")

	os.Setenv("RABBIT_CONNECTION", newValue)
	initConfig()

	if config.RabbitConnection != newValue {
		t.Errorf("RabbitConnection specified. It should be modified. expected=%v,got=%v", newValue, config.RabbitConnection)
	}
}

func TestConfig_RabbitConnection_Invalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("initConfig should panic on invalid rabbit connection config")
		}
    }()
    newValue := "invalid"
	defer os.Unsetenv("RABBIT_CONNECTION")

	os.Setenv("RABBIT_CONNECTION", newValue)
	initConfig()
}
