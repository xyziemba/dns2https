SRC = $(shell find . -type f -name "*.go" -print)

dnsproxy: $(SRC)
	go build

.PHONY: clean test

clean:
	rm dnsproxy

test-dns: dnsproxy
	./run_tests.sh