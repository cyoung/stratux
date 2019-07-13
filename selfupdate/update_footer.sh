cp -f gen_gdl90 /usr/bin/gen_gdl90
chmod 755 /usr/bin/gen_gdl90
cp -f libdump978.so /usr/lib/libdump978.so
chmod 655 /usr/bin/gen_gdl90

# Startup script.
RASPBIAN_VERSION=`cat /etc/debian_version`
if test "$RASPBIAN_VERSION" = "8.0" ; then
	# Install the systemd startup scripts in any case, even if they won't be used. If this is being run, then the old init.d script
	#  is still intact and we just leave it. If running Wheezy, then remove the old init.d script.
	rm -f /etc/init.d/stratux
	rm -f /etc/rc2.d/S01stratux
	rm -f /etc/rc6.d/K01stratux
fi

cp -f __lib__systemd__system__stratux.service /lib/systemd/system/stratux.service
cp -f __root__stratux-pre-start.sh /root/stratux-pre-start.sh
chmod 644 /lib/systemd/system/stratux.service
chmod 744 /root/stratux-pre-start.sh
ln -fs /lib/systemd/system/stratux.service /etc/systemd/system/multi-user.target.wants/stratux.service

#wifi config
cp -f hostapd.conf /etc/hostapd/hostapd.conf

#rsyslog config
cp -f rsyslog_d_stratux /etc/rsyslog.d/stratux.conf

#logrotate config
cp -f logrotate.conf /etc/logrotate.conf
cp -f logrotate_d_stratux /etc/logrotate.d/stratux

#WiFi Hostapd ver test and hostapd.conf builder script
cp -f stratux-wifi.sh /usr/sbin/
chmod 755 /usr/sbin/stratux-wifi.sh

#WiFi Config Manager
cp -f hostapd_manager.sh /usr/sbin/
chmod 755 /usr/sbin/hostapd_manager.sh

#SDR Serial Script
cp -f sdr-tool.sh /usr/sbin/
chmod 755 /usr/sbin/sdr-tool.sh

#boot config
cp -f config.txt /boot/config.txt

#rc.local
cp -f rc.local /etc/

#disable serial console
sed -i /boot/cmdline.txt -e "s/console=ttyAMA0,[0-9]\+ //"

#modprobe.d blacklist
cp -f rtl-sdr-blacklist.conf /etc/modprobe.d/

#udev config
cp -f 10-stratux.rules /etc/udev/rules.d
cp -f 99-uavionix.rules /etc/udev/rules.d

#go setup
cp -f bashrc.txt /root/.bashrc
cp -f stxAliases.txt /root/.stxAliases

# /etc/modules
cp -f modules.txt /etc/modules

#motd
cp -f motd /etc/motd

#fan control utility
#remove old script
rm -f /usr/bin/fancontrol.py
#install new program
/usr/bin/fancontrol stop
/usr/bin/fancontrol remove
cp -f fancontrol /usr/bin/
chmod 755 /usr/bin/fancontrol
/usr/bin/fancontrol install

cp -f dump1090 /usr/bin/
chmod 755 /usr/bin/dump1090

# Install libwiringpi
cp -f libwiringPi.so /usr/lib/

# AHRS approx data.
cp -f ahrs_table.log /root/
cp -f ahrs_approx /usr/bin/
chmod 755 /usr/bin/ahrs_approx

# DHCPD Config.
cp -f dhcpd.conf /etc/dhcp/dhcpd.conf
cp -f dhcpd.conf.template /etc/dhcp/dhcpd.conf.template

# Interfaces file.
cp -f interfaces /etc/network/interfaces
cp -f interfaces.template /etc/network/interfaces.template

# Web files install.
cd web/ && make stratuxBuild=${stratuxBuild}

# Remove old Wi-Fi watcher script.
rm -f /usr/sbin/wifi_watch.sh
sed -i "/\bwifi_watch\b/d" /etc/rc.local

cd /
rm -rf /root/stratux-update
