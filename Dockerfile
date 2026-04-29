FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/llmidi-gen ./cmd/llmidi-gen/

FROM alpine:3.21

RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/llmidi-gen /usr/local/bin/llmidi-gen
COPY prompts/ /app/prompts/

WORKDIR /app
RUN mkdir -p /app/output

ENV LLMIDI_PROMPTS_DIR=/app/prompts

ENTRYPOINT ["llmidi-gen"]
CMD ["--bpm", "122", "--key", "Am", "--no-llm"]
