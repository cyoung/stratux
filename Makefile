
ifeq "$(CIRCLECI)" "true"
	BUILDINFO=
	PLATFORMDEPENDENT=
else
	LDFLAGS_VERSION=-X main.stratuxVersion=`git describe --tags --abbrev=0` -X main.stratuxBuild=`git log -n 1 --pretty=%H`
	BUILDINFO=-ldflags "$(LDFLAGS_VERSION)"
	BUILDINFO_STATIC=-ldflags "-extldflags -static $(LDFLAGS_VERSION)"
$(if $(GOROOT),,$(error GOROOT is not set!))
	PLATFORMDEPENDENT=fancontrol
endif

all:
	make xdump978 xdump1090 xgen_gdl90 $(PLATFORMDEPENDENT)

xgen_gdl90:
	go get -t -d -v ./main ./godump978 ./uatparse ./sensors
	export CGO_CFLAGS_ALLOW="-L/root/stratux" && go build $(BUILDINFO) -p 4 main/gen_gdl90.go main/traffic.go main/gps.go main/network.go main/managementinterface.go main/sdr.go main/ping.go main/uibroadcast.go main/monotonic.go main/datalog.go main/equations.go main/sensors.go main/cputemp.go main/lowpower_uat.go main/flarm.go main/flarm-nmea.go main/networksettings.go

fancontrol:
	go get -t -d -v ./main
	go build $(BUILDINFO) -p 4 main/fancontrol.go main/equations.go main/cputemp.go

xdump1090:
	git submodule update --init
	cd dump1090 && make

xdump978:
	cd dump978 && make lib
	sudo cp -f ./libdump978.so /usr/lib/libdump978.so

.PHONY: test
test:
	make -C test

www:
	cd web && make

install:
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
	mkdir -p  /var/lib/stratux/
	cp -f ogn/rtlsdr-ogn/stratux.conf.template /etc/stratux-ogn.conf.template
	rm -f /var/run/ogn-rf.fifo
	mkfifo /var/run/ogn-rf.fifo
	cp -f ogn/rtlsdr-ogn/ogn-rf /usr/bin/
	chmod a+s /usr/bin/ogn-rf
	cp -f ogn/rtlsdr-ogn/ogn-decode /usr/bin/
	cp -f ogn/ddb.json /etc/

clean:
	rm -f gen_gdl90 libdump978.so fancontrol ahrs_approx
	cd dump1090 && make clean
	cd dump978 && make clean
