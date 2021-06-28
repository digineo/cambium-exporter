.PHONY: run
run: config.toml
	go run main.go --config $<
