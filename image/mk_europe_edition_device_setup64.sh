#!/bin/bash

# DO NOT CALL ME DIRECTLY!
# This script is called by mk_europe_edition.sh via qemu
set -ex

cd /root/stratux

# Make sure that the upgrade doesn't restart services in the chroot..
mkdir /root/fake
ln -s /bin/true /root/fake/initctl
ln -s /bin/true /root/fake/invoke-rc.d
ln -s /bin/true /root/fake/restart
ln -s /bin/true /root/fake/start
ln -s /bin/true /root/fake/stop
ln -s /bin/true /root/fake/start-stop-daemon
ln -s /bin/true /root/fake/service
ln -s /bin/true /root/fake/deb-systemd-helper

# Fake a proc FS for raspberrypi-sys-mods_20170519_armhf... Extend me as needed
mkdir -p /proc/sys/vm/

apt update
#PATH=/root/fake:$PATH apt dist-upgrade --yes
apt clean

PATH=/root/fake:$PATH apt install --yes libjpeg62-turbo-dev libconfig9 rpi-update hostapd isc-dhcp-server tcpdump git cmake \
    libusb-1.0-0-dev build-essential build-essential autoconf libtool i2c-tools screen libfftw3-dev libncurses-dev python-serial
apt clean
#echo y | rpi-update

systemctl enable isc-dhcp-server
systemctl enable ssh
systemctl disable dhcpcd
systemctl disable hciuart
systemctl disable hostapd

sed -i 's/INTERFACESv4=""/INTERFACESv4="wlan0"/g' /etc/default/isc-dhcp-server

rm -r /proc/*
rm -r /root/fake


# Install golang
cd /root
wget https://golang.org/dl/go1.16.1.linux-arm64.tar.gz
tar xzf go1.16.1.linux-arm64.tar.gz
rm go1.16.1.linux-arm64.tar.gz


# Prepare wiringpi for fancontrol and some more tools. Need latest version for pi4 support
cd /root && git clone https://github.com/WiringPi/WiringPi.git && cd WiringPi/wiringPi && make -j8 && make install
cd /root && rm -r WiringPi
#wget https://project-downloads.drogon.net/wiringpi-latest.deb
#dpkg -i wiringpi-latest.deb
#rm wiringpi-latest.deb


cd /root/stratux
cp image/bashrc.txt /root/.bashrc
source /root/.bashrc

# Prepare librtlsdr. The one shipping with buster uses usb_zerocopy, which is extremely slow on newer kernels, so
# we manually compile the osmocom version that disables zerocopy by default..
cd /root/
rm -rf rtl-sdr
git clone https://github.com/osmocom/rtl-sdr.git
cd rtl-sdr
git checkout 0847e93e0869feab50fd27c7afeb85d78ca04631 # Nov. 20, 2020
mkdir build && cd build
cmake .. -DENABLE_ZEROCOPY=0
make -j8
make install
cd /root/
rm -r rtl-sdr

ldconfig

# Debian seems to ship with an invalid pkgconfig for librtlsdr.. fix it:
#sed -i -e 's/prefix=/prefix=\/usr/g' /usr/lib/arm-linux-gnueabihf/pkgconfig/librtlsdr.pc
#sed -i -e 's/libdir=/libdir=${prefix}\/lib\/arm-linux-gnueabihf/g' /usr/lib/arm-linux-gnueabihf/pkgconfig/librtlsdr.pc


# Compile stratux
cd /root/stratux

make clean
make -j8
make install


##### Some device setup - copy files from image directory ####
cd /root/stratux/image
#motd
cp -f motd /etc/motd

#dhcpd config
cp -f dhcpd.conf /etc/dhcp/dhcpd.conf
cp -f dhcpd.conf.template /etc/dhcp/dhcpd.conf.template

#hostapd config
cp -f hostapd.conf /etc/hostapd/hostapd.conf
cp -f hostapd.conf.template /etc/hostapd/hostapd.conf.template

#WPA supplicant config for wifi direct
cp -f wpa_supplicant.conf.template /etc/wpa_supplicant/wpa_supplicant.conf.template

#hostapd manager script
cp -f hostapd_manager.sh /usr/sbin/hostapd_manager.sh
chmod 755 /usr/sbin/hostapd_manager.sh

#remove hostapd startup scripts
rm -f /etc/rc*.d/*hostapd /etc/network/if-pre-up.d/hostapd /etc/network/if-post-down.d/hostapd /etc/init.d/hostapd /etc/default/hostapd
#interface config
cp -f interfaces /etc/network/interfaces
cp -f interfaces.template /etc/network/interfaces.template

#custom hostapd start script
cp stratux-wifi.sh /usr/sbin/
chmod 755 /usr/sbin/stratux-wifi.sh

#SDR Serial Script
cp -f sdr-tool.sh /usr/sbin/sdr-tool.sh
chmod 755 /usr/sbin/sdr-tool.sh

#ping udev
cp -f 99-uavionix.rules /etc/udev/rules.d

#logrotate conf
cp -f logrotate.conf /etc/logrotate.conf

#fan/temp control script
#remove old script
rm -rf /usr/bin/fancontrol.py /usr/bin/fancontrol
#install new program
cp ../fancontrol /usr/bin
chmod 755 /usr/bin/fancontrol
/usr/bin/fancontrol remove
/usr/bin/fancontrol install

#isc-dhcp-server config
cp -f isc-dhcp-server /etc/default/isc-dhcp-server

#sshd config
# Do not copy for now. It contains many deprecated options and isn't needed.
cp -f sshd_config /etc/ssh/sshd_config

#udev config
cp -f 10-stratux.rules /etc/udev/rules.d

#stratux files
cp -f ../libdump978.so /usr/lib/libdump978.so

#debug aliases
cp -f stxAliases.txt /root/.stxAliases

#rtl-sdr setup
cp -f rtl-sdr-blacklist.conf /etc/modprobe.d/

#system tweaks
cp -f modules.txt /etc/modules

#boot settings
cp -f config.txt /boot/
echo -e "\narm_64bit=1" >> /boot/config.txt

#startup scripts
cp -f ../__lib__systemd__system__stratux.service /lib/systemd/system/stratux.service
cp -f ../__root__stratux-pre-start.sh /root/stratux-pre-start.sh
cp -f rc.local /etc/rc.local

#kalibrate-rtl
cd /root
rm -rf kalibrate-rtl
git clone https://github.com/steve-m/kalibrate-rtl
cd kalibrate-rtl
./bootstrap
./configure
make -j8
make install
cd /root && rm -rf kalibrate-rtl

#disable serial console
sed -i /boot/cmdline.txt -e "s/console=serial0,[0-9]\+ //"

#Set the keyboard layout to US.
sed -i /etc/default/keyboard -e "/^XKBLAYOUT/s/\".*\"/\"us\"/"

# Finally, try to reduce writing to SD card as much as possible, so they don't get bricked when yanking the power cable
# Disable swap...
systemctl disable dphys-swapfile
apt purge -y dphys-swapfile
apt autoremove -y

# Mount logs/tmp stuff as tmpfs
echo "" >> /etc/fstab # newline
echo "tmpfs    /var/log    tmpfs    defaults,noatime,nosuid,mode=0755,size=100m    0 0" >> /etc/fstab
echo "tmpfs    /tmp        tmpfs    defaults,noatime,nosuid,size=100m    0 0" >> /etc/fstab
echo "tmpfs    /var/tmp    tmpfs    defaults,noatime,nosuid,size=30m    0 0" >> /etc/fstab

# Now also prepare the update file..
cd /root/stratux/selfupdate
./makeupdate.sh

