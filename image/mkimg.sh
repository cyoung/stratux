#!/bin/bash


kpartx -a asdfsadf.img

dd if=/dev/zero bs=1M count=200 >>2016-02-26-raspbian-jessie-lite.img

#mount root partition
mount -o loop,offset=67108864 2016-02-26-raspbian-jessie-lite.img mnt/
resize2fs /dev/loop0
#mount boot partition
mount -o loop,offset=4194304 2016-02-26-raspbian-jessie-lite.img mnt/boot/

#TODO
chroot mnt/

apt-get update

#update firmware
apt-get -y install rpi-update
rpi-update
#wifi
apt-get install -y hostapd isc-dhcp-server
#troubleshooting
apt-get install -y tcpdump
#wifi startup
systemctl enable isc-dhcp-server

#disable ntpd autostart
systemctl disbable ntp

#root key
cp -f root mnt/etc/ssh/authorized_keys/root
chown root.root mnt/etc/ssh/authorized_keys/root
chmod 644 mnt/etc/ssh/authorized_keys/root

#dhcpd config
cp -f dhcpd.conf mnt/etc/dhcp/dhcpd.conf

#hostapd config
cp -f hostapd.conf mnt/etc/hostapd/hostapd.conf
cp -f hostapd-edimax.conf mnt/etc/hostapd/hostapd-rpi3.conf
#hostapd
cp -f hostapd-edimax mnt/usr/sbin/hostapd-edimax
chmod 755 mnt/usr/sbin/hostapd-edimax
#remove hostapd startup scripts
rm -f mnt/etc/rc*.d/*hostapd mnt/etc/network/if-pre-up.d/hostapd mnt/etc/network/if-post-down.d/hostapd mnt/etc/init.d/hostapd mnt/etc/default/hostapd
#interface config
cp -f interfaces mnt/etc/network/interfaces
#custom hostapd start script
cp stratux-hostapd.sh mnt/usr/sbin/
chmod 755 mnt/usr/sbin/stratux-hostapd.sh


#isc-dhcp-server config
cp -f isc-dhcp-server mnt/etc/default/isc-dhcp-server

#sshd config
cp -f sshd_config mnt/etc/ssh/sshd_config

#stratux files
cp -f ../libdump978.so mnt/usr/lib/libdump978.so
cp -f ../linux-mpu9150/libimu.so mnt/usr/lib/libimu.so
cp -f rc.local mnt/etc/rc.local

#TODO:go setup
cp -rf /root/go mnt/root/
echo export PATH=/root/go/bin:\$\{PATH\} >>/root/.bashrc
echo export GOROOT=/root/go >>/root/.bashrc
echo export GOPATH=/root/go_path >>/root/.bashrc


#rtl-sdr setup
echo blacklist dvb_usb_rtl28xxu >>/etc/modprobe.d/rtl-sdr-blacklist.conf
echo blacklist e4000 >>/etc/modprobe.d/rtl-sdr-blacklist.conf
echo blacklist rtl2832 >>/etc/modprobe.d/rtl-sdr-blacklist.conf
apt-get install -y git cmake libusb-1.0-0.dev build-essential
cd /root
rm -rf librtlsdr
git clone https://github.com/jpoirier/librtlsdr
cd librtlsdr
mkdir build
cd build
cmake ../
make
make install
ldconfig


#stratux setup
cd /root
apt-get install -y mercurial
apt-get install -y build-essential
rm -rf stratux
git clone https://github.com/cyoung/stratux --recursive
cd stratux
make
systemctl enable stratux

#system tweaks
echo "i2c-bcm2708" >>/etc/modules
echo "i2c-dev" >>/etc/modules

#kalibrate-rtl
cd /root
git clone https://github.com/steve-m/kalibrate-rtl
cd kalibrate-rtl
apt-get install -y autoconf fftw3 fftw3-dev
apt-get install -y libtool
./bootstrap
./configure
make
make install

#disable serial console
sed -i /etc/inittab -e "s|^.*:.*:respawn:.*ttyAMA0|#&|"

#Set the keyboard layout to US.
sed -i /etc/default/keyboard -e "/^XKBLAYOUT/s/\".*\"/\"us\"/"
