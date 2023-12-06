# Build Stage
FROM golang:1.21.5 as builder

# Set the working directory
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o medic .

# Final Stage
FROM gcr.io/distroless/base-debian11

# Copy the binary from the builder stage
COPY --from=builder /app/medic /

# Command to run
ENTRYPOINT ["/medic"]
