# RabbitMQ Exporter [![Build Status](https://travis-ci.org/kbudde/rabbitmq_exporter.svg?branch=master)](https://travis-ci.org/kbudde/rabbitmq_exporter) [![Coverage Status](https://coveralls.io/repos/kbudde/rabbitmq_exporter/badge.svg?branch=master)](https://coveralls.io/r/kbudde/rabbitmq_exporter?branch=master) [![](https://images.microbadger.com/badges/image/kbudde/rabbitmq-exporter.svg)](http://microbadger.com/images/kbudde/rabbitmq-exporter "Get your own image badge on microbadger.com")

Prometheus exporter for RabbitMQ metrics.
Data is scraped by [prometheus](https://prometheus.io).

Please note this an unofficial plugin. There is also an official plugin from [RabbitMQ.com](https://www.rabbitmq.com/prometheus.html). See [comparison to official exporter](#comparison-to-official-exporter)

## Maintainers

* Yoel Padron ([yoelpadronglez](https://github.com/yoelpadronglez))
* Przemysław Nowak ([pnowy](https://github.com/pnowy))


## Installation

### Binary release

You can download the latest release on the [release page](https://github.com/kbudde/rabbitmq_exporter/releases).
Docker images are push to [docker hub](https://hub.docker.com/r/kbudde/rabbitmq-exporter/tags)

### docker: rabbitmq container with network sharing

The rabbitmq_exporter is sharing the network interface with the rabbitmq container -> it is possible to use localhost and default user/password (guest).

1. Start rabbitMQ

        docker run -d -e RABBITMQ_NODENAME=my-rabbit --name my-rabbit -p 9419:9419 rabbitmq:3-management

1. Start rabbitmq_exporter in container.

        docker run -d --net=container:my-rabbit kbudde/rabbitmq-exporter

Now your metrics are exposed through [http://host:9419/metrics](http://host:9419/metrics). The management plugin does not need to be exposed.

## Configuration

Rabbitmq_exporter can be configured using json config file or environment variables for configuration.

### Config file

Rabbitmq_exporter expects config file in "conf/rabbitmq.conf". If you are running the exporter in a container (docker/kubernetes) the config must be in "/conf/rabbitmq.conf"
The name of the file can be overriden with flag:

    ./rabbitmq_exporter -config-file config.example.json

You can find an example [here](config.example.json). *Note:* If you are using a config file, you must provide all values as there is no default value.

### Settings

Environment variable|default|description
--------------------|-------|------------
RABBIT_URL | <http://127.0.0.1:15672>| url to rabbitMQ management plugin (must start with http(s)://)
RABBIT_USER | guest | username for rabbitMQ management plugin. User needs monitoring tag!
RABBIT_PASSWORD | guest | password for rabbitMQ management plugin
RABBIT_CONNECTION | direct | direct or loadbalancer, strips the self label when loadbalancer
RABBIT_USER_FILE| | location of file with username (useful for docker secrets)
RABBIT_PASSWORD_FILE | | location of file with password (useful for docker secrets)
PUBLISH_PORT | 9419 | Listening port for the exporter
PUBLISH_ADDR | "" | Listening host/IP for the exporter
OUTPUT_FORMAT | TTY | Log ouput format. TTY and JSON are suported
LOG_LEVEL | info | log level. possible values: "debug", "info", "warning", "error", "fatal", or "panic"
CAFILE | ca.pem | path to root certificate for access management plugin. Just needed if self signed certificate is used. Will be ignored if the file does not exist
CERTFILE | client-cert.pem | path to client certificate used to verify the exporter's authenticity. Will be ignored if the file does not exist
KEYFILE | client-key.pem | path to private key used with certificate to verify the exporter's authenticity. Will be ignored if the file does not exist
SKIPVERIFY | false | true/0 will ignore certificate errors of the management plugin
SKIP_VHOST | ^$ |regex, matching vhost names are not exported. First performs INCLUDE_VHOST, then SKIP_VHOST. Applies to queues and exchanges
INCLUDE_VHOST | .* | regex vhost filter. Only matching vhosts are exported. Applies to queues and exchanges
INCLUDE_QUEUES | .* | regex queue filter. Just matching names are exported
SKIP_QUEUES | ^$ |regex, matching queue names are not exported (useful for short-lived rpc queues). First performed INCLUDE, after SKIP
INCLUDE_EXCHANGES | .* | regex exchange filter. (Only exchanges in matching vhosts are exported)
SKIP_EXCHANGES  | ^$ | regex, matching exchanges names are not exported. First performed INCLUDE, after SKIP
RABBIT_CAPABILITIES | bert,no_sort | comma-separated list of extended scraping capabilities supported by the target RabbitMQ server
RABBIT_EXPORTERS | exchange,node,queue | List of enabled modules. Possible modules: connections,shovel,federation,exchange,node,queue,memory
RABBIT_TIMEOUT | 30 | timeout in seconds for retrieving data from management plugin.
MAX_QUEUES | 0 | max number of queues before we drop metrics (disabled if set to 0)
EXCLUDE_METRICS | | Metric names to exclude from export. comma-seperated. e.g. "recv_oct, recv_cnt". See exporter_*.go for names

Example and recommended settings:

    SKIP_QUEUES="RPC_.*" MAX_QUEUES=5000 ./rabbitmq_exporter

### Extended RabbitMQ capabilities

Newer version of RabbitMQ can provide some features that reduce
overhead imposed by scraping the data needed by this exporter. The
following capabilities are currently supported in
`RABBIT_CAPABILITIES` env var:

* `no_sort`: By default RabbitMQ management plugin sorts results using
  the default sort order of vhost/name. This sorting overhead can be
  avoided by passing empty sort argument (`?sort=`) to RabbitMQ
  starting from version 3.6.8. This option can be safely enabled on
  earlier 3.6.X versions, but it'll not give any performance
  improvements. And it's incompatible with 3.4.X and 3.5.X.
* `bert`: Since 3.6.9 (see
   <https://github.com/rabbitmq/rabbitmq-management/pull/367>) RabbitMQ
   supports BERT encoding as a JSON alternative. Given that BERT
   encoding is implemented in C inside the Erlang VM, it's way more
   effective than pure-Erlang JSON encoding. So this greatly reduces
   monitoring overhead when we have a lot of objects in RabbitMQ.

## Comparison to official exporter

[official exporter](https://www.rabbitmq.com/prometheus.html):

- has runtime/erlang metrics
- aggregated or per-object metrics
- missing filter

This exporter:

- works also with older versions of rabbitmq
- has more configuration options/ filtering of objects
- (bad) depends on data from management interface which can be slow/delayed

probalby best solution is to use both exporters:
[comment from shamil](https://github.com/kbudde/rabbitmq_exporter/issues/156#issuecomment-631979910)

## common errors / FAQ

### msg: Error while retrieving data from rabbitHost statusCode: 500

This exporter expects capabilities from rabbitmq 3.6.8 or newer by default.
If you are running older than 3.6.8 you must disable bert and no_sort with the setting RABBIT_CAPABILITIES=compat.
If you are running 3.13.0 or newer you must disable no_sort with the setting RABBIT_CAPABILITIES=no_sort.

### missing data in graphs

If there is a load balancer between the exporter and the RabbitMQApi, the setting `RABBIT_CONNECTION=loadbalancer` must be activated.
See https://github.com/kbudde/rabbitmq_exporter/issues/131 for details.

## build and test

This project uses goreleaser to build and release the project. You can build the project with the following command:

    goreleaser build --snapshot

`go build` will also work, but it will not include the version information, uses cgo, etc.

To run the tests, use the following command:

    go test -v ./...

If you have docker installed, you can run the tests with the following command:

    go test -v ./... --tags integration

This will start a rabbitmq container and run the tests against it.

## Metrics

The metrics are documented in the [metrics.md](metrics.md) file.
