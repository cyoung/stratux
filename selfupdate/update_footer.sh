cp -f gen_gdl90 /usr/bin/gen_gdl90
cp -f libdump978.so /usr/lib/libdump978.so
cp -f libimu.so /usr/lib/libimu.so


# Startup script.
cp -f init.d-stratux /etc/init.d/stratux
chmod 755 /etc/init.d/stratux
ln -fs /etc/init.d/stratux /etc/rc2.d/S01stratux
ln -fs /etc/init.d/stratux /etc/rc6.d/K01stratux

#wifi config
cp -f hostapd.conf /etc/hostapd/hostapd.conf

#WiFi Config Manager
cp -f hostapd_manager.sh /usr/sbin/

#boot config
cp -f config.txt /boot/config.txt

#modprobe.d blacklist
cp -f rtl-sdr-blacklist.conf /etc/modprobe.d/

#udev config
cp -f 10-stratux.rules /etc/udev/rules.d

#go setup
cp -f bashrc.txt /root/.bashrc
cp -f stxAliases /root/.stxAliases

# /etc/modules
cp -f modules.txt /etc/modules

cp -f dump1090 /usr/bin/

# Web files install.
cd web/ && make stratuxBuild=${stratuxBuild}

# Remove old Wi-Fi watcher script.
rm -f /usr/sbin/wifi_watch.sh
sed -i "/\bwifi_watch\b/d" /etc/rc.local