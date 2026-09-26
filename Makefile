.PHONY: build generate check completions snapshot

build:
	CGO_ENABLED=0 go build -trimpath -o mh ./cmd/mh

generate:
	go generate ./...

check:
	go vet ./...
	go build ./...

completions:
	mkdir -p completions
	go run ./cmd/mh completion bash > completions/mh.bash
	go run ./cmd/mh completion zsh > completions/mh.zsh
	go run ./cmd/mh completion fish > completions/mh.fish
	go run ./cmd/mh completion powershell > completions/mh.ps1

snapshot:
	goreleaser release --snapshot --clean
