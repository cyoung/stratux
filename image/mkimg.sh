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

#motd
cp -f motd mnt/etc/motd

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

#SDR Serial Script
cp -f sdr-tool.sh mnt/usr/sbin/sdr-tool.sh
chmod 755 mnt/usr/sbin/sdr-tool.sh

#ping udev
cp -f 99-uavionix.rules mnt/etc/udev/rules.d

#fan/temp control script
cp fancontrol.py mnt/usr/bin/
chmod 755 mnt/usr/bin/fancontrol.py

#isc-dhcp-server config
cp -f isc-dhcp-server mnt/etc/default/isc-dhcp-server

#sshd config
cp -f sshd_config mnt/etc/ssh/sshd_config

#udev config
cp -f 10-stratux.rules mnt/etc/udev/rules.d

#stratux files
cp -f ../libdump978.so mnt/usr/lib/libdump978.so

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
sed -i /boot/cmdline.txt -e "s/console=ttyAMA0,[0-9]\+ //"

#Set the keyboard layout to US.
sed -i /etc/default/keyboard -e "/^XKBLAYOUT/s/\".*\"/\"us\"/"

#boot settings
cp -f config.txt mnt/boot/

#external OLED screen
apt-get install -y libjpeg-dev i2c-tools python-smbus python-pip python-dev python-pil python-daemon screen
#for fancontrol.py:
pip install wiringpi
cd /root
git clone https://github.com/rm-hull/ssd1306
cd ssd1306 && python setup.py install
cp /root/stratux/test/screen/screen.py /usr/bin/stratux-screen.py
mkdir -p /etc/stratux-screen/
cp -f /root/stratux/test/screen/stratux-logo-64x64.bmp /etc/stratux-screen/stratux-logo-64x64.bmp
cp -f /root/stratux/test/screen/CnC_Red_Alert.ttf /etc/stratux-screen/CnC_Red_Alert.ttf

#startup scripts
cp -f ../__lib__systemd__system__stratux.service mnt/lib/systemd/system/stratux.service
cp -f ../__root__stratux-pre-start.sh mnt/root/stratux-pre-start.sh
cp -f rc.local mnt/etc/rc.local
