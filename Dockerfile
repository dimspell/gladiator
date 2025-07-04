FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY ./ .
RUN go build -o gladiator .

FROM gcr.io/distroless/static
COPY --from=builder /app/gladiator /gladiator
EXPOSE 8080
ENTRYPOINT ["/gladiator"]