FROM golang:1.25-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG BUILD_VERSION=dev
ARG BUILD_COMMIT=unknown
ARG BUILD_TIME=unknown
RUN go build -ldflags "-X main.version=${BUILD_VERSION} -X main.commit=${BUILD_COMMIT} -X main.buildTime=${BUILD_TIME}" -o /bin/boxengage-api ./cmd/api
RUN go build -o /bin/engagefit-migrate ./cmd/migrate
RUN go build -o /bin/engagefit-rotate-secrets ./cmd/rotate-secrets
RUN go build -o /bin/engagefit-privacy-retention ./cmd/privacy-retention

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S engagefit \
    && adduser -S -G engagefit engagefit

COPY --from=build --chown=engagefit:engagefit /bin/boxengage-api /usr/local/bin/engagefit-api
COPY --from=build --chown=engagefit:engagefit /bin/engagefit-migrate /usr/local/bin/engagefit-migrate
COPY --from=build --chown=engagefit:engagefit /bin/engagefit-rotate-secrets /usr/local/bin/engagefit-rotate-secrets
COPY --from=build --chown=engagefit:engagefit /bin/engagefit-privacy-retention /usr/local/bin/engagefit-privacy-retention

USER engagefit

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null "http://127.0.0.1:${PORT:-${HTTP_PORT:-8080}}/health/live" || exit 1

CMD ["/usr/local/bin/engagefit-api"]
