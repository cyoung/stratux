
ifeq "$(CIRCLECI)" "true"
	BUILDINFO=
else
	BUILDINFO=-ldflags "-X main.stratuxVersion=`git describe --tags --abbrev=0` -X main.stratuxBuild=`git log -n 1 --pretty=%H`"
$(if $(GOROOT),,$(error GOROOT is not set!))
endif

all:
	make xdump978
	make xdump1090
	make xlinux-mpu9150
	make xgen_gdl90

xdump978:
	cd dump978 && make lib
	sudo cp -f ./libdump978.so /usr/lib/libdump978.so

xdump1090:
	git submodule update --init
	cd dump1090 && make

xlinux-mpu9150:
	go get -d -v github.com/ccicchitelli/linux-mpu9150/mpu
	cd linux-mpu9150 && make -f Makefile-native-shared
	go build -o linux-mpu9150/mpu/mpu.a linux-mpu9150/mpu/mpu.go 

xgen_gdl90:
	go get -t -d -v ./main ./test ./godump978 ./mpu6050 ./uatparse
	go build $(BUILDINFO) -p 4 main/gen_gdl90.go main/traffic.go main/ry835ai.go main/network.go main/managementinterface.go main/sdr.go main/uibroadcast.go main/monotonic.go main/datalog.go main/equations.go main/ahrs.go

.PHONY: test
test:
	make -C test	

www:
	cd web && make

install:
	cp -f gen_gdl90 /usr/bin/gen_gdl90
	chmod 755 /usr/bin/gen_gdl90
	cp init.d-stratux /etc/init.d/stratux
	chmod 755 /etc/init.d/stratux
	ln -sf /etc/init.d/stratux /etc/rc2.d/S01stratux
	ln -sf /etc/init.d/stratux /etc/rc6.d/K01stratux
	make www
	cp -f dump1090/dump1090 /usr/bin/

clean:
	rm -f gen_gdl90 libdump978.so
	cd dump1090 && make clean
	cd dump978 && make clean
	rm -f linux-mpu9150/*.o linux-mpu9150/*.so
