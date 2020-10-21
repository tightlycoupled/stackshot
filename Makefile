.PHONY: test
test:
	go test ./...

.PHONY: test-coverage
test-coverage:
	go test -coverprofile cp.out ./...
	go tool cover -html=cp.out

.PHONY: build
build:
	go build ./cmds/...

build-distribs:
	GOOS=darwin go build -o stackshot-darwin-${VERSION} ./cmds/...
	GOOS=linux go build -o stackshot-linux-${VERSION} ./cmds/...

.PHONY: clean
clean:
	-rm cp.out
	-rm stackshot

