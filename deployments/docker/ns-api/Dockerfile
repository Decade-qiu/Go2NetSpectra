# Stage 1: Build the Go binary
FROM golang:1.25-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to download dependencies
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the entire source code
# === Why not COPY . . in one step? ===
# To leverage Docker layer caching effectively, we first copy only the go.mod and go.sum files
# and run 'go mod download'. This way, if only the source code changes but the dependencies remain the same,
# Docker can cache the layer with downloaded dependencies, speeding up subsequent builds.
# After that, we copy the rest of the source code.
# ====================================== 
COPY . .

# Build the ns-api binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /ns-api ./cmd/ns-api/v2

# Stage 2: Create the final lightweight image 
# The image of Stage 1 will not be included in the final image
FROM alpine:latest

# Copy the binary from the builder stage
COPY --from=builder /ns-api /ns-api

# Expose the port the API server listens on
EXPOSE 8080
EXPOSE 50051

# Command to run the application
ENTRYPOINT ["/ns-api"]
