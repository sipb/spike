.PHONY: all clean

LIBFILES = common/backend.go common/tuple.go health/health.go maglev/maglev.go tracking/tracking.go

all: demo.exe lookup.so lookup_processed.h

demo.exe: demo/main/demo.go $(LIBFILES)
	go build -o $@ github.com/sipb/spike/demo/main

l%okup.so l%okup.h: lookup/main/lookup.go $(LIBFILES)
	go build -o lookup.so -buildmode=c-shared github.com/sipb/spike/lookup/main

lookup_processed.h: lookup.h
	gcc -E $< | grep -v '^#' >$@

clean:
	rm -f demo.exe lookup.so lookup.h
