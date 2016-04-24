#!/bin/bash
#
# mtk_config.sh: Script to set up MTK3339 receiver for Stratux. 
# Resets receiver to 9600 and 1 Hz GPRMC messaging, then enables 
# WAAS, 5 Hz position reporting, the NMEA messages needed by
# Stratux, and 38400 bps serial output.

printf "About to configure MTK3339 receiver on /dev/ttyAMA0.\n"
printf "Press ctrl-C to abort or any other key to continue.\n"
read

# Iterate through common bitrates and send commands to reduce output to 1 Hz / 9600 bps.
printf "Setting MTK and RPi baud rate of /dev/ttyAMA0 to 9600. Iterating through common rates.\n"
printf "Current /dev/ttyAMA0 baudrate.\n"
printf "\$PMTK220,1000*1F\r\n" > /dev/ttyAMA0
printf "\$PMTK251,9600*17\r\n" > /dev/ttyAMA0
sleep 0.2
printf "38400 bps.\n"
stty -F /dev/ttyAMA0 38400
printf "\$PMTK220,1000*1F\r\n" > /dev/ttyAMA0
printf "\$PMTK251,9600*17\r\n" > /dev/ttyAMA0
sleep 0.2
printf "115200 bps.\n"
printf "\$PMTK220,1000*1F\r\n" > /dev/ttyAMA0
stty -F /dev/ttyAMA0 115200
printf "\$PMTK220,1000*1F\r\n" > /dev/ttyAMA0
printf "\$PMTK251,9600*17\r\n" > /dev/ttyAMA0
sleep 0.2
printf "57600 bps.\n"
stty -F /dev/ttyAMA0 57600
printf "\$PMTK220,1000*1F\r\n" > /dev/ttyAMA0
printf "\$PMTK251,9600*17\r\n" > /dev/ttyAMA0
sleep 0.2
printf "19200 bps.\n"
stty -F /dev/ttyAMA0 19200
printf "\$PMTK220,1000*1F\r\n" > /dev/ttyAMA0
printf "\$PMTK251,9600*17\r\n" > /dev/ttyAMA0
sleep 0.2
printf "4800 bps.\n"
stty -F /dev/ttyAMA0 4800
printf "\$PMTK220,1000*1F\r\n" > /dev/ttyAMA0
printf "\$PMTK251,9600*17\r\n" > /dev/ttyAMA0
sleep 0.2


stty -F /dev/ttyAMA0 9600
printf "\$PMTK314,0,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0*29\r\n" > /dev/ttyAMA0

printf "MTK has been set to 9600 baud with RMC messages at 1 Hz.\n"
printf "Press ctrl-C to abort, or any other key enter to continue with setup.\n"
read

# Now start the Stratux setup.
printf "Sending MTK command to set GPS baud rate to 38400\n"
printf "\$PMTK251,38400*27\r\n" > /dev/ttyAMA0

printf "Setting RPi baud rate of /dev/ttyAMA0 to 38400\n"
stty -F /dev/ttyAMA0 38400
sleep 0.2
printf "Sending MTK command to configure NMEA message output\n"
printf "\$PMTK314,0,1,1,1,5,5,0,0,0,0,0,0,0,0,0,0,0,0,0*29\r\n" > /dev/ttyAMA0
sleep 0.2
printf "Sending MTK commands to enable WAAS\n"
printf "\$PMTK301,2*2E\r\n" > /dev/ttyAMA0
sleep 0.2
printf "\$PMTK513,1*28\r\n" > /dev/ttyAMA0
sleep 0.2
printf "Sending MTK commands to enable 5 Hz position reporting\n"
printf "\$PMTK220,200*2C\r\n" > /dev/ttyAMA0

# Finally, test the connection.
printf "Opening /dev/ttyAMA0 to listen to GPS. Press ctrl-C to cancel.\n"
cat /dev/ttyAMA0