.PHONY: test

test:
	go test -race -coverprofile=coverage.txt -covermode=atomic ./ ./internal/completion/ ./internal/option ./internal/help ./dag ./internal/sliceiterator ./text

race:
	go test -race ./dag -count=1

view: test
	go tool cover -html=coverage.txt -o coverage.html

# Assumes github.com/dgryski/semgrep-go is checked out in ../
rule-check:
	semgrep -f ../semgrep-go .
	for dir in ./ ./internal/completion ./internal/option ./internal/help ./dag ; do \
		echo $$dir ; \
		ruleguard -c=0 -rules ../semgrep-go/ruleguard.rules.go $$dir ; \
	done


lint:
	golangci-lint run --enable-all \
		-D funlen \
		-D dupl \
		-D lll \
		-D gocognit
