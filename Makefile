ifeq "$(CIRCLECI)" "true"
	BUILDINFO=""
else
	BUILDINFO=" -X main.stratuxVersion=`git describe --tags --abbrev=0` -X main.stratuxBuild=`git log -n 1 --pretty=%H`"
endif

all:
	cd dump978 && make lib
	sudo cp -f ./libdump978.so /usr/lib/libdump978.so
	go get -t -d -v ./...
	go build -ldflags $(BUILDINFO) main/gen_gdl90.go main/traffic.go main/ry835ai.go main/network.go main/managementinterface.go main/sdr.go main/uibroadcast.go

test:
	sh -c true

www:
	mkdir -p /var/www
	mkdir -p /var/www/css
	cp web/css/*.css /var/www/css
	mkdir -p /var/www/js
	cp web/js/main.js /var/www/js
	cp web/js/addtohomescreen.min.js /var/www/js
	cp web/js/j3di-all.min.js /var/www/js
	mkdir -p /var/www/img
	cp web/img/logo*.png /var/www/img
	cp web/img/screen*.png /var/www/img
	cp web/img/world.png /var/www/img
	mkdir -p /var/www/maui
	mkdir -p /var/www/maui/js
	cp web/maui/js/angular-ui-router.min.js /var/www/maui/js
	cp web/maui/js/mobile-angular-ui.min.js /var/www/maui/js
	cp web/maui/js/angular.min.js /var/www/maui/js
	cp web/maui/js/mobile-angular-ui.gestures.min.js /var/www/maui/js
	cp web/maui/js/mobile-angular-ui.core.min.js /var/www/maui/js
	mkdir -p /var/www/maui/css
	cp web/maui/css/mobile-angular-ui-hover.min.css /var/www/maui/css
	cp web/maui/css/mobile-angular-ui-desktop.min.css /var/www/maui/css
	cp web/maui/css/mobile-angular-ui-base.min.css /var/www/maui/css
	mkdir -p /var/www/maui/fonts
	cp web/maui/fonts/fontawesome-webfont.woff /var/www/maui/fonts
	mkdir -p /var/www/plates
	cp web/plates/*.html /var/www/plates
	mkdir -p /var/www/plates/js
	cp web/plates/js/*.js /var/www/plates/js
	cp web/index.html /var/www

install:
	cp -f gen_gdl90 /usr/bin/gen_gdl90
	chmod 755 /usr/bin/gen_gdl90

clean:
	rm -f gen_gdl90 libdump978.so
