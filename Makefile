DOCKER_TAG = ghcr.io/digineo/cambium-exporter:latest

RELEASE ?= 0

.PHONY: dev
dev: config.toml
	go run main.go --config $< --verbose

.PHONY: release
release:
ifeq ($(RELEASE),0)
	goreleaser release --clean --skip=publish --snapshot
	docker build --tag $(DOCKER_TAG) --pull .
else
	goreleaser release --clean --skip=sign --release-footer debian/release-footer.md
	docker build --tag $(DOCKER_TAG) --pull .
	docker push $(DOCKER_TAG)
endif
