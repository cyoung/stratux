#!/bin/bash
echo "Running"
echo powersave >/sys/devices/system/cpu/cpu0/cpufreq/scaling_governor

SCRIPT_MASK="update*stratux*v*.sh"
TEMP_LOCATION="/boot/StratuxUpdates/$SCRIPT_MASK"
UPDATE_LOCATION="/root/$SCRIPT_MASK"

echo "Temp Location $TEMP_LOCATION"

if [ -e ${TEMP_LOCATION} ]; then
	echo "Found Update Script in $TEMP_LOCATION$SCRIPT_MASK"
	TEMP_SCRIPT=`ls -1t ${TEMP_LOCATION} | head -1`
	echo "Moving Script $TEMP_SCRIPT"
	cp -r $TEMP_SCRIPT /root/
	chmod a+x $UPDATE_LOCATION
	rm -rf $TEMP_SCRIPT
fi

# Check if we need to run an update.
if [ -e ${UPDATE_LOCATION} ]; then
	echo "ls -1t ${UPDATE_LOCATION} | head -1"
	UPDATE_SCRIPT=`ls -1t ${UPDATE_LOCATION} | head -1`
	if [ -n ${UPDATE_SCRIPT} ] ; then
		# Execute the script, remove it, then reboot.
		echo
		echo "Running update script ${UPDATE_SCRIPT}..."
		bash ${UPDATE_SCRIPT}
		rm -f $UPDATE_SCRIPT
		reboot
	fi
fi
