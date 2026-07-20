# syntax=docker/dockerfile:1
# Build stage
FROM golang:1-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /gladiator .

# Runtime stage
FROM gcr.io/distroless/static-debian12

ARG BUILD_DATE
ARG VERSION
ARG GIT_COMMIT

LABEL org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${GIT_COMMIT}"

COPY --from=builder /gladiator /gladiator

VOLUME /data

EXPOSE 2137 9999

ENTRYPOINT ["/gladiator"]
