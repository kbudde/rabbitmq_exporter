// Package testenv provides a rabbitmq test environment in docker for a full set of integration tests.
// Some usefull helper functions for rabbitmq interaction are included as well
package testenv

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"os"

	"log"

	dockertest "github.com/ory/dockertest/v3"
)

//list of docker tags with rabbitmq versions
const (
	RabbitMQ3Latest = "3-management-alpine"
)

// MaxWait is time before the docker setup will fail with timeout
var MaxWait = 20 * time.Second

//TestEnvironment contains all necessars
type TestEnvironment struct {
	t        *testing.T
	docker   *dockertest.Pool
	resource *dockertest.Resource
	Rabbit   rabbit
}

//NewEnvironment sets up a new environment. It will nlog fatal if something goes wrong
func NewEnvironment(t *testing.T, dockerTag string) TestEnvironment {
	t.Helper()
	tenv := TestEnvironment{t: t}

	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("Could not connect to docker: %s", err)
	}

	tenv.docker = pool
	tenv.docker.MaxWait = MaxWait

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.BuildAndRunWithOptions("./testenv/Dockerfile", &dockertest.RunOptions{Name: "rabbitmq-test", Hostname: "localtest"})
	if err != nil {
		t.Fatalf("Could not start resource: %s", err)
	}
	if err := resource.Expire(120); err != nil {
		t.Fatalf("Could not set container expiration: %s", err)
	}

	tenv.resource = resource

	checkManagementWebsite := func() error {
		_, err := GetURL(tenv.ManagementURL(), 5*time.Second)
		return err
	}

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := tenv.docker.Retry(checkManagementWebsite); err != nil {
		perr := tenv.docker.Purge(resource)
		log.Fatalf("Could not connect to docker: %s; Purge Error: %s", err, perr)
	}

	r := rabbit{}
	r.connect(tenv.AmqpURL(true))
	tenv.Rabbit = r
	return tenv
}

func (tenv *TestEnvironment) getHost() string {
	url, err := url.Parse(tenv.docker.Client.Endpoint())
	if err != nil {
		tenv.t.Fatal("url to docker host could was not parsed:", err)
		return ""
	}
	return url.Hostname()
}

// CleanUp removes the container. If not called the container will run forever
func (tenv *TestEnvironment) CleanUp() {
	if err := tenv.docker.Purge(tenv.resource); err != nil {
		fmt.Fprintf(os.Stderr, "Could not purge resource: %s", err)
	}
}

//ManagementURL returns the full http url including username/password to the management api in the docker environment.
// e.g. http://guest:guest@localhost:15672
func (tenv *TestEnvironment) ManagementURL() string {
	return fmt.Sprintf("http://guest:guest@%s:%s", tenv.getHost(), tenv.resource.GetPort("15672/tcp"))
}

//AmqpURL returns the url to the rabbitmq server
// e.g. amqp://localhost:5672
func (tenv *TestEnvironment) AmqpURL(withCred bool) string {
	return fmt.Sprintf("amqp://%s:%s", tenv.getHost(), tenv.resource.GetPort("5672/tcp"))
}

// GetURL fetches the url. Will return error.
func GetURL(url string, timeout time.Duration) (string, error) {
	maxTime := time.Duration(timeout)
	client := http.Client{
		Timeout: maxTime,
	}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	return string(body), err
}

// GetOrDie fetches the url. Will log.Fatal on error/timeout.
func GetOrDie(url string, timeout time.Duration) string {
	body, err := GetURL(url, timeout)
	if err != nil {
		log.Fatalf("Failed to get url in time: %s", err)
	}
	return body
}

// MustSetPolicy adds a policy "name" to the default vhost. On error it will log.Fatal
func (tenv *TestEnvironment) MustSetPolicy(name string, pattern string) {
	policy := fmt.Sprintf(`{"pattern":"%s", "definition": {"ha-mode":"all"}, "priority":0, "apply-to": "all"}`, pattern)
	url := fmt.Sprintf("%s/api/policies/%%2f/%s", tenv.ManagementURL(), name)

	client := &http.Client{}
	request, err := http.NewRequest("PUT", url, strings.NewReader(policy))
	if err != nil {
		log.Fatal(fmt.Errorf("could not create NewRequst: %w", err))
	}
	request.Header.Add("Content-Type", "application/json")
	request.ContentLength = int64(len(policy))
	response, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	} else {
		response.Body.Close()
	}
}
