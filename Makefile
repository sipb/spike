.PHONY: all clean

all: libspike.so libspike.h

%.so %.h: main.go health/health.go maglev/maglev.go
	go build -o $*.so -buildmode=c-shared

clean:
	rm -f libspike.so libspike.h
