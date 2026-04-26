# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS build

WORKDIR /src

COPY go.mod ./
COPY registry ./registry
COPY tenant-lookup ./tenant-lookup
COPY internal ./internal
COPY launcher ./launcher
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

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/launcher ./launcher && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/bin/registry ./registry && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/bin/tenant-lookup ./tenant-lookup && \
    for app in svc1 svc2 svc3 svc4 svc5 svc6 svc7 svc8 svc9 svc10 svc11 svc12 svc13 svc14 svc15; do \
      CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o "/out/bin/$app" "./$app"; \
    done

FROM scratch

WORKDIR /app
COPY --from=build /out/launcher /app/launcher
COPY --from=build /out/bin /app/bin

ENV APP=registry
EXPOSE 8080 8081 9000
ENTRYPOINT ["/app/launcher"]
