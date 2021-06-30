.PHONY: dev
dev: config.toml
	go run main.go --config $<

.PHONY: release
release:
	goreleaser release --rm-dist --skip-sign --skip-publish --auto-snapshot
