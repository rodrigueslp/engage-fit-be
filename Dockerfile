FROM golang:1.25-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /bin/boxengage-api ./cmd/api

FROM alpine:3.20

WORKDIR /app

COPY --from=build /bin/boxengage-api /bin/boxengage-api

EXPOSE 8080

CMD ["/bin/boxengage-api"]
