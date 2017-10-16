.PHONY: toc test deps

toc:
	docker run --rm -it -v ${PWD}:/usr/src jorgeandrada/doctoc --github

test:
	staticcheck . ./services
	go vet . ./services
	go test -v -cpu 1,4 -run=. -bench=xxx . ./services

deps:
	dep ensure -v
	go generate .
	go tool fix -force context -r context . || true
