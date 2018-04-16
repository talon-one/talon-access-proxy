build:
	go generate ./...
	go build ./cmd/talon-access-proxy

test:
	go generate ./...
	go test -v -count=1 ./...