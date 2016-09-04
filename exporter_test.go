package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	overviewTestData    = `{"management_version":"3.5.1","rates_mode":"basic","exchange_types":[{"name":"topic","description":"AMQP topic exchange, as per the AMQP specification","enabled":true},{"name":"fanout","description":"AMQP fanout exchange, as per the AMQP specification","enabled":true},{"name":"direct","description":"AMQP direct exchange, as per the AMQP specification","enabled":true},{"name":"headers","description":"AMQP headers exchange, as per the AMQP specification","enabled":true}],"rabbitmq_version":"3.5.1","cluster_name":"my-rabbit@ae74c041248b","erlang_version":"17.5","erlang_full_version":"Erlang/OTP 17 [erts-6.4] [source] [64-bit] [smp:2:2] [async-threads:30] [kernel-poll:true]","message_stats":{},"queue_totals":{"messages":48,"messages_details":{"rate":0.0},"messages_ready":48,"messages_ready_details":{"rate":0.0},"messages_unacknowledged":0,"messages_unacknowledged_details":{"rate":0.0}},"object_totals":{"consumers":0,"queues":4,"exchanges":8,"connections":0,"channels":0},"statistics_db_event_queue":0,"node":"my-rabbit@ae74c041248b","statistics_db_node":"my-rabbit@ae74c041248b","listeners":[{"node":"my-rabbit@ae74c041248b","protocol":"amqp","ip_address":"::","port":5672},{"node":"my-rabbit@ae74c041248b","protocol":"clustering","ip_address":"::","port":25672}],"contexts":[{"node":"my-rabbit@ae74c041248b","description":"RabbitMQ Management","path":"/","port":"15672"}]}`
	queuesTestData      = `[{"memory":16056,"message_stats":{"disk_writes":6,"disk_writes_details":{"rate":0.4},"publish":6,"publish_details":{"rate":0.4}},"messages":6,"messages_details":{"rate":0.4},"messages_ready":6,"messages_ready_details":{"rate":0.4},"messages_unacknowledged":0,"messages_unacknowledged_details":{"rate":0.0},"idle_since":"2015-07-07 19:02:19","consumer_utilisation":"","policy":"","exclusive_consumer_tag":"","consumers":0,"recoverable_slaves":"","state":"running","messages_ram":6,"messages_ready_ram":6,"messages_unacknowledged_ram":0,"messages_persistent":6,"message_bytes":30,"message_bytes_ready":30,"message_bytes_unacknowledged":0,"message_bytes_ram":30,"message_bytes_persistent":30,"disk_reads":0,"disk_writes":6,"backing_queue_status":{"q1":0,"q2":0,"delta":["delta","undefined",0,"undefined"],"q3":0,"q4":6,"len":6,"target_ram_count":"infinity","next_seq_id":6,"avg_ingress_rate":0.007658533940556533,"avg_egress_rate":0.0,"avg_ack_ingress_rate":0.0,"avg_ack_egress_rate":0.0},"name":"myQueue1","vhost":"/","durable":true,"auto_delete":false,"arguments":{},"node":"my-rabbit@ae74c041248b"},{"memory":55344,"message_stats":{"disk_reads":25,"disk_reads_details":{"rate":0.0}},"messages":25,"messages_details":{"rate":0.0},"messages_ready":25,"messages_ready_details":{"rate":0.0},"messages_unacknowledged":0,"messages_unacknowledged_details":{"rate":0.0},"idle_since":"2015-07-07 18:57:52","consumer_utilisation":"","policy":"","exclusive_consumer_tag":"","consumers":0,"recoverable_slaves":"","state":"running","messages_ram":25,"messages_ready_ram":25,"messages_unacknowledged_ram":0,"messages_persistent":25,"message_bytes":75,"message_bytes_ready":75,"message_bytes_unacknowledged":0,"message_bytes_ram":75,"message_bytes_persistent":75,"disk_reads":25,"disk_writes":0,"backing_queue_status":{"q1":0,"q2":0,"delta":["delta","undefined",0,"undefined"],"q3":24,"q4":1,"len":25,"target_ram_count":"infinity","next_seq_id":16384,"avg_ingress_rate":0.0,"avg_egress_rate":0.0,"avg_ack_ingress_rate":0.0,"avg_ack_egress_rate":0.0},"name":"myQueue2","vhost":"/","durable":true,"auto_delete":false,"arguments":{},"node":"my-rabbit@ae74c041248b"},{"memory":34648,"message_stats":{"disk_reads":23,"disk_reads_details":{"rate":0.0}},"messages":23,"messages_details":{"rate":0.0},"messages_ready":23,"messages_ready_details":{"rate":0.0},"messages_unacknowledged":0,"messages_unacknowledged_details":{"rate":0.0},"idle_since":"2015-07-07 18:57:52","consumer_utilisation":"","policy":"","exclusive_consumer_tag":"","consumers":0,"recoverable_slaves":"","state":"running","messages_ram":23,"messages_ready_ram":23,"messages_unacknowledged_ram":0,"messages_persistent":23,"message_bytes":207,"message_bytes_ready":207,"message_bytes_unacknowledged":0,"message_bytes_ram":207,"message_bytes_persistent":207,"disk_reads":23,"disk_writes":0,"backing_queue_status":{"q1":0,"q2":0,"delta":["delta","undefined",0,"undefined"],"q3":22,"q4":1,"len":23,"target_ram_count":"infinity","next_seq_id":16384,"avg_ingress_rate":0.0,"avg_egress_rate":0.0,"avg_ack_ingress_rate":0.0,"avg_ack_egress_rate":0.0},"name":"myQueue3","vhost":"/","durable":true,"auto_delete":false,"arguments":{},"node":"my-rabbit@ae74c041248b"},{"memory":13912,"messages":0,"messages_details":{"rate":0.0},"messages_ready":0,"messages_ready_details":{"rate":0.0},"messages_unacknowledged":0,"messages_unacknowledged_details":{"rate":0.0},"idle_since":"2015-07-07 18:57:52","consumer_utilisation":"","policy":"","exclusive_consumer_tag":"","consumers":0,"recoverable_slaves":"","state":"running","messages_ram":0,"messages_ready_ram":0,"messages_unacknowledged_ram":0,"messages_persistent":0,"message_bytes":0,"message_bytes_ready":0,"message_bytes_unacknowledged":0,"message_bytes_ram":0,"message_bytes_persistent":0,"disk_reads":0,"disk_writes":0,"backing_queue_status":{"q1":0,"q2":0,"delta":["delta","undefined",0,"undefined"],"q3":0,"q4":0,"len":0,"target_ram_count":"infinity","next_seq_id":0,"avg_ingress_rate":0.0,"avg_egress_rate":0.0,"avg_ack_ingress_rate":0.0,"avg_ack_egress_rate":0.0},"name":"myQueue4","vhost":"vhost4","durable":true,"auto_delete":false,"arguments":{},"node":"my-rabbit@ae74c041248b"}]`
	exchangeAPIResponse = `[{
    "name": "",
    "vhost": "/",
    "type": "direct",
    "durable": true,
    "auto_delete": false,
    "internal": false,
    "arguments": {}
}, {
    "name": "amq.direct",
    "vhost": "/",
    "type": "direct",
    "durable": true,
    "auto_delete": false,
    "internal": false,
    "arguments": {}
}, {
    "name": "amq.fanout",
    "vhost": "/",
    "type": "fanout",
    "durable": true,
    "auto_delete": false,
    "internal": false,
    "arguments": {}
}, {
    "name": "amq.headers",
    "vhost": "/",
    "type": "headers",
    "durable": true,
    "auto_delete": false,
    "internal": false,
    "arguments": {}
}, {
    "name": "amq.match",
    "vhost": "/",
    "type": "headers",
    "durable": true,
    "auto_delete": false,
    "internal": false,
    "arguments": {}
}, {
    "name": "amq.rabbitmq.log",
    "vhost": "/",
    "type": "topic",
    "durable": true,
    "auto_delete": false,
    "internal": true,
    "arguments": {}
}, {
    "name": "amq.rabbitmq.trace",
    "vhost": "/",
    "type": "topic",
    "durable": true,
    "auto_delete": false,
    "internal": true,
    "arguments": {}
}, {
    "name": "amq.topic",
    "vhost": "/",
    "type": "topic",
    "durable": true,
    "auto_delete": false,
    "internal": false,
    "arguments": {}
}, {
    "message_stats": {
        "publish": 0,
        "publish_details": {
            "rate": 0.0
        },
        "publish_in": 5,
        "publish_in_details": {
            "rate": 0.0
        },
        "publish_out": 0,
        "publish_out_details": {
            "rate": 0.0
        },
        "ack": 0,
        "ack_details": {
            "rate": 0.0
        },
        "deliver_get": 0,
        "deliver_get_details": {
            "rate": 0.0
        },
        "confirm": 5,
        "confirm_details": {
            "rate": 0.0
        },
        "return_unroutable": 5,
        "return_unroutable_details": {
            "rate": 0.0
        },
        "redeliver": 0,
        "redeliver_details": {
            "rate": 0.0
        }
    },
    "name": "myExchange",
    "vhost": "/",
    "type": "fanout",
    "durable": true,
    "auto_delete": false,
    "internal": false,
    "arguments": {}
}]`
)

func expectSubstring(t *testing.T, body string, substr string) {
	if !strings.Contains(body, substr) {
		t.Errorf("Substring expected but not found. Substr=%v", substr)
	}
}

func TestWholeApp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if r.RequestURI == "/api/overview" {
			fmt.Fprintln(w, overviewTestData)
		} else if r.RequestURI == "/api/queues" {
			fmt.Fprintln(w, queuesTestData)
		} else if r.RequestURI == "/api/exchanges" {
			fmt.Fprintln(w, exchangeAPIResponse)
		} else {
			t.Errorf("Invalid request. URI=%v", r.RequestURI)
			fmt.Fprintf(w, "Invalid request. URI=%v", r.RequestURI)
		}

	}))
	defer server.Close()
	os.Setenv("RABBIT_URL", server.URL)
	initConfig()

	exporter := newExporter()
	prometheus.MustRegister(exporter)

	req, _ := http.NewRequest("GET", "", nil)
	w := httptest.NewRecorder()
	prometheus.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Home page didn't return %v", http.StatusOK)
	}
	body := w.Body.String()
	// fmt.Println(body)
	expectSubstring(t, body, `rabbitmq_exchangesTotal 8`)
	expectSubstring(t, body, `rabbitmq_queue_messages_ready{queue="myQueue2",vhost="/"} 25`)
	expectSubstring(t, body, `rabbitmq_queue_message_bytes_persistent{queue="myQueue3",vhost="/"} 207`)
	expectSubstring(t, body, `rabbitmq_queue_memory{queue="myQueue4",vhost="vhost4"} 13912`)
	expectSubstring(t, body, `rabbitmq_queue_messages_published_total{queue="myQueue1",vhost="/"} 6`)
	expectSubstring(t, body, `rabbitmq_queue_disk_writes{queue="myQueue1",vhost="/"} 6`)
	expectSubstring(t, body, `rabbitmq_up 1`)
	expectSubstring(t, body, `rabbitmq_exchange_messages_published_in_total{exchange="myExchange",vhost="/"} 5`)
}
