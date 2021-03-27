
ifeq "$(CIRCLECI)" "true"
	BUILDINFO=
	PLATFORMDEPENDENT=
else
	LFLAGS=-X main.stratuxVersion=`git describe --tags --abbrev=0` -X main.stratuxBuild=`git log -n 1 --pretty=%H`
	BUILDINFO=

ifeq "$(debug)" "true"
	LFLAGS+=-compressdwarf=false
	BUILDINFO+=-gcflags '-N -l'
endif

	BUILDINFO+=-ldflags "$(LFLAGS)"
	BUILDINFO_STATIC=-ldflags "-extldflags -static $(LFLAGS)"
$(if $(GOROOT),,$(error GOROOT is not set!))
	PLATFORMDEPENDENT=fancontrol
endif

ARCH := $(shell arch)
ifeq ($(ARCH),aarch64)
	OGN_RX_BINARY=ogn/ogn-rx-eu_aarch64
else
	OGN_RX_BINARY=ogn/ogn-rx-eu_arm
endif


all:
	make xdump978 xdump1090 gen_gdl90 $(PLATFORMDEPENDENT)

gen_gdl90: main/*.go common/*.go
	export CGO_CFLAGS_ALLOW="-L/root/stratux" && go build $(BUILDINFO) -o gen_gdl90 -p 4 ./main/

fancontrol: fancontrol_main/*.go common/*.go
	go build $(BUILDINFO) -o fancontrol -p 4 ./fancontrol_main/

xdump1090:
	git submodule update --init
	cd dump1090 && make BLADERF=no

xdump978:
	cd dump978 && make lib
	sudo cp -f ./libdump978.so /usr/lib/libdump978.so

.PHONY: test
test:
	make -C test

www:
	cd web && make

ogn/ddb.json:
	cd ogn && ./fetch_ddb.sh

install: ogn/ddb.json
	cp -f gen_gdl90 /usr/bin/gen_gdl90
	chmod 755 /usr/bin/gen_gdl90
	cp -f fancontrol /usr/bin/fancontrol
	chmod 755 /usr/bin/fancontrol
	-/usr/bin/fancontrol remove
	/usr/bin/fancontrol install
	cp image/10-stratux.rules /etc/udev/rules.d/10-stratux.rules
	cp image/99-uavionix.rules /etc/udev/rules.d/99-uavionix.rules
	rm -f /etc/init.d/stratux
	cp __lib__systemd__system__stratux.service /lib/systemd/system/stratux.service
	cp __root__stratux-pre-start.sh /root/stratux-pre-start.sh
	chmod 644 /lib/systemd/system/stratux.service
	chmod 744 /root/stratux-pre-start.sh
	ln -fs /lib/systemd/system/stratux.service /etc/systemd/system/multi-user.target.wants/stratux.service
	make www
	cp -f libdump978.so /usr/lib/libdump978.so
	cp -f dump1090/dump1090 /usr/bin/
	cp -f image/hostapd_manager.sh /usr/sbin/
	cp -f image/stratux-wifi.sh /usr/sbin/
	cp -f image/hostapd.conf.template /etc/hostapd/
	cp -f image/interfaces.template /etc/network/
	cp -f image/wpa_supplicant.conf.template /etc/wpa_supplicant/
	cp -f $(OGN_RX_BINARY) /usr/bin/ogn-rx-eu
	cp -f ogn/ddb.json /etc/

clean:
	rm -f gen_gdl90 libdump978.so fancontrol ahrs_approx
	cd dump1090 && make clean
	cd dump978 && make clean
