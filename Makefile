install-linter:
	go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.31.0

lint:
	golangci-lint run

test:
	go test -coverprofile=coverage.out -covermode=count ./...
	go tool cover -func=coverage.out
	rm coverage.out

functional-test: cleanup-docker
	rm -f loggo-logs.pos
	docker-compose up -d rabbit
	./build/tests --create-rabbit-queues
	timeout --preserve-status 10 ./build/loggo --logs-path="pkg/tests/fixtures/pods" --position-file-path="loggo-logs.pos" --reader-max-chunk=2 && echo "ok" || echo "bad"
	./build/tests

functional-test-redis: cleanup-docker
	rm -f loggo-logs.pos
	docker-compose up -d redis
	timeout --preserve-status 10 ./build/loggo --transport="redis" --logs-path="pkg/tests/fixtures/pods" --position-file-path="loggo-logs.pos" --reader-max-chunk=2 && echo "ok" || echo "bad"
	./build/tests --transport="redis"

cleanup-docker:
	docker-compose stop
	docker-compose rm -f

build:
	mkdir -p build
	CGO_ENABLED=1 GOOS=linux go build -tags netgo --ldflags '-extldflags "-static"' -installsuffix cgo -o build/loggo cmd/loggo/main.go

build-test:
	go build -o build/tests cmd/tests/main.go

.PHONY: build test functional-test functional-test-redis lint cleanup-docker build-test
