all: client server

client:
	go build -o client ./cmd/client/go

server:
	go build -o server ./cmd/server

clean:
	rm client server

.PHONY: all client server clean
