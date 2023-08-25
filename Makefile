.PHONY: clean

all:
	go get
	go build cmd/godl.go cmd/progress.go

test: all
	@set -e; \
	STATUS=0; \
	go test || STATUS=$$?; \
	go test ./cmd || STATUS=$$?; \
	exit $$STATUS; \

clean:
	rm godl
