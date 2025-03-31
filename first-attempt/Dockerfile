# Use the official Golang image to build the binary
FROM golang:1.20 as builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the Go binary
RUN go build -o ffprobe-shim ./cmd/ffprobe-shim

# Use a minimal base image to run the binary
FROM debian:bullseye-slim

# Set the working directory inside the container
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/ffprobe-shim .

# Ensure ffprobe-real is available in the container
# Replace this with the actual installation or copy of ffprobe-real
RUN apt-get update && apt-get install -y ffmpeg && apt-get clean

# Expose the binary as the entrypoint
ENTRYPOINT ["./ffprobe-shim"]