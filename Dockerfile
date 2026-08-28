# Pre Build stage
FROM golang:1.24-alpine AS builder-base
RUN apk add --no-cache build-base

# Build Stage
FROM builder-base AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o mediax


# Pre Runtime stage
FROM alpine:3.19 AS pre-runtime

# Install only the runtime dependencies (not dev headers)
RUN apk add --no-cache \
    imagemagick \
    libjpeg-turbo \
    libgcc \
    libstdc++ \
    libwebp-tools \
    libwebp \
    ffmpeg \
    libreoffice \
    poppler-utils


FROM pre-runtime
WORKDIR /app

# Copy the binary and any needed files
COPY --from=builder /app/mediax .
COPY docker-entrypoint.sh /app/docker-entrypoint.sh

# Make sure binary is executable
RUN chmod +x ./mediax /app/docker-entrypoint.sh

ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/app/docker-entrypoint.sh"]
