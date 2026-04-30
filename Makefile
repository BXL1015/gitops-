.PHONY: build-all build-svc

SERVICES := svc1 svc2 svc3 svc4 svc5 svc6 svc7 svc8 svc9 svc10 svc11 svc12 svc13 svc14 svc15

build-all:
@for svc in ; do \
echo "Building $dockerfilesvc..."; \
go build -o bin/$dockerfilesvc ./$dockerfilesvc; \
done

# 适配本地开发测试
run-svc%:
SERVICE_NAME=svc$* SERVICE_ENV=0 LISTEN_ADDR=:900$* go run ./svc$*
