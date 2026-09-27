.PHONY: completions snapshot spec

spec:
	@raw=$$(mktemp); pretty=$$(mktemp api/openapi.json.XXXXXX); \
	trap 'rm -f "$$raw" "$$pretty"' 0; \
	curl -fsSL --retry 3 https://magichour.ai/openapi.json -o "$$raw" && \
	jq -e '.openapi and (.paths | type == "object")' "$$raw" >/dev/null && \
	jq . "$$raw" > "$$pretty" && mv "$$pretty" api/openapi.json

completions:
	mkdir -p completions
	go run ./cmd/mh completion bash > completions/mh.bash
	go run ./cmd/mh completion zsh > completions/mh.zsh
	go run ./cmd/mh completion fish > completions/mh.fish
	go run ./cmd/mh completion powershell > completions/mh.ps1

snapshot:
	go run github.com/goreleaser/goreleaser/v2@v2.18.2 release --snapshot --clean
