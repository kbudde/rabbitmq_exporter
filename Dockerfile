FROM golang:alpine as builder

RUN apk add --no-cache git bash make
COPY src/* /go/src/github.com/anchorfree/rabbitmq_exporter/
COPY /testdata /go/src/github.com/anchorfree/rabbitmq_exporter/testdata
COPY /testenv /go/src/github.com/anchorfree/rabbitmq_exporter/testenv

RUN cd /go && go get -u github.com/golang/dep/cmd/dep
RUN cd /go && go get github.com/mitchellh/gox && go get github.com/axw/gocov/gocov && go get github.com/mattn/goveralls \
    && go get golang.org/x/tools/cmd/cover && go get github.com/aktau/github-release && go get github.com/Sirupsen/logrus \
    && go get github.com/kbudde/gobert && go get github.com/prometheus/client_golang/prometheus

RUN cd /go/src/github.com/anchorfree/rabbitmq_exporter/ && make build

FROM scratch
MAINTAINER Dmitry Pronkin <d.pronkin@anchorfree.com>


COPY --from=builder /go/src/github.com/anchorfree/rabbitmq_exporter/rabbitmq_exporter /

CMD ["/rabbitmq_exporter"]
