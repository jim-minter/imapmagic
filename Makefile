imapmagic: $(shell find cmd pkg)
	go build ./cmd/imapmagic

clean:
	rm -f imapmagic

.PHONY: clean
