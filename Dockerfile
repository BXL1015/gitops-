# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS build

ARG APP=svc1
WORKDIR /src

COPY go.mod ./
COPY registry ./registry
COPY tenant-lookup ./tenant-lookup
COPY internal ./internal
COPY svc1 ./svc1
COPY svc2 ./svc2
COPY svc3 ./svc3
COPY svc4 ./svc4
COPY svc5 ./svc5
COPY svc6 ./svc6
COPY svc7 ./svc7
COPY svc8 ./svc8
COPY svc9 ./svc9
COPY svc10 ./svc10
COPY svc11 ./svc11
COPY svc12 ./svc12
COPY svc13 ./svc13
COPY svc14 ./svc14
COPY svc15 ./svc15

RUN case "$APP" in \
      registry|tenant-lookup|svc1|svc2|svc3|svc4|svc5|svc6|svc7|svc8|svc9|svc10|svc11|svc12|svc13|svc14|svc15) ;; \
      *) echo "invalid APP=$APP" >&2; exit 1 ;; \
    esac && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/app "./$APP"

FROM scratch

WORKDIR /app
COPY --from=build /out/app /app/app

EXPOSE 8080 8081 9000
ENTRYPOINT ["/app/app"]
