package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func expect(t *testing.T, got interface{}, expected interface{}) {
	t.Helper()
	if got != expected {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", expected, reflect.TypeOf(expected), got, reflect.TypeOf(got))
	}
}

func createTestserver(result int, answer string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(result)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, answer)
	}))
}

func TestGetMetricMap(t *testing.T) {
	// Test server that always responds with 200 code, and specific payload
	server := createTestserver(200, `{"nonFloat":"bob@example.com","float1":1.23456789101112,"number":2}`)
	defer server.Close()

	config := &rabbitExporterConfig{
		RabbitURL: server.URL,
	}

	overview, _ := getMetricMap(*config, "overview")

	expect(t, len(overview), 2)
	expect(t, overview["float1"], 1.23456789101112)
	expect(t, overview["number"], 2.0)

	//Unknown error Server
	errorServer := createTestserver(500, http.StatusText(500))
	defer errorServer.Close()

	config = &rabbitExporterConfig{
		RabbitURL: errorServer.URL,
	}

	overview, _ = getMetricMap(*config, "overview")

	expect(t, len(overview), 0)
}

func TestQueues(t *testing.T) {
	// Test server that always responds with 200 code, and specific payload
	server := createTestserver(200, `[{"name":"Queue1","nonFloat":"bob@example.com","float1":1.23456789101112,"number":2},{"name":"Queue2","vhost":"Vhost2","nonFloat":"bob@example.com","float1":3.23456789101112,"number":3}]`)
	defer server.Close()

	config := &rabbitExporterConfig{
		RabbitURL: server.URL,
	}

	queues, err := getStatsInfo(*config, "queues", queueLabelKeys)
	expect(t, err, nil)
	expect(t, len(queues), 2)
	expect(t, queues[0].labels["name"], "Queue1")
	expect(t, queues[0].labels["vhost"], "")
	expect(t, queues[1].labels["name"], "Queue2")
	expect(t, queues[1].labels["vhost"], "Vhost2")
	expect(t, len(queues[0].metrics), 2)
	expect(t, len(queues[1].metrics), 2)
	expect(t, queues[0].metrics["float1"], 1.23456789101112)
	expect(t, queues[1].metrics["float1"], 3.23456789101112)
	expect(t, queues[0].metrics["number"], 2.0)
	expect(t, queues[1].metrics["number"], 3.0)

	//Unknown error Server
	errorServer := createTestserver(500, http.StatusText(500))
	defer errorServer.Close()

	config = &rabbitExporterConfig{
		RabbitURL: errorServer.URL,
	}

	queues, err = getStatsInfo(*config, "queues", queueLabelKeys)
	if err == nil {
		t.Errorf("Request failed. An error was expected but not found")
	}
	expect(t, len(queues), 0)
}

func TestExchanges(t *testing.T) {

	// Test server that always responds with 200 code, and specific payload
	server := createTestserver(200, exchangeAPIResponse)
	defer server.Close()

	config := &rabbitExporterConfig{
		RabbitURL: server.URL,
	}

	exchanges, err := getStatsInfo(*config, "exchanges", exchangeLabelKeys)
	expect(t, err, nil)
	expect(t, len(exchanges), 9)
	expect(t, exchanges[0].labels["name"], "")
	expect(t, exchanges[0].labels["vhost"], "/")
	expect(t, exchanges[1].labels["name"], "amq.direct")
	expect(t, exchanges[1].labels["vhost"], "/")
	expect(t, len(exchanges[0].metrics), 3)
	expect(t, len(exchanges[1].metrics), 3)

	expect(t, exchanges[8].labels["name"], "myExchange")
	expect(t, exchanges[8].labels["vhost"], "/")
	expect(t, exchanges[8].metrics["message_stats.confirm"], 5.0)
	expect(t, exchanges[8].metrics["message_stats.publish_in"], 5.0)
	expect(t, exchanges[8].metrics["message_stats.ack"], 0.0)
	expect(t, exchanges[8].metrics["message_stats.return_unroutable"], 5.0)

	//Unknown error Server
	errorServer := createTestserver(500, http.StatusText(500))
	defer errorServer.Close()

	config = &rabbitExporterConfig{
		RabbitURL: errorServer.URL,
	}

	exchanges, err = getStatsInfo(*config, "exchanges", exchangeLabels)
	if err == nil {
		t.Errorf("Request failed. An error was expected but not found")
	}
	expect(t, len(exchanges), 0)
}

func TestNoSort(t *testing.T) {
	assertNoSortRespected(t, false)
	assertNoSortRespected(t, true)
}

func assertNoSortRespected(t *testing.T, enabled bool) {
	var args string
	if enabled {
		args = "?sort="
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if r.RequestURI == "/api/overview"+args {
			fmt.Fprintln(w, `{"nonFloat":"bob@example.com","float1":1.23456789101112,"number":2}`)
		} else {
			t.Errorf("Invalid request with enabled=%t. URI=%v", enabled, r.RequestURI)
			fmt.Fprintf(w, "Invalid request. URI=%v", r.RequestURI)
		}

	}))
	defer server.Close()

	config := &rabbitExporterConfig{
		RabbitURL:          server.URL,
		RabbitCapabilities: rabbitCapabilitySet{rabbitCapNoSort: enabled},
	}

	if _, err := getMetricMap(*config, "overview"); err != nil {
		t.Errorf("Error getting overview: %v", err)
	}
}
