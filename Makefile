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

.PHONY: clean
clean:
	-rm cp.out
	-rm stackshot

