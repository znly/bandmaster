.PHONY: toc test bench test-bench

toc:
	docker run --rm -it -v ${PWD}:/usr/src jorgeandrada/doctoc --github

build:
	go build -v $(shell go list ./...)

test:
	staticcheck . ./services
	go vet . ./services
	go test -v -race -cpu 1,2,4,8 -cover -run=. -bench=. . ./services

bench:
	go test -v -cpu 1,2,4,8,24 -cover -run=xxx -bench=. . ./services

test-bench:
	go test -v -cpu 1,2,4,8,24 -cover -run=. -bench=. . ./services
