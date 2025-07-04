FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY hello-app/ .
RUN go build -o hello-app .

FROM gcr.io/distroless/static
COPY --from=builder /app/hello-app /hello-app
EXPOSE 8080
ENTRYPOINT ["/hello-app"]