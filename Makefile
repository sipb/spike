.PHONY: all clean test

LIBFILES := $(shell find common health maglev tracking -name '*.go')

all: bin/demo lookup.so lookup_processed.h

test:
	go test github.com/sipb/spike/maglev

bin/demo: $(shell find demo -name '*.go') $(LIBFILES)
	go build -o $@ github.com/sipb/spike/demo/main

l%okup.so l%okup.h: $(shell find lookup -name '*.go') $(LIBFILES)
	go build -o lookup.so -buildmode=c-shared github.com/sipb/spike/lookup/main

lookup_processed.h: lookup.h
	gcc -E $< | grep -v '^#' >$@

clean:
	rm -f bin/demo lookup.so lookup.h
