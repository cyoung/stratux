#!/bin/bash

echo powersave >/sys/devices/system/cpu/cpu0/cpufreq/scaling_governor

# Check if we need to run an update.
if [ -e /root/update*stratux*v*.sh ] ; then
	UPDATE_SCRIPT=`ls -1t /root/update*stratux*v*.sh | head -1`
	if [ -n "$UPDATE_SCRIPT" ] ; then
		# Execute the script, remove it, then reboot.
		echo
		echo "Running update script ${UPDATE_SCRIPT}..."
		bash ${UPDATE_SCRIPT}
		rm -f $UPDATE_SCRIPT
		reboot
	fi
fi

##### Script for setting up new file structure for hostapd settings
##### Look for hostapd.user and if found do nothing.
##### If not assume because of previous version and convert to new file structure
DAEMON_USER_PREF=/etc/hostapd/hostapd.user
if [ ! -f $DAEMON_USER_PREF ]; then
	DAEMON_CONF=/etc/hostapd/hostapd.conf
    DAEMON_CONF_EDIMAX=/etc/hostapd/hostapd-edimax.conf
	HOSTAPD_VALUES=('ssid=' 'channel=' 'auth_algs=' 'wpa=' 'wpa_passphrase=' 'wpa_key_mgmt=' 'wpa_pairwise=' 'rsn_pairwise=')
	HOSTAPD_VALUES_RM=('#auth_algs=' '#wpa=' '#wpa_passphrase=' '#wpa_key_mgmt=' '#wpa_pairwise=' '#rsn_pairwise=')
	
	for i in "${HOSTAPD_VALUES[@]}" 
	do
    	if grep -q "^$i" $DAEMON_CONF
		then
        	grep "^$i" $DAEMON_CONF >> $DAEMON_USER_PREF
        	sed -i '/^'"$i"'/d' $DAEMON_CONF
			sed -i '/^'"$i"'/d' $DAEMON_CONF_EDIMAX
		fi
	done
	for i in "${HOSTAPD_VALUES_RM[@]}"
	do
    	if grep -q "^$i" $DAEMON_CONF
		then
			sed -i '/^'"$i"'/d' $DAEMON_CONF
			sed -i '/^'"$i"'/d' $DAEMON_CONF_EDIMAX
		fi
	done
    sleep 1     #make sure there is time to get the file written before checking for it again
    # If once the code above runs and there is still no hostapd.user file then something is wrong and we will just create the file with basic settings. 
    # Any more then this they somebody was messing with things and its not our fault things are this bad
	if [ ! -f $DAEMON_USER_PREF ]; then 
		echo "ssid=stratux" >> $DAEMON_USER_PREF
	    echo "channel=1" >> $DAEMON_USER_PREF
	fi
fi
##### End hostapd settings structure script
