
fmt:
	go mod tidy
	gofumpt -w -l .
.PHONY: fmt

# bake can only execute unit tests
# integration tests that require a Docker environment cannot be run with bake
# and so we run them separately
test:
	rm -rf .coverage
	mkdir -p .coverage
	docker buildx bake go-test
	go test -run '_Docker$' -json -covermode=atomic -coverprofile=./.coverage/integration.gcov ./... | tparse -progress
.PHONY: test

lint:
	docker buildx bake linting
.PHONY: lint
