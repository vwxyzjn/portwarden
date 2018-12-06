FROM ubuntu:latest

# Install Go
RUN apt-get update && apt-get install -y wget git gcc unzip
RUN wget -q -P /tmp https://dl.google.com/go/go1.11.2.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf /tmp/go1.11.2.linux-amd64.tar.gz
RUN rm /tmp/go1.11.2.linux-amd64.tar.gz
ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

# Setup work directory
COPY . /go/src/github.com/vwxyzjn/portwarden
WORKDIR /go/src/github.com/vwxyzjn/portwarden

# Install Go Dep
RUN wget -q https://github.com/golang/dep/releases/download/v0.4.1/dep-linux-amd64
RUN mv dep-linux-amd64 /usr/bin/dep
RUN chmod +x /usr/bin/dep

# Install Bitwarden CLI
RUN wget -q https://github.com/bitwarden/cli/releases/download/v1.6.0/bw-linux-1.6.0.zip
RUN unzip bw-linux-1.6.0.zip -d /usr/bin/
RUN chmod +x /usr/bin/bw

# Ready to run
EXPOSE 5000
