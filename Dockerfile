FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o forum ./cmd/web

FROM alpine:3.19

RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /app
COPY --from=builder /app/forum .
COPY --from=builder /app/ui ./ui

EXPOSE 443 80

CMD ["./forum"]
