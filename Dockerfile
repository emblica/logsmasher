FROM golang:1.10 AS builder

# Download and install the latest release of dep
RUN curl -L https://github.com/golang/dep/releases/download/v0.4.1/dep-linux-amd64 -o /usr/bin/dep
RUN chmod +x /usr/bin/dep

# Copy the code from the host and compile it
WORKDIR $GOPATH/src/github.com/emblica/logsmasher
COPY Gopkg.toml Gopkg.lock ./
RUN dep ensure --vendor-only
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o /app .

FROM scratch
COPY /res ./res
COPY --from=builder /app ./
ENTRYPOINT ["./app"]
