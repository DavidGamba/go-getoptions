.PHONY: test

test:
	go test -coverprofile=coverage.txt -covermode=atomic ./ ./completion/ ./option ./help ./dag ./sliceiterator ./text
	cd go-getoptions && go test -coverprofile=coverage.txt -covermode=atomic ./

race:
	go test -race ./dag -count=1

view: test
	go tool cover -html=coverage.txt -o coverage.html

# Assumes github.com/dgryski/semgrep-go is checked out in ../
rule-check:
	semgrep -f ../semgrep-go .
	for dir in ./ ./completion ./option ./help ./dag ; do \
		echo $$dir ; \
		ruleguard -c=0 -rules ../semgrep-go/ruleguard.rules.go $$dir ; \
	done


lint:
	golangci-lint run --enable-all \
		-D funlen \
		-D dupl \
		-D lll \
		-D gocognit
