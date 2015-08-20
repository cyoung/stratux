GOOS   ?= linux
GOARCH ?= arm
GOARM  ?= 7

all:
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) go build gen_gdl90.go traffic.go ry835ai.go network.go
clean:
	rm -f gen_gdl90
