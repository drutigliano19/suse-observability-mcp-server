# --- Build Stage ---
# Use a specific Go version from your go.mod file for reproducible builds.
FROM golang:1.24-alpine AS build

# Set the working directory inside the container.
WORKDIR /app

# Copy go.mod and go.sum files to download dependencies.
# This is done as a separate step to leverage Docker's layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application's source code.
COPY . .

# Build the application.
# CGO_ENABLED=0 creates a static binary.
# -o specifies the output file name.
RUN CGO_ENABLED=0 go build -o /suse-observability-mcp-server cmd/server/main.go

# --- Final Stage ---
# Use a minimal, non-root base image for the final container.
# scratch is the most minimal image, containing only the application binary.
FROM scratch

# Copy the static binary from the build stage.
COPY --from=build /suse-observability-mcp-server /suse-observability-mcp-server

# Expose the port that the HTTP server will listen on, as mentioned in the README.
EXPOSE 8080

# Set the entrypoint for the container.
# This allows you to pass command-line arguments when running the container.
# Example: docker run <image> -http :8080 -url "..." -token "..."
ENTRYPOINT ["/suse-observability-mcp-server"]