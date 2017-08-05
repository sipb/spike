.PHONY: all clean

all: lookup.so lookup_processed.h demo.exe

demo.exe: demo/main/demo.go health/health.go maglev/maglev.go
	go build -o $@ github.com/sipb/spike/demo/main

l%okup.so l%okup.h: lookup/main/lookup.go health/health.go maglev/maglev.go
	go build -o lookup.so -buildmode=c-shared github.com/sipb/spike/lookup/main

lookup_processed.h: lookup.h
	gcc -E $< | grep -v '^#' >$@

clean:
	rm -f demo.exe lookup.so lookup.h
