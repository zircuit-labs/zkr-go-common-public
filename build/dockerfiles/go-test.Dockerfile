FROM parent:latest AS worker

WORKDIR /src/
RUN mkdir /coverage
RUN mkdir /test-results

# Run the tests, excluding those with names ending in "_Docker" (e.g., "TestXXX_Docker")
# Convention will assume that all tests with "_Docker" in the name are integration tests which require
# a running Docker daemon and should not be run in this context.
RUN --mount=type=cache,target=/go-cache \
    --mount=type=cache,target=/gomod-cache \
    go test -skip '_Docker$' -race -json -shuffle=on -covermode=atomic -coverprofile=/coverage/unit.gcov ./... | tee /test-results/unit-.json | tparse -progress

RUN tparse -notests -format=markdown -file /test-results/unit.json | tee /test-results/unit.md

FROM scratch AS output
COPY --from=worker /coverage /
COPY --from=worker /test-results/*.md /
