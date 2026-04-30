FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod .
# 如果有 go.sum 请取消注释
# COPY go.sum .
RUN go mod download
COPY . .
ARG SERVICE_NAME
RUN go build -o main ./\

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
EXPOSE 9000
CMD ["./main"]
