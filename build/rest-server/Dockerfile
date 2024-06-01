# First Stage: Builder
FROM golang:1.21-alpine AS builder
# Install necessary dependencies
RUN apk --no-cache add \
    alpine-sdk \
    librdkafka-dev \
    pkgconf && \
    rm -rf /var/cache/apk/*

WORKDIR /build

# Copy the source code
COPY . .

# Download Go modules
RUN go mod download

# Build the rest-server
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -ldflags "-extldflags -static" -tags musl \
    github.com/sanLimbu/todo-api/cmd/rest-server

# Install the migrate tool
RUN CGO_ENABLED=0 GOOS=linux go install -a -installsuffix cgo -ldflags "-s -w" -tags 'postgres' \
    github.com/golang-migrate/migrate/v4/cmd/migrate@v4.14.1

# Second Stage: Certificates
FROM alpine:3.13 AS certificates

# Install ca-certificates
RUN apk --no-cache add ca-certificates

# Final Stage: Scratch
FROM scratch

WORKDIR /api/
ENV PATH=/api/bin/:$PATH

# Copy the ca-certificates
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy the binaries and configuration
COPY --from=builder /build/rest-server ./bin/rest-server
COPY --from=builder /go/bin/migrate ./bin/migrate
COPY --from=builder /build/env.example .
COPY --from=builder /build/db/ .

# Expose the application port
EXPOSE 9234

# Set the entrypoint
CMD ["rest-server", "-env", "/api/env.example"]
