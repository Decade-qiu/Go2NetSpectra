# Stage 1: Build the Go binary
FROM golang:1.25-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to download dependencies
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the entire source code
COPY . .

# Build the ns-api binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /ns-api ./cmd/ns-api

# Stage 2: Create the final lightweight image
FROM alpine:latest

# Copy the binary from the builder stage
COPY --from=builder /ns-api /ns-api

# Copy the configuration file
COPY configs/config.docker.yaml /configs/config.yaml

# Expose the port the API server listens on
EXPOSE 8080

# Command to run the application
ENTRYPOINT ["/ns-api"]
