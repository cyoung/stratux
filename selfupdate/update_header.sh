#!/bin/bash

SCRIPT=$(realpath $0)

rm -rf /root/stratux-update
mkdir -p /root/stratux-update
cd /root/stratux-update
rm -f /var/log/stratux*

# Extract archive from below
echo "Extracting archive.."
ARCHIVE=`awk '/^__ARCHIVE_BELOW__/ {print NR + 1; exit 0; }' $SCRIPT`
tail -n +$ARCHIVE $SCRIPT | tar xjf -
echo "Extracting done. Installing"

# Need to stop fancontrol to install new version
/opt/stratux/bin/fancontrol stop
/opt/stratux/bin/fancontrol remove

cp -ra stratux /opt/ 

# Startup script.
RASPBIAN_VERSION=`cat /etc/debian_version`

cp -f __lib__systemd__system__stratux.service /lib/systemd/system/stratux.service
chmod 644 /lib/systemd/system/stratux.service
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
/opt/stratux/bin/fancontrol install

# overlayctl
cp -f overlayctl init-overlay /sbin/
/sbin/overlayctl install

# Make sure apt cache doesn't fill up the overlay
systemctl disable apt-daily.timer
systemctl disable apt-daily-upgrade.timer

# cleanup after switch to overlayfs: remove tmpfs lines from fstab, move stratux.conf to /boot and potentially enable the overlay depending on user settings
cat /etc/fstab | grep -v tmpfs > /tmp/fstab
mv /tmp/fstab /etc/fstab
mv /etc/stratux.conf /boot/stratux.conf

# Add optional usb stick mount if it's not already there
if [ "$(grep /dev/sda1 /etc/fstab)" == "" ]; then
    echo -e "\n/dev/sda1             /var/log        auto    defaults,nofail,noatime,x-systemd.device-timeout=1ms  0       2" >> /etc/fstab
fi

cd /
rm -rf /root/stratux-update

# re-enable overlay if it is configured. TODO: switch to jq for json parsing in the future once it's available in all installations
if [ "$(cat /boot/stratux.conf | grep 'PersistentLogging.:true')" != "" ]; then
    /sbin/overlayctl disable
else
    /sbin/overlayctl enable
fi
mkdir -p /overlay/robase

exit 0

# After this line there needs to be EXACTLY ONE NEWLINE!
__ARCHIVE_BELOW__
