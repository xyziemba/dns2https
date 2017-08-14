SRC = $(shell find . -type f -name "*.go" -print)

dns2https: $(SRC)
	go build

.PHONY: clean test

clean:
	rm dns2https

test-dns: dns2https
	./dns2https -selftest
	./run_tests.sh