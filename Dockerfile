# Stage 1: Build the Go application
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy Go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux go build -o /go-app .

# Stage 2: Setup PostgreSQL and final image
FROM postgres:14-alpine

# Copy the built Go application from the builder stage
COPY --from=builder /go-app /usr/local/bin/go-app

# Expose PostgreSQL port (default 5432)
EXPOSE 5432

# Note: Exposing the Go application's port will depend on the application itself.
# For example, if the Go app listens on port 8080:
# EXPOSE 8080

# Set the entrypoint or command for the Go application
# This will also depend on how the Go application is intended to be run.
# For example, if it's a simple executable:
# CMD ["/usr/local/bin/go-app"]

# If the Go application needs to be run alongside PostgreSQL,
# a custom entrypoint script might be needed to start both services.
# For now, we'll assume the Go app is the primary process.
# If PostgreSQL needs to be started explicitly, that would be handled by the user
# or a docker-compose setup. This Dockerfile focuses on getting the Go app
# into an image with PostgreSQL available.

# Default command for postgres image will start postgres server.
# To run the go app, the user will need to override the command or use an entrypoint script.
# For simplicity in this step, we are not creating a custom entrypoint script.
# The primary goal is to have both components in the image.
