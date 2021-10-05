# Makefile
.DEFAULT_GOAL := all

deps:
	@go mod tidy \
		&& mkdir -p bin

cinitd-static: deps
	@cd cinitd \
		&& CGO_ENABLED=1 GOOS=linux go build \
			-a -installsuffix cgo \
			-ldflags "-X main.sver=${VERSION} -linkmode external -extldflags -static" \
			-o ../bin/cinit-daemon cinitd.go \
		&& cd ..

cinitd: deps
	@cd cinitd \
		&& CGO_ENABLED=1 GOOS=linux go build \
			-a -installsuffix cgo \
			-ldflags "-X main.sver=${VERSION}" \
			-o ../bin/cinit-daemon cinitd.go \
		&& cd ..

cli-static: deps
	@cd cli \
		&& CGO_ENABLED=1 GOOS=linux go build \
			-a -installsuffix cgo \
			-ldflags "-X main.sver=${VERSION} -linkmode external -extldflags -static" \
			-o ../bin/cinit cli.go \
		&& cd ..

cli: deps
	@cd cli \
		&& CGO_ENABLED=1 GOOS=linux go build \
			-a -installsuffix cgo \
			-ldflags "-X main.sver=${VERSION}" \
			-o ../bin/cinit cli.go \
		&& cd ..

.PHONY: all
all: cinitd cli

docker:
	@docker-compose -f docker-compose.yml down \
		&& docker-compose -f docker-compose.yml build \
		&& docker-compose -f docker-compose.yml up -d \
		&& docker logs -f debug

.PHONY: run
run:
	go run cinitd/cinitd.go -dev