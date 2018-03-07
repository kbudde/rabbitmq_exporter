// +build pprof

package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/anchorfree/rabbitmq_exporter/testenv"
)

func TestPProf(t *testing.T) {
	// go test -v -run TestPProf -tags pprof -cpuprofile=cpuprof.out
	// go-torch rabbitmq_exporter.test cpuprof.out
	var env testenv.TestEnvironment
	var exporterURL string
	var rabbitManagementURL string
	t.Run("Preparation", func(t *testing.T) {
		env = testenv.NewEnvironment(t, testenv.RabbitMQ3Latest)

		exporterURL = fmt.Sprintf("http://localhost:%s/metrics", defaultConfig.PublishPort)
		rabbitManagementURL = env.ManagementURL()
		os.Setenv("LOG_LEVEL", "FATAL")
		defer os.Unsetenv("LOG_LEVEL")
		go main()

		for i := 0; i < 100; i++ {
			queue := fmt.Sprintf("queue-%d", i)
			env.Rabbit.DeclareQueue(queue, false)
		}

		time.Sleep(5 * time.Second) // give rabbitmq management plugin a bit of time
	})
	defer env.CleanUp() // do not panic or exit fatally or the container will stay up
	os.Setenv("RABBIT_URL", rabbitManagementURL)
	defer os.Unsetenv("RABBIT_URL")

	t.Run("Fetch Exporter Bert, no sort", func(t *testing.T) {
		os.Setenv("RABBIT_CAPABILITIES", "bert,no_sort")
		defer os.Unsetenv("RABBIT_CAPABILITIES")
		initConfig()
		for i := 0; i < 100; i++ {
			testenv.GetOrDie(exporterURL, 5*time.Second)
		}
	})
	t.Run("Fetch Exporter Json, no sort", func(t *testing.T) {
		os.Setenv("RABBIT_CAPABILITIES", "no_sort")
		defer os.Unsetenv("RABBIT_CAPABILITIES")
		initConfig()
		for i := 0; i < 100; i++ {
			testenv.GetOrDie(exporterURL, 5*time.Second)
		}
	})
}
