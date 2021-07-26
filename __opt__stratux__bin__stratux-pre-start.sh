#!/bin/bash

#echo powersave >/sys/devices/system/cpu/cpu0/cpufreq/scaling_governor

#Logging Function
SCRIPT=`basename ${BASH_SOURCE[0]}`
STX_LOG="/var/log/stratux.log"
function wLog () {
	echo "$(date +"%Y/%m/%d %H:%M:%S")  - $SCRIPT - $1" >> ${STX_LOG}
}
wLog "Running Stratux Updater Script."

SCRIPT_MASK="update*stratux*v*.sh"
TEMP_LOCATION="/boot/StratuxUpdates/$SCRIPT_MASK"
UPDATE_LOCATION="/root/$SCRIPT_MASK"

if [ -e ${TEMP_LOCATION} ]; then
	wLog "Found Update Script in $TEMP_LOCATION$SCRIPT_MASK"
	TEMP_SCRIPT=`ls -1t ${TEMP_LOCATION} | head -1`
	wLog "Moving Script $TEMP_SCRIPT"
	cp -r ${TEMP_SCRIPT} /root/
	wLog "Changing permissions to chmod a+x $UPDATE_LOCATION"
	chmod a+x ${UPDATE_LOCATION}
	wLog "Removing Update file from $TEMP_LOCATION"
	rm -rf ${TEMP_SCRIPT}
fi

# Check if we need to run an update.
if [ -e ${UPDATE_LOCATION} ]; then
	UPDATE_SCRIPT=`ls -1t ${UPDATE_LOCATION} | head -1`
	if [ -n ${UPDATE_SCRIPT} ] ; then
		# Execute the script, remove it, then reboot.
		wLog "Running update script ${UPDATE_SCRIPT}..."
		bash ${UPDATE_SCRIPT}
		wLog "Removing Update SH"
		rm -f ${UPDATE_SCRIPT}
		wLog "Finished... Rebooting... Bye"
		reboot
	fi
fi
wLog "Exited without updating anything..."
