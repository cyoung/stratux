#!/bin/bash
#
# sirf_config.sh: Script to set up BU-353-S4 receiver for Stratux. 
# Resets receiver to 4800 baud and 1 Hz GPRMC messaging, then enables 
# WAAS, 5 Hz position reporting, all  NMEA messages needed by
# Stratux, and 38400 bps serial output.

printf "About to configure SIRF receiver on /dev/ttyUSB0.\n"
printf "Press ctrl-C to abort or any other key to continue.\n"
read

# Iterate through common bitrates and send commands to reduce output to 1 Hz / 4800 bps.
printf "Setting SIRF and RPi baud rate of /dev/ttyUSB0 to 4800. Iterating through common rates.\n"
printf "Current /dev/ttyUSB0 baudrate.\n"
printf "\$PSRF103,00,7,00,0*22\r\n" > /dev/ttyUSB0
printf "\$PSRF100,1,4800,8,1,0*0E\r\n" > /dev/ttyUSB0
sleep 0.2
printf "38400 bps.\n"
stty -F /dev/ttyUSB0 38400
printf "\$PSRF103,00,7,00,0*22\r\n" > /dev/ttyUSB0
printf "\$PSRF100,1,4800,8,1,0*0E\r\n" > /dev/ttyUSB0
sleep 0.2
printf "115200 bps.\n"
printf "\$PSRF103,00,7,00,0*22\r\n" > /dev/ttyUSB0
stty -F /dev/ttyUSB0 115200
printf "\$PSRF103,00,7,00,0*22\r\n" > /dev/ttyUSB0
printf "\$PSRF100,1,4800,8,1,0*0E\r\n" > /dev/ttyUSB0
sleep 0.2
printf "57600 bps.\n"
stty -F /dev/ttyUSB0 57600
printf "\$PSRF103,00,7,00,0*22\r\n" > /dev/ttyUSB0
printf "\$PSRF100,1,4800,8,1,0*0E\r\n" > /dev/ttyUSB0
sleep 0.2
printf "19200 bps.\n"
stty -F /dev/ttyUSB0 19200
printf "\$PSRF103,00,7,00,0*22\r\n" > /dev/ttyUSB0
printf "\$PSRF100,1,4800,8,1,0*0E\r\n" > /dev/ttyUSB0
sleep 0.2
printf "9600 bps.\n"
stty -F /dev/ttyUSB0 9600
printf "\$PSRF103,00,7,00,0*22\r\n" > /dev/ttyUSB0
printf "\$PMTK251,4800*17\r\n" > /dev/ttyUSB0
sleep 0.2


stty -F /dev/ttyUSB0 4800
# GGA off:
printf "\$PSRF103,00,00,00,01*24\r\n" > /dev/ttyUSB0

# GLL off:
printf "\$PSRF103,01,00,00,01*27\r\n" > /dev/ttyUSB0

# GSA off:
printf "\$PSRF103,02,00,00,01*26\r\n" > /dev/ttyUSB0

# GSV off:
printf "\$PSRF103,03,00,00,01*27\r\n" > /dev/ttyUSB0

# RMC on:
printf "\$PSRF103,04,00,01,01*21\r\n" > /dev/ttyUSB0

# VTG off:
printf "\$PSRF103,05,00,00,01*21\r\n" > /dev/ttyUSB0

printf "SIRF device has been set to 4800 baud with RMC messages at 1 Hz.\n"
printf "Press ctrl-C to abort, or any other key enter to continue with setup.\n"
read

# Now start the Stratux setup.
printf "Sending Sirf PSRF100 command to set GPS baud rate to 38400\n"
printf "\$PSRF100,1,38400,8,1,0*3D\r\n" > /dev/ttyUSB0

printf "Resetting RPi baud rate of /dev/ttyUSB0 to 38400\n"
stty -F /dev/ttyUSB0 38400
sleep 0.2
printf "Sending SIRF PSRF103 commands to configure NMEA message output\n"

#GGA:
printf "\$PSRF103,00,00,01,01*25\r\n" > /dev/ttyUSB0

# Uncomment next two commands set GSA/GSV on each position message.
# Stratux doesn't need this much info - but keep for developer debug
# GSA (every position message):
#printf "\$PSRF103,02,00,01,01*27\r\n" > /dev/ttyUSB0
# GSV (every position message):
#printf "\$PSRF103,03,00,01,01*26\r\n" > /dev/ttyUSB0

# Next two commands set GSA/GSV on every 5th position message.
# Comment out (and uncomment above two commands) to report on
# every position.
# GSA (every 5 position messages):
printf "\$PSRF103,02,00,05,01*23\r\n" > /dev/ttyUSB0
# GSV (every 5 position messages):
printf "\$PSRF103,03,00,05,01*22\r\n" > /dev/ttyUSB0


# RMC:
printf "\$PSRF103,04,00,01,01*21\r\n" > /dev/ttyUSB0
# VTG:
printf "\$PSRF103,05,00,01,01*20\r\n" > /dev/ttyUSB0
sleep 0.2

printf "Sending SIRF PSRF151 command to enable WAAS\n"
printf "\$PSRF151,01*3F\r\n" > /dev/ttyUSB0
sleep 0.2

printf "Sending SIRF PSRF103 command to enable 5 Hz position reporting\n"
printf "\$PSRF103,00,6,00,0*23\r\n" > /dev/ttyUSB0

# Finally, test the connection.
printf "Opening /dev/ttyUSB0 to listen to GPS. Press ctrl-C to cancel.\n"
cat /dev/ttyUSB0
