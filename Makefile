GOOS   ?= linux
GOARCH ?= arm
GOARM  ?= 7

all:
#	GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) go get -t -d -v ./...
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) go build -ldflags " -X main.stratuxVersion=`git describe --abbrev=0 --tags` -X main.stratuxBuild=`git log -n 1 --pretty=%H`" main/gen_gdl90.go main/traffic.go main/ry835ai.go main/network.go main/managementinterface.go main/sdr.go

test:
	sh -c true

install:
	cp -f gen_gdl90 /usr/bin/gen_gdl90
	chmod 755 /usr/bin/gen_gdl90
clean:
	rm -f gen_gdl90
