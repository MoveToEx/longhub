FROM golang:1.26.0-alpine3.23 AS deps

WORKDIR /app
COPY go.mod go.sum ./

RUN go mod download

FROM deps AS builder

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build

FROM alpine:3.23 AS runner

WORKDIR /app
COPY --from=builder /app/long ./

CMD ["./long"]