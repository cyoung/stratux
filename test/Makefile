$(if $(GOROOT),,$(error GOROOT is not set!))

SRCS = $(wildcard *.go)
DEST = $(patsubst %.go,%,$(SRCS))

all: $(DEST)

%: %.go
	go build $<
	
