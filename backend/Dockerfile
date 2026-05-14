FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/cadenza ./cmd/cadenza/

FROM alpine:3.21

RUN apk add --no-cache ca-certificates && \
    adduser -D -g '' cadenza

COPY --from=builder /bin/cadenza /usr/local/bin/cadenza

WORKDIR /app
RUN mkdir -p /app/output && chown cadenza:cadenza /app/output

USER cadenza

ENTRYPOINT ["cadenza"]
CMD ["--bpm", "122", "--key", "Am", "--no-llm"]
