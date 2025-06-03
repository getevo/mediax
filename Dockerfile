# Pre Build stage
FROM golang:1.23.5-alpine AS builder-base
RUN apk add --no-cache build-base imagemagick-dev imagemagick
ENV CGO_ENABLED=1
ENV CGO_CFLAGS_ALLOW=-Xpreprocessor


# Build Stage
FROM builder-base AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .


RUN go build -o mediax


# Pre Runtime stage
FROM alpine:3.19 AS runtime


# Install only the runtime dependencies
RUN apk add --no-cache imagemagick libjpeg-turbo libgcc libstdc++ libwebp-tools libwebp


FROM runtime
WORKDIR /app
RUN apk add --no-cache imagemagick libjpeg-turbo libgcc libstdc++ libwebp-tools libwebp
# Copy the binary and any needed files
COPY --from=builder /app/mediax .
COPY --from=builder /app/config.yml .

# Make sure binary is executable
RUN chmod +x ./mediax

CMD ["./mediax"]