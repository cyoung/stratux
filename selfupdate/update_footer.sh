cp -ra stratux /opt/ 

# Startup script.
RASPBIAN_VERSION=`cat /etc/debian_version`

cp -f __lib__systemd__system__stratux.service /lib/systemd/system/stratux.service
chmod 644 /lib/systemd/system/stratux.service
chmod 744 /root/stratux-pre-start.sh
ln -fs /lib/systemd/system/stratux.service /etc/systemd/system/multi-user.target.wants/stratux.service

#rsyslog config
cp -f rsyslog_d_stratux /etc/rsyslog.d/stratux.conf

#logrotate config
cp -f logrotate.conf /etc/logrotate.conf
cp -f logrotate_d_stratux /etc/logrotate.d/stratux

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
/opt/stratux/bin/fancontrol stop
/opt/stratux/bin/fancontrol remove
/opt/stratux/bin/fancontrol install

cd /
rm -rf /root/stratux-update
