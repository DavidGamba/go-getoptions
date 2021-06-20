.PHONY: test

test:
	go test -coverprofile=coverage.txt -covermode=atomic ./

view: test
	go tool cover -html=coverage.txt -o coverage.html

