#!/bin/bash


#kpartx -a asdfsadf.img

##dd if=/dev/zero bs=1M count=200 >>2016-02-26-raspbian-jessie-lite.img

#mount root partition
##mount -o loop,offset=67108864 2016-02-26-raspbian-jessie-lite.img mnt/
##resize2fs /dev/loop0
#mount boot partition
##mount -o loop,offset=4194304 2016-02-26-raspbian-jessie-lite.img mnt/boot/

chroot mnt/ apt-get update

#update firmware
chroot mnt/ apt-get -y install rpi-update
chroot mnt/ rpi-update
#wifi
chroot mnt/ apt-get install -y hostapd isc-dhcp-server
#troubleshooting
chroot mnt/ apt-get install -y tcpdump
#wifi startup
chroot mnt/ systemctl enable isc-dhcp-server

#disable ntpd autostart
chroot mnt/ systemctl disable ntp

#root key
cp -f root mnt/etc/ssh/authorized_keys/root
chown root.root mnt/etc/ssh/authorized_keys/root
chmod 644 mnt/etc/ssh/authorized_keys/root

#dhcpd config
cp -f dhcpd.conf mnt/etc/dhcp/dhcpd.conf

#hostapd config
cp -f hostapd.conf mnt/etc/hostapd/hostapd.conf
cp -f hostapd-edimax.conf mnt/etc/hostapd/hostapd-edimax.conf
#hostapd manager script
cp -f hostapd_manager.sh mnt/usr/sbin/hostapd_manager.sh
chmod 755 mnt/usr/sbin/hostapd_manager.sh
#hostapd
cp -f hostapd-edimax mnt/usr/sbin/hostapd-edimax
chmod 755 mnt/usr/sbin/hostapd-edimax
#remove hostapd startup scripts
rm -f mnt/etc/rc*.d/*hostapd mnt/etc/network/if-pre-up.d/hostapd mnt/etc/network/if-post-down.d/hostapd mnt/etc/init.d/hostapd mnt/etc/default/hostapd
#interface config
cp -f interfaces mnt/etc/network/interfaces
#custom hostapd start script
cp stratux-wifi.sh mnt/usr/sbin/
chmod 755 mnt/usr/sbin/stratux-wifi.sh

#isc-dhcp-server config
cp -f isc-dhcp-server mnt/etc/default/isc-dhcp-server

#sshd config
cp -f sshd_config mnt/etc/ssh/sshd_config

#udev config
cp -f 10-stratux.rules mnt/etc/udev/rules.d

#stratux files
cp -f ../libdump978.so mnt/usr/lib/libdump978.so
cp -f ../linux-mpu9150/libimu.so mnt/usr/lib/libimu.so

#go1.5.1 setup
cp -rf /root/go mnt/root/
cp -f bashrc.txt mnt/root/.bashrc

#debug aliases
cp -f stxAliases.txt mnt/root/.stxAliases

#rtl-sdr setup
cp -f rtl-sdr-blacklist.conf mnt/etc/modprobe.d/

chroot mnt/ apt-get install -y git cmake libusb-1.0-0.dev build-essential
rm -rf mnt/root/librtlsdr
git clone https://github.com/jpoirier/librtlsdr mnt/root/librtlsdr
mkdir -p mnt/root/librtlsdr/build
#FIXME
chroot mnt/ 'cd /root/librtlsdr/build && cmake ../ && make && make install && ldconfig'


#stratux setup
cd /root
apt-get install -y mercurial
apt-get install -y build-essential
rm -rf stratux
git clone https://github.com/cyoung/stratux --recursive
cd stratux
make
make install
systemctl enable stratux

#system tweaks
cp -f modules.txt mnt/etc/modules

#kalibrate-rtl
cd /root
rm -rf kalibrate-rtl
git clone https://github.com/steve-m/kalibrate-rtl
cd kalibrate-rtl
apt-get install -y autoconf fftw3 fftw3-dev libtool
./bootstrap
./configure
make
make install

#disable serial console
sed -i /etc/inittab -e "s|^.*:.*:respawn:.*ttyAMA0|#&|"

#Set the keyboard layout to US.
sed -i /etc/default/keyboard -e "/^XKBLAYOUT/s/\".*\"/\"us\"/"

#boot settings
cp -f config.txt mnt/boot/
