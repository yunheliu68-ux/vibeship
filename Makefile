.PHONY: build run collect test clean

build:
	go build -o vibeship .

run: build
	./vibeship

collect: build
	./vibeship collect

test:
	go test ./...

clean:
	rm -f vibeship
