FROM ubuntu:latest as builder

# Install Go
RUN apt-get update && apt-get install -y wget git gcc unzip
RUN wget -q -P /tmp https://dl.google.com/go/go1.11.2.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf /tmp/go1.11.2.linux-amd64.tar.gz
RUN rm /tmp/go1.11.2.linux-amd64.tar.gz
ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
ARG Salt
ENV Salt $Salt
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

# Setup work directory
COPY . /go/src/github.com/vwxyzjn/portwarden
WORKDIR /go/src/github.com/vwxyzjn/portwarden

# Install Go Dep
RUN wget -q https://github.com/golang/dep/releases/download/v0.4.1/dep-linux-amd64
RUN mv dep-linux-amd64 /usr/bin/dep
RUN chmod +x /usr/bin/dep

# Install Bitwarden CLI
RUN wget -q https://ci.appveyor.com/api/buildjobs/pdartqa5k00a5b23/artifacts/dist%2Flinux%2Fbw -O /usr/bin/bw
RUN chmod +x /usr/bin/bw

# Run dep
# Notice git is the dependency for running dep
RUN cd /go/src/github.com/vwxyzjn/portwarden && dep ensure --vendor-only
RUN go run /go/src/github.com/vwxyzjn/portwarden/utils/generate_salt_file.go
RUN go build /go/src/github.com/vwxyzjn/portwarden/web/worker/main.go && mv ./main /worker
RUN go build /go/src/github.com/vwxyzjn/portwarden/web/scheduler/main.go && mv ./main /scheduler

# Ready to run
EXPOSE 5000

FROM debian:stretch-20181112 as production
RUN apt-get update && apt-get install -y ca-certificates openssl
COPY --from=builder /usr/bin/bw /usr/bin/bw
COPY --from=builder /scheduler /go/src/github.com/vwxyzjn/portwarden/web/scheduler/scheduler
COPY --from=builder /worker /go/src/github.com/vwxyzjn/portwarden/web/worker/worker
COPY --from=builder /go/src/github.com/vwxyzjn/portwarden/web/portwardenCredentials.json /go/src/github.com/vwxyzjn/portwarden/web/portwardenCredentials.json
RUN chmod +x /go/src/github.com/vwxyzjn/portwarden/web/scheduler/scheduler
RUN chmod +x /go/src/github.com/vwxyzjn/portwarden/web/worker/worker
WORKDIR /go/src/github.com/vwxyzjn/portwarden
EXPOSE 5000
