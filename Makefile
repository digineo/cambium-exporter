DOCKER_TAG = digineode/cambium-exporter:latest

RELEASE ?= 0

.PHONY: dev
dev: config.toml
	go run main.go --config $<

.PHONY: release
release:
ifeq ($(RELEASE),0)
	goreleaser release --rm-dist --skip-publish --snapshot
	docker build --tag $(DOCKER_TAG) --pull .
else
	goreleaser release --rm-dist --skip-sign --skip-publish --auto-snapshot --release-footer debian/release-footer.md
	docker built --tag $(DOCKER_TAG) --pull .
	docker push $(DOCKER_TAG)
endif
