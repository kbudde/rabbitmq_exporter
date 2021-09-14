# RabbitMQ Exporter [![Build Status](https://travis-ci.org/kbudde/rabbitmq_exporter.svg?branch=master)](https://travis-ci.org/kbudde/rabbitmq_exporter) [![Coverage Status](https://coveralls.io/repos/kbudde/rabbitmq_exporter/badge.svg?branch=master)](https://coveralls.io/r/kbudde/rabbitmq_exporter?branch=master) [![](https://images.microbadger.com/badges/image/kbudde/rabbitmq-exporter.svg)](http://microbadger.com/images/kbudde/rabbitmq-exporter "Get your own image badge on microbadger.com")

Prometheus exporter for RabbitMQ metrics.
Data is scraped by [prometheus](https://prometheus.io).

Please note this an unofficial plugin. There is also an official plugin from [RabbitMQ.com](https://www.rabbitmq.com/prometheus.html). See [comparison to official exporter](#comparison-to-official-exporter)

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

You can find an example [here](config.example.json).

### Settings

Environment variable|default|description
--------------------|-------|------------
RABBIT_URL | <http://127.0.0.1:15672>| url to rabbitMQ management plugin (must start with http(s)://)
RABBIT_USER | guest | username for rabbitMQ management plugin. User needs monitoring tag!
RABBIT_PASSWORD | guest | password for rabbitMQ management plugin
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
SKIP_VHOST | ^$ |regex, matching vhost names are not exported. First performs INCLUDE_VHOST, then SKIP_VHOST
INCLUDE_VHOST | .* | regex vhost filter. Only queues in matching vhosts are exported
INCLUDE_QUEUES | .* | regex queue filter. Just matching names are exported
SKIP_QUEUES | ^$ |regex, matching queue names are not exported (useful for short-lived rpc queues). First performed INCLUDE, after SKIP
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

**Note for users of rabbmitmq < 3.6**

no_sort and bert are enabled by default. You must overwrite the default settings with:

    RABBIT_CAPABILITIES=nobert ./rabbitmq_exporter

## Metrics

All metrics (except golang/prometheus metrics) are prefixed with "rabbitmq_".

### Global

Always exported.

metric | description
-------| ------------
|up | Was the last scrape of rabbitmq successful.
|module_up | Was the last scrape of rabbitmq module successful. labels: module
|module_scrape_duration_seconds | Duration of the last scrape of rabbitmq module. labels: module
|exporter_build_info | A metric with a constant '1' value labeled by version, revision, branch and build date on which the rabbitmq_exporter was built.

### Overview

Always exported.
Labels: cluster

metric | description
-------| ------------
|channels | Number of channels|
|connections | Number of connections|
|consumers | Number of message consumers|
|queues | Number of queues in use|
|exchanges | Number of exchanges in use|
|queue_messages_global | Number ready and unacknowledged messages in cluster.|
|queue_messages_ready_global | Number of messages ready to be delivered to clients.|
|queue_messages_unacknowledged_global | Number of messages delivered to clients but not yet acknowledged.|
|version_info| A metric with a constant '1' value labeled by rabbitmq version, erlang version, node, cluster.|

### Queues

Labels: cluster, vhost, queue, durable, policy, self

#### Queues - Gauge

metric | description
-------| ------------
|queue_messages_ready|Number of messages ready to be delivered to clients.|
|queue_messages_unacknowledged|Number of messages delivered to clients but not yet acknowledged.|
|queue_messages|Sum of ready and unacknowledged messages (queue depth).|
|queue_messages_ack_total|Number of messages delivered in acknowledgement mode in response to basic.get.|
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
|queue_head_message_timestamp|The timestamp property of the first message in the queue, if present. Timestamps of messages only appear when they are in the paged-in state.|
|queue_max_length_bytes|Total body size for ready messages a queue can contain before it starts to drop them from its head.|
|queue_max_length|How many (ready) messages a queue can contain before it starts to drop them from its head.|
|queue_idle_since_seconds|starttime where the queue switched to idle state; seconds since epoch (1970); only set if queue state is idle|
|queue_reductions_total|Number of reductions which take place on this process.|
|queue_state|A metric with a value of constant '1' if the queue is in a certain state. Labels: vhost, queue, *state* (running, idle, flow,..)|
|queue_slave_nodes_len|Number of slave nodes attached to the queue|
|queue_synchronised_slave_nodes_len|Number of slave nodes in sync to the queue|

#### Queues - Counter

metric | description
-------| ------------
|queue_disk_reads_total|Total number of times messages have been read from disk by this queue since it started.|
|queue_disk_writes_total|Total number of times messages have been written to disk by this queue since it started.|
|queue_messages_published_total|Count of messages published.|
|queue_messages_confirmed_total|Count of messages confirmed. |
|queue_messages_delivered_total|Count of messages delivered in acknowledgement mode to consumers.|
|queue_messages_delivered_noack_total|Count of messages delivered in no-acknowledgement mode to consumers. |
|queue_messages_get_total|Count of messages delivered in acknowledgement mode in response to basic.get.|
|queue_messages_get_noack_total|Count of messages delivered in no-acknowledgement mode in response to basic.get.|
|queue_messages_redelivered_total|Count of subset of messages in deliver_get which had the redelivered flag set.|
|queue_messages_returned_total|Count of messages returned to publisher as unroutable.|

### Exchanges - Counter

Labels: cluster, vhost, exchange

metric | description
-------| ------------
|exchange_messages_published_in_total|Count of messages published in to an exchange, i.e. not taking account of routing.|
|exchange_messages_published_out_total|Count of messages published out of an exchange, i.e. taking account of routing.|

### Node - Counter

Labels: cluster, node, self

metric | description
-------| ------------
|uptime|Uptime in milliseconds|
|running|number of running nodes|
|node_mem_used|Memory used in bytes|
|node_mem_limit|Point at which the memory alarm will go off|
|node_mem_alarm|Whether the memory alarm has gone off|
|node_disk_free|Disk free space in bytes.|
|node_disk_free_alarm|Whether the disk alarm has gone off.|
|node_disk_free_limit|Point at which the disk alarm will go off.|
|fd_used|Used File descriptors|
|fd_available|File descriptors available|
|sockets_used|File descriptors used as sockets.|
|sockets_available|File descriptors available for use as sockets|
|partitions | Current Number of network partitions. 0 is ok. If the cluster is splitted the value is at least 2|

### Connections - Gauge

_disabled by default_. Depending on the environment and change rate it can create a high number of dead metrics. Otherwise it could be usefull and can be enabled.

Labels: cluster, vhost, node, peer_host, user, self

Please note: The data is aggregated by label values as it is possible that there are multiple connections for a certain combination of labels.

metric | description
-------| ------------
|connection_channels|number of channels in use|
|connection_received_bytes|received bytes|
|connection_received_packets|received packets|
|connection_send_bytes|send bytes|
|connection_send_packets|send packets|
|connection_send_pending|Send queue size|

Labels: vhost, node, peer_host, user, *state* (running, flow,..), self

metric | description
-------| ------------
|connection_status|Number of connections in a certain state aggregated per label combination. Metric will disappear if there are no connections in a state. |

### Shovel

_disabled by default_
Labels: cluster, vhost, shovel, type, self, state

metric | description
-------| ------------
|shovel_state|A metric with a value of constant '1' for each shovel in a certain state|

### Memory

_disabled by default_
Labels: cluster, node, self

metric | description
-------| ------------
|memory_allocated_unused_bytes|Memory preallocated by the runtime (VM allocators) but not yet used|
|memory_atom_bytes|Memory used by atoms. Should be fairly constant|
|memory_binary_bytes|Memory used by shared binary data in the runtime. Most of this memory is message bodies and metadata.)|
|memory_code_bytes|Memory used by code (bytecode, module metadata). This section is usually fairly constant and relatively small (unless the node is entirely blank and stores no data).|
|memory_connection_channels_bytes|Memory used by client connections - channels|
|memory_connection_other_bytes|Memory used by client connection - other|
|memory_connection_readers|Memory used by processes responsible for connection parser and most of connection state. Most of their memory attributes to TCP buffers|
|memory_connection_writers_bytes|Memory used by processes responsible for serialization of outgoing protocol frames and writing to client connections socktes|
|memory_mgmt_db_bytes|Management DB ETS tables + processes|
|memory_mnesia_bytes|Internal database (Mnesia) tables keep an in-memory copy of all its data (even on disc nodes)|
|memory_msg_index_bytes|Message index ETS + processes|
|memory_other_ets_bytes|Other in-memory tables besides those belonging to the stats database and internal database tables|
|memory_other_proc_bytes|Memory used by all other processes that RabbitMQ cannot categorise|
|memory_other_system_bytes|Memory used by all other system that RabbitMQ cannot categorise|
|memory_plugins_bytes|Memory used by plugins (apart from the Erlang client which is counted under Connections, and the management database which is counted separately).|
|memory_queue_procs_bytes|Memory used by class queue masters, queue indices, queue state|
|memory_queue_slave_procs_bytes|Memory used by class queue mirrors, queue indices, queue state|
|memory_reserved_unallocated_bytes|Memory preallocated/reserved by the kernel but not the runtime|
|memory_total_allocated_bytes|Node-local total memory - allocated|
|memory_total_rss_bytes|Node-local total memory - rss|
|memory_total_erlang_bytes|Node-local total memory - erlang|

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

## common errors: Error while retrieving data from rabbitHost / code 500

msg: Error while retrieving data from rabbitHost
statusCode: 500

This exporter expects capabilities from rabbitmq 3.6.8 or newer by default.
If you are running older than 3.6.8 you have to set RABBIT_CAPABILITIES=compat.
