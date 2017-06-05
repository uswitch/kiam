.PHONY: clean
	
bin/kiam: $(shell find . -name '*.go')
	go build -o bin/kiam cmd/*.go

clean:
	rm -rf bin/