SRC = $(shell find . -type f -name "*.go" -print)

dnsproxy: $(SRC)
	go build

.PHONY: clean test

clean:
	rm dnsproxy

test-dns: dnsproxy
	./dnsproxy -selftest
	./run_tests.sh