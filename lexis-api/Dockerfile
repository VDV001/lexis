FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /api ./cmd/api

FROM alpine:3.21
RUN apk --no-cache add ca-certificates
COPY --from=builder /api /api
COPY migrations /migrations
EXPOSE 8080
CMD ["/api"]
