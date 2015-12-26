ifeq "$(CIRCLECI)" "true"
	#
else
$(if $(GOROOT),,$(error GOROOT is not set!))
endif

SRCS = $(wildcard *.go)
DEST = $(patsubst %.go,%,$(SRCS))

all: $(DEST)

%: %.go
	go build $<
	
