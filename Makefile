# Build all programs:
#   make build
#
# Run manually in this order:
#   1. Start registry:
#        go run ./registry
#      Start tenant lookup:
#        TENANT_BINDINGS=alice=1,bob=2 go run ./tenant-lookup
#
#   2. Start all baseline services svc1~svc15 with SERVICE_ENV=0.
#      If DOWNSTREAM_SERVICE is not set, svc1->svc2->...->svc15 is used.
#
#   3. Start hot SET services svc3~svc8 with SERVICE_ENV=1.
#
#   4. Test from the baseline entrance. svc2 is a default boundary service:
#        curl "http://127.0.0.1:9001/?tenant=alice&n=100"
#
#      svc2 queries tenant-lookup, gets env 1, then routes to svc3 env 1.
#      svc3~svc8 stay in env 1. svc8 falls back to svc9 env 0 because svc9 is
#      not in the hot layer, and the rest of the chain stays baseline.
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
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o tenant-lookup/tenant-lookup$(EXE) ./tenant-lookup
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
