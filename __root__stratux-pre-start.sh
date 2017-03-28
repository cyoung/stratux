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

