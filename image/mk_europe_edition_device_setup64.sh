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
    libusb-1.0-0-dev build-essential autoconf libtool i2c-tools libfftw3-dev libncurses-dev python-serial jq

# try to reduce writing to SD card as much as possible, so they don't get bricked when yanking the power cable
# Disable swap...
systemctl disable dphys-swapfile
apt purge -y dphys-swapfile
apt autoremove -y
apt clean
#echo y | rpi-update

systemctl enable isc-dhcp-server
systemctl enable ssh
systemctl disable dhcpcd
systemctl disable hciuart
systemctl disable hostapd
systemctl disable apt-daily.timer
systemctl disable apt-daily-upgrade.timer

sed -i 's/INTERFACESv4=""/INTERFACESv4="wlan0"/g' /etc/default/isc-dhcp-server

rm -r /proc/*
rm -r /root/fake



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

# Debian seems to ship with an invalid pkgconfig for librtlsdr.. fix it:
#sed -i -e 's/prefix=/prefix=\/usr/g' /usr/lib/arm-linux-gnueabihf/pkgconfig/librtlsdr.pc
#sed -i -e 's/libdir=/libdir=${prefix}\/lib\/arm-linux-gnueabihf/g' /usr/lib/arm-linux-gnueabihf/pkgconfig/librtlsdr.pc

# Install golang
cd /root
wget https://golang.org/dl/go1.16.1.linux-arm64.tar.gz
tar xzf go1.16.1.linux-arm64.tar.gz
rm go1.16.1.linux-arm64.tar.gz

# Compile stratux
cd /root/stratux

make clean
make -j8

# Now also prepare the update file..
cd /root/stratux/selfupdate
./makeupdate.sh
mv /root/stratux/work/update-*.sh /root/
rm -r /root/stratux/work
cd /root/stratux


rm -r /root/go_path/* # safe space again..
make install

##### Some device setup - copy files from image directory ####
cd /root/stratux/image
#motd
cp -f motd /etc/motd

#network default config. TODO: can't we just implement gen_gdl90 -write_network_settings or something to generate them from template?
cp -f dhcpd.conf /etc/dhcp/dhcpd.conf
cp -f hostapd.conf /etc/hostapd/hostapd.conf
cp -f interfaces /etc/network/interfaces
cp -f isc-dhcp-server /etc/default/isc-dhcp-server

#logrotate conf
cp -f logrotate.conf /etc/logrotate.conf

#sshd config
# Do not copy for now. It contains many deprecated options and isn't needed.
cp -f sshd_config /etc/ssh/sshd_config

#debug aliases
cp -f stxAliases.txt /root/.stxAliases

#rtl-sdr setup
cp -f rtl-sdr-blacklist.conf /etc/modprobe.d/

#system tweaks
cp -f modules.txt /etc/modules

#boot settings
cp -f config.txt /boot/

#rootfs overlay stuff
cp -f overlayctl init-overlay /sbin/
overlayctl install
# init-overlay replaces raspis initial partition size growing.. Make sure we call that manually (see init-overlay script)
touch /var/grow_root_part
mkdir -p /overlay/robase # prepare so we can bind-mount root even if overlay is disabled

#startup scripts
cp -f rc.local /etc/rc.local

# Optionally mount /dev/sda1 as /var/log - for logging to USB stick
echo -e "\n/dev/sda1             /var/log        auto    defaults,nofail,noatime,x-systemd.device-timeout=1ms  0       2" >> /etc/fstab

#disable serial console
sed -i /boot/cmdline.txt -e "s/console=serial0,[0-9]\+ //"

#Set the keyboard layout to US.
sed -i /etc/default/keyboard -e "/^XKBLAYOUT/s/\".*\"/\"us\"/"

# Clean up source tree - we don't need it at runtime
rm -r /root/stratux


