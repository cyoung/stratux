#!/bin/bash

# DO NOT CALL ME DIRECTLY!
# This script is called by mk_europe_edition.sh via qemu
set -ex

mount -t proc proc /proc

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
ln -s /bin/true /root/fake/deb-systemd-invoke

# Fake a proc FS for raspberrypi-sys-mods_20170519_armhf... Extend me as needed
mkdir -p /proc/sys/vm/

apt update
apt clean

PATH=/root/fake:$PATH RUNLEVEL=1 apt install --yes libjpeg62-turbo-dev libconfig9 rpi-update dnsmasq git cmake  \
    libusb-1.0-0-dev build-essential autoconf libtool i2c-tools libfftw3-dev libncurses-dev python3-serial jq ifplugd iptables libttspico-utils bluez bluez-firmware

# Compile latest bluez.. version shipping with current RPiOS is buggy in peripheral mode..
# Note we only install it additionally, so new bluetoothd will be used, but config files from debian archive.
PATH=/root/fake:$PATH RUNLEVEL=1 apt install --yes libusb-dev libdbus-1-dev libglib2.0-dev libudev-dev libical-dev libreadline-dev python3-pygments # needed to compile bluez
cd /root/
wget -O- https://github.com/bluez/bluez/archive/refs/tags/5.76.tar.gz | tar xz
cd bluez-5.76
./bootstrap && ./configure --disable-manpages && make -j4 && make install
cd ..
rm -r bluez-5.76
PATH=/root/fake:$PATH RUNLEVEL=1 apt autoremove --purge --yes libusb-dev libdbus-1-dev libglib2.0-dev libudev-dev libical-dev libreadline-dev python3-pygments
systemctl enable bluetooth

# try to reduce writing to SD card as much as possible, so they don't get bricked when yanking the power cable
# Disable swap...
systemctl disable dphys-swapfile
apt purge -y dphys-swapfile
apt autoremove -y
apt clean
#echo y | rpi-update


systemctl enable ssh
systemctl disable dnsmasq # we start it manually on respective interfaces
#systemctl disable hciuart
systemctl disable triggerhappy
systemctl disable wpa_supplicant
systemctl disable systemd-timesyncd # We sync time with GPS. Make sure there is no conflict if we have internet connection
systemctl disable resize2fs_once


systemctl disable apt-daily.timer
systemctl disable apt-daily-upgrade.timer
systemctl disable man-db.timer

# Run DHCP on eth0 when cable is plugged in
sed -i -e 's/INTERFACES=""/INTERFACES="eth0"/g' /etc/default/ifplugd

# Generate ssh key for all installs. Otherwise it would have to be done on each boot, which takes a couple of seconds
ssh-keygen -A -v
systemctl disable regenerate_ssh_host_keys
# This is usually done by the console-setup service that takes quite long of first boot..
/lib/console-setup/console-setup.sh



cd /root/stratux
cp image/bashrc.txt /root/.bashrc
source /root/.bashrc

# Prepare librtlsdr. The one shipping with buster uses usb_zerocopy, which is extremely slow on newer kernels, so
# we manually compile the osmocom version that disables zerocopy by default..
cd /root/
rm -rf rtl-sdr
git clone https://github.com/osmocom/rtl-sdr.git
cd rtl-sdr
git checkout 1261fbb285297da08f4620b18871b6d6d9ec2a7b # Aug. 23, 2023
cp rtl-sdr.rules /etc/udev/rules.d/
mkdir build && cd build
cmake .. -DENABLE_ZEROCOPY=0
make -j1
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


# Prepare wiringpi for ogn trx via GPIO
cd /root && git clone https://github.com/WiringPi/WiringPi.git
cd WiringPi && WIRINGPI_SUDO="" ./build
cd /root && rm -r WiringPi

# Debian seems to ship with an invalid pkgconfig for librtlsdr.. fix it:
#sed -i -e 's/prefix=/prefix=\/usr/g' /usr/lib/arm-linux-gnueabihf/pkgconfig/librtlsdr.pc
#sed -i -e 's/libdir=/libdir=${prefix}\/lib\/arm-linux-gnueabihf/g' /usr/lib/arm-linux-gnueabihf/pkgconfig/librtlsdr.pc

# Install golang
cd /root
wget -O- https://go.dev/dl/go1.20.1.linux-arm64.tar.gz | tar xz


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
rm -rf /root/.cache

##### Some device setup - copy files from image directory ####
cd /root/stratux/image
#motd
cp -f motd /etc/motd

#network default config. TODO: can't we just implement gen_gdl90 -write_network_settings or something to generate them from template?
cp -f stratux-dnsmasq.conf /etc/dnsmasq.d/stratux-dnsmasq.conf
cp -f wpa_supplicant_ap.conf /etc/wpa_supplicant/wpa_supplicant_ap.conf
cp -f interfaces /etc/network/interfaces

#sshd config
cp -f sshd_config /etc/ssh/sshd_config

#debug aliases
cp -f stxAliases.txt /root/.stxAliases

#rtl-sdr setup
cp -f rtl-sdr-blacklist.conf /etc/modprobe.d/

#system tweaks
cp -f modules.txt /etc/modules

#boot settings
cp -f config.txt /boot/firmware/

#Create default pi password as in old times, and disable initial user creation
systemctl disable userconfig
echo "pi:raspberry" | chpasswd

#rootfs overlay stuff
cp -f overlayctl init-overlay /sbin/
overlayctl install
# init-overlay replaces raspis initial partition size growing.. Make sure we call that manually (see init-overlay script)
touch /var/grow_root_part
mkdir -p /overlay/robase # prepare so we can bind-mount root even if overlay is disabled

# So we can import network settings if needed
touch /boot/firmware/.stratux-first-boot

#startup scripts
cp -f rc.local /etc/rc.local

# Optionally mount /dev/sda1 as /var/log - for logging to USB stick
echo -e "\n/dev/sda1             /var/log        auto    defaults,nofail,noatime,x-systemd.device-timeout=1ms  0       2" >> /etc/fstab

#disable serial console, disable rfkill state restore, enable wifi on boot
sed -i /boot/firmware/cmdline.txt -e "s/console=serial0,[0-9]\+ /systemd.restore_state=0 rfkill.default_state=1 /"
sed -i 's/quiet//g' /boot/firmware/cmdline.txt

#Set the keyboard layout to US.
sed -i /etc/default/keyboard -e "/^XKBLAYOUT/s/\".*\"/\"us\"/"

# Legacy stratux.conf path so it can be found easily..
ln -s /boot/firmware/stratux.conf /boot/stratux.conf

# Set hostname
echo "stratux" > /etc/hostname
sed -i /etc/hosts -e "s/raspberrypi/stratux/g"

# Clean up source tree - we don't need it at runtime
rm -r /root/stratux


# Uninstall packages we don't need, clean up temp stuff
rm -rf /root/go /root/go_path /root/.cache

PATH=/root/fake:$PATH RUNLEVEL=1 apt autoremove --purge --yes alsa-ucm-conf alsa-topology-conf cifs-utils cmake cmake-data \
    v4l-utils rsync pigz pi-bluetooth cpp zlib1g-dev network-manager apparmor autotools-dev automake autoconf build-essential gcc-12 \
    git mkvtoolnix gdb 


apt clean

rm -rf /var/cache/apt

rm -r /root/fake


umount /proc
