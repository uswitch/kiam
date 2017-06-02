.PHONY: clean
	
bin/kiam:
	go build -o bin/kiam cmd/*.go

clean:
	rm -rf bin/