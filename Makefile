.PHONY: toc test deps

toc:
	docker run --rm -it -v ${PWD}:/usr/src jorgeandrada/doctoc --github

test:
	go vet . ./services/...
	docker-compose -p zenly -f ./docker-compose.yml up -d
	go test -v -cpu 1,4 -run=. -bench=xxx . ./services/...

deps:
	dep ensure -v
	dep prune -v
	go generate .
	go tool fix -force context -r context . || true
