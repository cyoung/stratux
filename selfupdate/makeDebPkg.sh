#!/bin/bash
#set -x # echo on

#apt-get install -y dh-make

stratuxVersion=`git describe --tags --abbrev=0 | sed -e "s/^v//"`
stratuxBuild=`git log -n 1 --pretty=%H`

echo "Packaging ${stratuxVersion} (${stratuxBuild})."

cd ..
make
rm -rf work
mkdir -p work
mkdir -p work/DEBIAN
read -d '' control <<- EOF
	Package: stratux 
	Version: ${stratuxVersion} 
	Architecture: all 
	Maintainer: Chris Young<cyoung@slack> 
	Installed-Size: 2 
	Depends: 
	Section: extras 
	Priority: optional 
	Homepage: your homepage 
	Description: describe 
EOF
echo "${control}" > work/DEBIAN/control

read -d '' preinst <<- EOF
#disable serial console
sed -i /boot/cmdline.txt -e "s/console=ttyAMA0,[0-9]\+ //"

# Remove old Wi-Fi watcher script.
rm -f /usr/sbin/wifi_watch.sh
sed -i "/\bwifi_watch\b/d" /etc/rc.local

#remove old fan control utility
rm -f /usr/bin/fancontrol.py

# Startup script.
RASPBIAN_VERSION=`cat /etc/debian_version`
if test "$RASPBIAN_VERSION" = "8.0" ; then
	# Install the systemd startup scripts in any case, even if they won't be used. If this is being run, then the old init.d script
	#  is still intact and we just leave it. If running Wheezy, then remove the old init.d script.
	rm -f /etc/init.d/stratux
	rm -f /etc/rc2.d/S01stratux
	rm -f /etc/rc6.d/K01stratux
fi

EOF
echo "${preinst}" > work/DEBIAN/preinst
chmod 775 work/DEBIAN/preinst

read -d '' postinst <<- EOF
/usr/bin/fancontrol remove
/usr/bin/fancontrol install
EOF
echo "${postinst}" > work/DEBIAN/postinst
chmod 775 work/DEBIAN/postinst

mkdir -p work/boot
mkdir -p work/etc/hostapd
mkdir -p work/etc/modprobe.d
mkdir -p work/etc/systemd/system/multi-user.target.wants
mkdir -p work/etc/udev/rules.d
mkdir -p work/lib/systemd/system
mkdir -p work/usr/bin
mkdir -p work/usr/lib
mkdir -p work/usr/sbin
mkdir -p work/root

cp gen_gdl90 work/usr/bin/
cp libdump978.so work/usr/lib/
cp linux-mpu9150/libimu.so work/usr/lib/
cp -f dump1090/dump1090 work/usr/bin/

cp __lib__systemd__system__stratux.service work/lib/systemd/system/stratux.service
chmod 644 work/lib/systemd/system/stratux.service
cp __root__stratux-pre-start.sh work/root/stratux-pre-start.sh
chmod 744 work/root/stratux-pre-start.sh
ln -fs /lib/systemd/system/stratux.service work/etc/systemd/system/multi-user.target.wants/stratux.service

#wifi config
	cp image/hostapd.conf work/etc/hostapd/
	cp image/hostapd-edimax.conf work/etc/hostapd/

#WiFi Hostapd ver test and hostapd.conf builder script
	cp image/stratux-wifi.sh work/usr/sbin/

#WiFi Config Manager
	cp image/hostapd_manager.sh work/usr/sbin/

#SDR Serial Script
	cp image/sdr-tool.sh work/usr/sbin/

#boot config
	cp image/config.txt work/boot/

#modprobe.d blacklist
	cp -f image/rtl-sdr-blacklist.conf work/etc/modprobe.d/

#udev config
	cp -f image/10-stratux.rules work/etc/udev/rules.d
	cp -f image/99-uavionix.rules work/etc/udev/rules.d

#go setup
	cp -f image/bashrc.txt work/root/.bashrc
	cp -f image/stxAliases.txt work/root/.stxAliases

# /etc/modules
	cp -f image/modules.txt work/etc/modules

#motd
	cp -f image/motd work/etc/motd

#fan control utility
	cp -f fancontrol work/usr/bin/
	chmod 755 work/usr/bin/fancontrol

# Web files install.
  mkdir -p work/var/www
  mkdir -p work/var/www/css
  cp web/css/*.css work/var/www/css
  mkdir -p work/var/www/js
  cp web/js/main.js work/var/www/js
  cp web/js/addtohomescreen.min.js work/var/www/js
  cp web/js/j3di-all.min.js work/var/www/js
  mkdir -p work/var/www/img
  cp web/img/logo*.png work/var/www/img
  cp web/img/screen*.png work/var/www/img
  cp web/img/world.png work/var/www/img
  mkdir -p work/var/www/maui
  mkdir -p work/var/www/maui/js
  cp web/maui/js/angular-ui-router.min.js work/var/www/maui/js
  cp web/maui/js/mobile-angular-ui.min.js work/var/www/maui/js
  cp web/maui/js/angular.min.js work/var/www/maui/js
  cp web/maui/js/mobile-angular-ui.gestures.min.js work/var/www/maui/js
  cp web/maui/js/mobile-angular-ui.core.min.js work/var/www/maui/js
  mkdir -p work/var/www/maui/css
  cp web/maui/css/mobile-angular-ui-hover.min.css work/var/www/maui/css
  cp web/maui/css/mobile-angular-ui-desktop.min.css work/var/www/maui/css
  cp web/maui/css/mobile-angular-ui-base.min.css work/var/www/maui/css
  mkdir -p work/var/www/maui/fonts
  cp web/maui/fonts/fontawesome-webfont.woff work/var/www/maui/fonts
  mkdir -p work/var/www/plates
  cp web/plates/*.html work/var/www/plates
  mkdir -p work/var/www/plates/js
  cp web/plates/js/*.js work/var/www/plates/js
  cp web/index.html work/var/www
  cp web/stratux.appcache work/var/www
  # Mark the manifest with the git hash.
  echo "# build time: " ${buildtime} >>work/var/www/stratux.appcache
  echo "# Stratux build: " ${stratuxBuild} >>work/var/www/stratux.appcache

# Create Debian package
dpkg -b work
mv work.deb stratux-${stratuxVersion}.deb

rm -rf work
