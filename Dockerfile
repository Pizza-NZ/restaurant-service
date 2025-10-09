# --- Builder Stage ---
# Use a slim Go image for the builder stage
FROM golang:1.23-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy module files and download dependencies first to leverage Docker layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the server and printer-service binaries
# CGO_ENABLED=0 creates static binaries, which are ideal for minimal container images
RUN CGO_ENABLED=0 go build -o /server ./cmd/server
RUN CGO_ENABLED=0 go build -o /printer-service ./cmd/printer-service

# --- Final Stage ---
# Use a minimal "distroless" image for the final container for a smaller footprint and attack surface
FROM gcr.io/distroless/static-debian11

# Copy the compiled binaries from the builder stage
COPY --from=builder /server /server
COPY --from=builder /printer-service /printer-service

# Set the default command to run the main server application.
# This can be overridden when running the container (e.g., `docker run <image> /printer-service`)
ENTRYPOINT ["/server"]
