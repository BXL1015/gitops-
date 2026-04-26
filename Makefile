# Build all 16 programs:
#   make build
#
# Run manually in this order:
#   1. Start registry:
#        go run ./registry
#
#   2. Start baseline services, for example:
#        SERVICE_NAME=svc15 SERVICE_ENV=0 LISTEN_ADDR=:9015 PUBLIC_ADDR=http://127.0.0.1:9015 REGISTRY_ADDR=http://127.0.0.1:8080 go run ./svc15
#        SERVICE_NAME=svc14 SERVICE_ENV=0 DOWNSTREAM_SERVICE=svc15 LISTEN_ADDR=:9014 PUBLIC_ADDR=http://127.0.0.1:9014 REGISTRY_ADDR=http://127.0.0.1:8080 go run ./svc14
#
#   3. Start a lane instance, for example svc14 in env 1:
#        SERVICE_NAME=svc14 SERVICE_ENV=1 DOWNSTREAM_SERVICE=svc15 LISTEN_ADDR=:9114 PUBLIC_ADDR=http://127.0.0.1:9114 REGISTRY_ADDR=http://127.0.0.1:8080 go run ./svc14
#
#   4. Test "one lane until fallback, then baseline from there":
#      because you call svc14 env 1 directly, svc14 uses its own SERVICE_ENV=1
#      to look for svc15 env 1. If svc15 env 1 is absent, registry returns
#      svc15 env 0. After that, svc15 is an env 0 process; no protocol header
#      is used to carry the original env 1.
#        curl "http://127.0.0.1:9114/?n=100"
#
# Memory tip:
#   Run with a small Go memory target if you want to stress low-footprint behavior:
#        GOMEMLIMIT=32MiB GOGC=50 <the command above>

GOFLAGS := -trimpath
LDFLAGS := -s -w

EXE :=
ifeq ($(OS),Windows_NT)
EXE := .exe
endif

.PHONY: build clean test

build:
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o registry/registry$(EXE) ./registry
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o svc1/svc1$(EXE) ./svc1
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o svc2/svc2$(EXE) ./svc2
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o svc3/svc3$(EXE) ./svc3
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o svc4/svc4$(EXE) ./svc4
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o svc5/svc5$(EXE) ./svc5
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o svc6/svc6$(EXE) ./svc6
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o svc7/svc7$(EXE) ./svc7
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o svc8/svc8$(EXE) ./svc8
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o svc9/svc9$(EXE) ./svc9
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o svc10/svc10$(EXE) ./svc10
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o svc11/svc11$(EXE) ./svc11
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o svc12/svc12$(EXE) ./svc12
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o svc13/svc13$(EXE) ./svc13
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o svc14/svc14$(EXE) ./svc14
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o svc15/svc15$(EXE) ./svc15

test:
	go test ./...

clean:
	go clean ./...
