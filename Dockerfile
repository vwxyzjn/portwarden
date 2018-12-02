FROM golang:1.11.2-alpine3.8

# Setup work directory
COPY . /go/src/github.com/vwxyzjn/portwarden
WORKDIR /go/src/github.com/vwxyzjn/portwarden/web/scheduler

# Install Go Dep
RUN wget -q https://github.com/golang/dep/releases/download/v0.4.1/dep-linux-amd64
RUN mv dep-linux-amd64 /usr/bin/dep
RUN chmod +x /usr/bin/dep

# Run dep
# Notice git is the dependency for running dep
RUN apk add --no-cache bash git openssh
RUN cd /go/src/github.com/vwxyzjn/portwarden && dep ensure --vendor-only

# Ready to run
EXPOSE 5000
