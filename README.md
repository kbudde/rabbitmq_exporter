# RabbitMQ Exporter [![Build Status](https://travis-ci.org/kbudde/rabbitmq_exporter.svg?branch=master)](https://travis-ci.org/kbudde/rabbitmq_exporter) [![Coverage Status](https://coveralls.io/repos/kbudde/rabbitmq_exporter/badge.svg?branch=master)](https://coveralls.io/r/kbudde/rabbitmq_exporter?branch=master)

Prometheus exporter for RabbitMQ metrics, based on RabbitMQ HTTP API.

### Dependencies

* Prometheus [client](https://github.com/prometheus/client_golang) for Golang
* [Logging](https://github.com/Sirupsen/logrus)

### Setting up locally rabbitMQ and exporter with docker

1. Start rabbitMQ

        $ docker run -d -e RABBITMQ_NODENAME=my-rabbit --name my-rabbit -p 15672:15672 -p 9090:9090 rabbitmq:3-management

2. Start rabbitmq_exporter in container.

        $ docker run -d --net=container:my-rabbit kbudde/rabbitmq-exporter

Now your metrics are available through [http://localhost:9090/metrics](http://localhost:9090/metrics).

The rabbitmq_exporter is sharing the network interface with the rabbitmq container -> it is possible to use localhost and default user/password.
Disadvantage: you have to publish the port (9090) in the rabbitmq container.

### Configuration

Rabbitmq_exporter uses environment variables for configuration.
Settings:

* RABBIT_URL:      "http://localhost:15672",
* RABBIT_USER:     "guest",
* RABBIT_PASSWORD: "guest",
* PUBLISH_PORT:    "9090",
* OUTPUT_FORMAT:   "TTY", //change to JSON if needed
* LOG_LEVEL:       "info", // can be "debug", "info", "warning", "error", "fatal", or "panic"

Example

    OUTPUT_FORMAT=JSON PUBLISH_PORT=9099 ./rabbitmq_exporter

### Metrics

#### Global 


metric | description
-------| ------------
|up | Was the last scrape of rabbitmq successful.|
|channelsTotal | Total number of open channels|
|connectionsTotal | Total number of open connections|
|consumersTotal | Total number of message consumers|
|queuesTotal | Total number of queues in use|
|exchangesTotal | Total number of exchanges in use|


#### Queues

Labels: vhost, queue

##### Gauge

metric | description
-------| ------------
|queue_messages_ready|Number of messages ready to be delivered to clients.|
|queue_messages_unacknowledged|Number of messages delivered to clients but not yet acknowledged.|
|queue_messages|Sum of ready and unacknowledged messages (queue depth).|
|queue_messages_ready_ram|Number of messages from messages_ready which are resident in ram.|
|queue_messages_unacknowledged_ram|Number of messages from messages_unacknowledged which are resident in ram.|
|queue_messages_ram|Total number of messages which are resident in ram.|
|queue_messages_persistent|Total number of persistent messages in the queue (will always be 0 for transient queues).|
|queue_message_bytes|Sum of the size of all message bodies in the queue. This does not include the message properties (including headers) or any overhead.|
|queue_message_bytes_ready|Like message_bytes but counting only those messages ready to be delivered to clients.|
|queue_message_bytes_unacknowledged|Like message_bytes but counting only those messages delivered to clients but not yet acknowledged.|
|queue_message_bytes_ram|Like message_bytes but counting only those messages which are in RAM.|
|queue_message_bytes_persistent|Like message_bytes but counting only those messages which are persistent.|
|queue_consumers|Number of consumers.|
|queue_consumer_utilisation|Fraction of the time (between 0.0 and 1.0) that the queue is able to immediately deliver messages to consumers. This can be less than 1.0 if consumers are limited by network congestion or prefetch count.|
|queue_memory|Bytes of memory consumed by the Erlang process associated with the queue, including stack, heap and internal structures.|

##### Counter

metric | description
-------| ------------
|queue_disk_reads|Total number of times messages have been read from disk by this queue since it started.|
|queue_disk_writes|Total number of times messages have been written to disk by this queue since it started.|
|queue_messages_published_total|Count of messages published.|
|queue_messages_confirmed_total|Count of messages confirmed. |
|queue_messages_delivered_total|Count of messages delivered in acknowledgement mode to consumers.|
|queue_messages_delivered_noack_total|Count of messages delivered in no-acknowledgement mode to consumers. |
|queue_messages_get_total|Count of messages delivered in acknowledgement mode in response to basic.get.|
|queue_messages_get_noack_total|Count of messages delivered in no-acknowledgement mode in response to basic.get.|
|queue_messages_redelivered_total|Count of subset of messages in deliver_get which had the redelivered flag set.|
|queue_messages_returned_total|Count of messages returned to publisher as unroutable.|

#### Exchanges - Counter

Labels: vhost, exchange

metric | description
-------| ------------
|exchange_messages_published_total|Count of messages published.|
|exchange_messages_published_in_total|Count of messages published in to an exchange, i.e. not taking account of routing.|
|exchange_messages_published_out_total|Count of messages published out of an exchange, i.e. taking account of routing.|
|exchange_messages_confirmed_total|Count of messages confirmed. |
|exchange_messages_delivered_total|Count of messages delivered in acknowledgement mode to consumers.|
|exchange_messages_delivered_noack_total|Count of messages delivered in no-acknowledgement mode to consumers. |
|exchange_messages_get_total|Count of messages delivered in acknowledgement mode in response to basic.get.|
|exchange_messages_get_noack_total|Count of messages delivered in no-acknowledgement mode in response to basic.get.|
|exchange_messages_ack_total|Count of messages delivered in acknowledgement mode in response to basic.get.|
|exchange_messages_redelivered_total|Count of subset of messages in deliver_get which had the redelivered flag set.|
|exchange_messages_returned_total|Count of messages returned to publisher as unroutable.|
	
## Docker

To create a docker image locally it is recommened to use the Makefile.
Promu is used for building the statically linked binary which is added to the scratch image.

       make docker
