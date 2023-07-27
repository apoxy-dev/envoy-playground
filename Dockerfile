FROM golang:1.18 AS go

RUN go install github.com/mccutchen/go-httpbin/v2/cmd/go-httpbin@v2.10.0

WORKDIR /app

ADD ./api /app
RUN go build
RUN go build ./cmd/run_envoy

FROM ubuntu:20.04

RUN apt-get update && apt-get install -y curl bubblewrap gpg

RUN curl -SsL https://packages.httpie.io/deb/KEY.gpg | gpg --dearmor -o /usr/share/keyrings/httpie.gpg && \
    echo "deb [arch=amd64 signed-by=/usr/share/keyrings/httpie.gpg] https://packages.httpie.io/deb ./" > /etc/apt/sources.list.d/httpie.list

RUN apt-get update && apt-get install -y httpie

RUN curl https://func-e.io/install.sh | bash -s -- -b /usr/local/bin

COPY --from=go /go/bin/go-httpbin /usr/local/bin/go-httpbin
COPY --from=go /app/envoy-playground /app/envoy-playground
COPY --from=go /app/run_envoy /app/run_envoy

WORKDIR /app

CMD ["/app/envoy-playground"]
