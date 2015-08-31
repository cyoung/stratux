GOOS   ?= linux
GOARCH ?= arm
GOARM  ?= 7

all:
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) go get -t -d -v ./...
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) go build main/{gen_gdl90,traffic,ry835ai,network}.go

install:
	cp -f gen_gdl90 /usr/bin/gen_gdl90
	chmod 755 /usr/bin/gen_gdl90
clean:
	rm -f gen_gdl90
