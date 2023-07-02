export STRATUX_HOME := /opt/stratux/
ifeq "$(CIRCLECI)" "true"
	BUILDINFO=
	PLATFORMDEPENDENT=
else
	LFLAGS=-X main.stratuxVersion=`git describe --tags --abbrev=0` -X main.stratuxBuild=`git log -n 1 --pretty=%H`  
	BUILDINFO=-ldflags "$(LFLAGS)"
	BUILDINFO_STATIC=-ldflags "-extldflags -static $(LFLAGS)"
	PLATFORMDEPENDENT=fancontrol
endif

ifeq ($(debug),true)
	BUILDINFO := -gcflags '-N -l' $(BUILDINFO)
endif

ARCH=$(shell arch)
ifeq ($(ARCH),aarch64)
	OGN_RX_BINARY=ogn/ogn-rx-eu_aarch64
else ifeq ($(ARCH),x86_64)
	OGN_RX_BINARY=ogn/ogn-rx-eu_x86
else
	OGN_RX_BINARY=ogn/ogn-rx-eu_arm
endif



all: libdump978.so xdump1090 xrtlais gen_gdl90 $(PLATFORMDEPENDENT)

gen_gdl90: main/*.go common/*.go libdump978.so
	LIBRARY_PATH=$(CURDIR) CGO_CFLAGS_ALLOW="-L$(CURDIR)" go build $(BUILDINFO) -o gen_gdl90 -p 4 ./main/

fancontrol: fancontrol_main/*.go common/*.go
	go build $(BUILDINFO) -o fancontrol -p 4 ./fancontrol_main/

xdump1090:
	git submodule update --init
	cd dump1090 && make BLADERF=no

libdump978.so: dump978/*.c dump978/*.h
	cd dump978 && make lib

xrtlais:
	git submodule update --init
	cd rtl-ais && sed -i 's/^LDFLAGS+=-lpthread.*/LDFLAGS+=-lpthread -lm -lrtlsdr -L \/usr\/lib\//' Makefile && make


.PHONY: test
test:
	make -C test

www:
	make -C web

ogn/ddb.json:
	cd ogn && ./fetch_ddb.sh

optinstall: www ogn/ddb.json
	mkdir -p $(STRATUX_HOME)/bin
	mkdir -p $(STRATUX_HOME)/www
	mkdir -p $(STRATUX_HOME)/ogn
	mkdir -p $(STRATUX_HOME)/GxAirCom
	mkdir -p $(STRATUX_HOME)/cfg
	mkdir -p $(STRATUX_HOME)/lib
	mkdir -p $(STRATUX_HOME)/mapdata
	chmod a+rwx $(STRATUX_HOME)/mapdata # so users can upload their stuff as user pi

	# binaries
	cp -f gen_gdl90 $(STRATUX_HOME)/bin/
	cp -f fancontrol $(STRATUX_HOME)/bin/
	cp -f dump1090/dump1090 $(STRATUX_HOME)/bin
	cp -f rtl-ais/rtl_ais $(STRATUX_HOME)/bin
	cp -f $(OGN_RX_BINARY) $(STRATUX_HOME)/bin/ogn-rx-eu
	chmod +x $(STRATUX_HOME)/bin/*

	# Libs
	cp -f libdump978.so $(STRATUX_HOME)/lib/

	# map data
	cp -ru mapdata/* $(STRATUX_HOME)/mapdata/

	# OGN stuff
	cp -f ogn/ddb.json ogn/esp32-ogn-tracker-bin-*.zip ogn/install-ogntracker-firmware-pi.sh ogn/fetch_ddb.sh $(STRATUX_HOME)/ogn

	# GxAirCom stuff
	for artifact in "firmware_psRam.bin" "spiffs.bin" "partitions.bin" "version.txt" "README.md" "bootloader_dio_40m.bin" "boot_app0.bin" ; do \
		curl -L https://github.com/rvt/GxAirCom/releases/latest/download/$$artifact --output $(STRATUX_HOME)/GxAirCom/$$artifact ; \
	done
	cp -f GxAirCom/esptool.py GxAirCom/install-GxAirCom-Stratux-firmware.sh $(STRATUX_HOME)/GxAirCom

	# Scripts
	cp __opt__stratux__bin__stratux-pre-start.sh $(STRATUX_HOME)/bin/stratux-pre-start.sh
	chmod 744 $(STRATUX_HOME)/bin/stratux-pre-start.sh
	cp -f image/stratux-wifi.sh $(STRATUX_HOME)/bin/
	cp -f image/sdr-tool.sh $(STRATUX_HOME)/bin/
	chmod 755 $(STRATUX_HOME)/bin/*

	# Config templates
	cp -f image/stratux-dnsmasq.conf.template $(STRATUX_HOME)/cfg/
	cp -f image/interfaces.template $(STRATUX_HOME)/cfg/
	cp -f image/wpa_supplicant.conf.template $(STRATUX_HOME)/cfg/
	cp -f image/wpa_supplicant_ap.conf.template $(STRATUX_HOME)/cfg/


install: optinstall
	-$(STRATUX_HOME)/bin/fancontrol remove
	$(STRATUX_HOME)/bin/fancontrol install

	# System configuration
	cp image/10-stratux.rules /etc/udev/rules.d/10-stratux.rules
	cp image/99-uavionix.rules /etc/udev/rules.d/99-uavionix.rules
	cp __lib__systemd__system__stratux.service /lib/systemd/system/stratux.service
	chmod 644 /lib/systemd/system/stratux.service
	ln -fs /lib/systemd/system/stratux.service /etc/systemd/system/multi-user.target.wants/stratux.service


clean:
	rm -f gen_gdl90 libdump978.so fancontrol ahrs_approx
	cd dump1090 && make clean
	cd dump978 && make clean
	cd rtl-ais && make clean
