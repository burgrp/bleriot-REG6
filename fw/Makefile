.PHONY: build flash gdb rtt install-pack disassembly new hub test

build:
	go run ./cmd/dev node build --name ssr

flash:
	go run ./cmd/dev node build --name ssr --flash --rtt

disassembly:
	go run ./cmd/dev node build --name ssr --disassembly

gdb rtt install-pack:
	go run ./cmd/dev node $@ --name ssr

new:
	go run ./cmd/dev node new

hub:
	go run ./cmd/dev hub --diagnostics rf

test:
	go test ./...