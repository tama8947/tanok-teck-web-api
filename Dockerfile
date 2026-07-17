FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

ENV GOTOOLCHAIN=local

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/bin/api ./cmd/api

FROM alpine:3.21 AS runner

RUN apk add --no-cache ca-certificates tzdata wget

RUN addgroup -S -g 1000 appgroup \
 && adduser -S -u 1000 -G appgroup appuser

WORKDIR /app

COPY --from=builder /app/bin/api .

USER appuser

EXPOSE 4733

CMD ["./api"]
