ifeq ($(statuxBuild),)
stratuxBuild=`git log -n 1 --pretty=%H`
buildtime=`date +%s`
endif

all:
	mkdir -p /var/www
	mkdir -p /var/www/css
	cp css/*.css /var/www/css
	mkdir -p /var/www/js
	cp js/main.js /var/www/js
	cp js/addtohomescreen.min.js /var/www/js
	cp js/j3di-all.min.js /var/www/js
	mkdir -p /var/www/img
	cp img/logo*.png /var/www/img
	cp img/screen*.png /var/www/img
	cp img/world.png /var/www/img
	mkdir -p /var/www/maui
	mkdir -p /var/www/maui/js
	cp maui/js/angular-ui-router.min.js /var/www/maui/js
	cp maui/js/mobile-angular-ui.min.js /var/www/maui/js
	cp maui/js/angular.min.js /var/www/maui/js
	cp maui/js/mobile-angular-ui.gestures.min.js /var/www/maui/js
	cp maui/js/mobile-angular-ui.core.min.js /var/www/maui/js
	mkdir -p /var/www/maui/css
	cp maui/css/mobile-angular-ui-hover.min.css /var/www/maui/css
	cp maui/css/mobile-angular-ui-desktop.min.css /var/www/maui/css
	cp maui/css/mobile-angular-ui-base.min.css /var/www/maui/css
	mkdir -p /var/www/maui/fonts
	cp maui/fonts/fontawesome-webfont.woff /var/www/maui/fonts
	mkdir -p /var/www/plates
	cp plates/*.html /var/www/plates
	mkdir -p /var/www/plates/js
	cp plates/js/*.js /var/www/plates/js
	cp index.html /var/www
	cp stratux.appcache /var/www
	# Mark the manifest with the git hash.
	echo "# build time: " ${buildtime} >>/var/www/stratux.appcache
	echo "# Stratux build: " ${stratuxBuild} >>/var/www/stratux.appcache