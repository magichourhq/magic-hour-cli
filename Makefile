.PHONY: completions snapshot

completions:
	mkdir -p completions
	go run ./cmd/mh completion bash > completions/mh.bash
	go run ./cmd/mh completion zsh > completions/mh.zsh
	go run ./cmd/mh completion fish > completions/mh.fish
	go run ./cmd/mh completion powershell > completions/mh.ps1

snapshot:
	go run github.com/goreleaser/goreleaser/v2@v2.18.2 release --snapshot --clean
