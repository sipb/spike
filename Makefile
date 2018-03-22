.PHONY: all clean test lookup_dry

LIBFILES := $(shell find common config health maglev tracking lookup -name '*.go')

all: bin/demo lookup.so lookup_processed.h

test:
	go test github.com/sipb/spike/maglev

bin/demo: $(shell find demo -name '*.go') $(LIBFILES)
	go build -o $@ github.com/sipb/spike/demo/main

lookup_dry:  $(shell find lookup -name '*.go') $(LIBFILES)
	go build -o /dev/null github.com/sipb/spike/lookup/main

l%okup.so l%okup.h: $(shell find lookup -name '*.go') $(LIBFILES) lookup_dry
	go build -o lookup.so -buildmode=c-shared github.com/sipb/spike/lookup/main

lookup_processed.h: lookup.h
	gcc -E $< | grep -v '^#' >$@

clean:
	rm -f bin/demo lookup.so lookup.h lookup_processed.h
