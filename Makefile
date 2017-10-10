.PHONY: clean
	
bin/agent: $(shell find . -name '*.go')
	go build -o bin/agent cmd/agent/*.go

clean:
	rm -rf bin/