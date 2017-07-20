#!/bin/bash

######################################################################
#                   STRATUX QUIET HOSTAPD MANAGE                     #
######################################################################
# This script is almost identical to hostapd_manager.sh except all the
# screen outputs are supplressed except for error messages.
#
# Usage:
# hostapd_manager_quiet.sh -s Stratux-N12345 -p SquawkDirty! -c 5
# Command above sets the SSID to "Stratux-N12345, secures the network with the passphrase "SquawkDirty!, and changes the network channel to "5"
#
# hostapd_manager_quiet.sh -o
# Command above opens the network(removes any passphrase)
#
# hostapd_manager_quiet.sh -e
# Command above secures the WiFi network using the default passphrase "SquawkDirtyToMe!"

# Options:
# -s	--Sets the SSID to ${BOLD}ssid${NORM}. -s stratux
# -c	--Sets the channel to chan. -c 1
# -o	--Turns off encryption and sets network to open. Cannot be used with -e or -p.
# -e	--Turns on encryption with passphrase SquawkDirtyToMe!. Cannot be used with -o or -p
# -p	--Turns on encryption with your chosen passphrase pass. 8-63 Printable Characters(ascii 32-126). Cannot be used with -o or -e. -p password!
#
# Important:
# After each call of this script the wifi network will disconnect and restart all associated services to apply the changes

#Set Script Name variable
SCRIPT=`basename ${BASH_SOURCE[0]}`

#Initialize variables to default values.
OPT_S=false
OPT_C=false
OPT_E=false
OPT_O=false
OPT_P=false
OPT_Q=false
OPT_R=false
defaultPass="SquawkDirtyToMe!"

#apply settings and restart all processes
function APPLYSETTINGSQUIET {
	sleep 2
	TEMP=:$(/usr/bin/killall -9 hostapd hostapd-edimax > /dev/null 2>&1)
	sleep 1
	TEMP=:$(/usr/sbin/service isc-dhcp-server stop > /dev/null 2>&1)
	sleep 0.5
	TEMP=:$(ifdown wlan0)
	sleep 0.5
	TEMP=:$(ifup wlan0)
	sleep 0.5
}

function error_exit {
    echo "$1" >&2   ## Send message to stderr. Exclude >&2 if you don't want it that way.
    exit "${2:-1}"  ## Return a code specified by $2 or 1 by default.
}

if [ $(whoami) != 'root' ]; then
	echo "${BOLD}${RED}This script must be executed as root, exiting...${WHITE}${NORMAL}"
	echo "${BOLD}${RED}USAGE${WHITE}${NORMAL}"
	error_exit "Not Root"
fi

#Check the number of arguments. If none are passed, print help and exit.
NUMARGS=$#
if [ $NUMARGS -eq 0 ]; then
  error_exit "No Args Passed"
fi

### Start getopts code ###

#Parse command line flags
#If an option should be followed by an argument, it should be followed by a ":".
#Notice there is no ":" after "eoqh". The leading ":" suppresses error messages from
#getopts. This is required to get my unrecognized option code to work.
options=':s:c:p:eoh'
while getopts $options option; do
  case $option in
    s)  #set option "s"
      if [[ -z "${OPTARG}" || "${OPTARG}" == *[[:space:]]* || "${OPTARG}" == -* ]]; then
          error_exit "No SSID for -s, exiting..."
      else
          OPT_S=$OPTARG
      fi
      ;;
    c)  #set option "c"
      if [[ -z "${OPTARG}" || "${OPTARG}" == *[[:space:]]* || "${OPTARG}" == -* ]]; then
          error_exit "Channel option(-c) used without value, exiting... "
      else
		  OPT_C=$OPTARG
          if [[ "$OPT_C" =~ ^[0-9]+$ ]] && [ "$OPT_C" -ge 1 -a "$OPT_C" -le 13  ]; then
			OPT_C=$OPTARG
          else
            error_exit "Channel is not within acceptable values, exiting..."
          fi
      fi
      ;;
    e)  #set option "e" with default passphrase
		if [[ -z "${OPTARG}" || "${OPTARG}" == *[[:space:]]* || "${OPTARG}" == -* ]]; then
          OPT_E=$defaultPass
      else
          error_exit "Option -e does not require arguement.${WHITE}${NORMAL}"
      fi
      ;;
	p) #set encryption with user specified passphrase
		if [[ -z "${OPTARG}" || "${OPTARG}" =~ ^[[:space:]]*$ || "${OPTARG}" == -* ]]; then
			error_exit "Encryption option(-p) used without passphrase!"
		else
			OPT_P=$OPTARG
		fi
		echo "$parm Encryption option -p used:"
		if [ -z `echo $OPT_P | tr -d "[:print:]"` ] && [ ${#OPT_P} -ge 8 ]  && [ ${#OPT_P} -le 63 ]; then
			echo "${GREEN}     WiFi will be encrypted using ${BOLD}${UNDR}$OPT_P${NORMAL}${GREEN} as the passphrase!${WHITE}${NORMAL}"
		else
			error_exit "Invalid PASSWORD: 8 - 63 printable characters, exiting..."
		fi
    ;;
    o)  #set option "o"
      if [[ -z "${OPTARG}" || "${OPTARG}" == *[[:space:]]* || "${OPTARG}" == -* ]]; then
         OPT_O=true
      else
          error_exit  "${BOLD}${RED}$err Option -o does not require arguement. Exiting..."
      fi
      ;;
   \?) # invalid option
	error_exit "Invalid option -$OPTARG"
     ;;
   :) # Missing Arg
     error_exit "Missing option for argument -$OPTARG"
     ;;
   *) # Invalid
     error_exit "Unimplemented option -$OPTARG ${WHITE}${NORMAL}"
     ;;
  esac
done

shift $((OPTIND-1))  #This tells getopts to move on to the next argument.

### End getopts code ###


### Main loop to process files ###

#This is where your main file processing will take place. This example is just
#printing the files and extensions to the terminal. You should place any other
#file processing tasks within the while-do loop.

if [[ $OPT_O == true  && (  $OPT_E != false || $OPT_P != false ) ]]; then
  error_exit "Option -e , -p and -o cannot be used simultaneously"
fi

if [ $OPT_P != false ] && [ $OPT_E != false ]; then
  error_exit "Option -e and -p cannot be used simultaneously..."
fi

# files to edit
HOSTAPD=('/etc/hostapd/hostapd.user')

####
#### File modification loop
####
for i in "${HOSTAPD[@]}"
do
  if [ -f ${i} ]; then
    if [ $OPT_S != false ]; then
        if grep -q "^ssid=" ${HOSTAPD[$x]}; then
        sed -i "s/^ssid=.*/ssid=${OPT_S}/" ${i}
      else
        echo ${OPT_S} >> ${i}
      fi
    fi

    if [ $OPT_C != false ]; then
        if grep -q "^channel=" ${i}; then
            sed -i "s/^channel=.*/channel=${OPT_C}/" ${i}
        else
            echo ${OPT_C} >> ${i}
        fi
    fi

    if [ $OPT_E != false ]; then
        if grep -q "^#auth_algs=" ${i}; then
            sed -i "s/^#auth_algs=.*/auth_algs=1/" ${i}
            sed -i "s/^#wpa=.*/wpa=3/" ${i}
            sed -i "s/^#wpa_passphrase=.*/wpa_passphrase=$OPT_E/" ${i}
            sed -i "s/^#wpa_key_mgmt=.*/wpa_key_mgmt=WPA-PSK/" ${i}
            sed -i "s/^#wpa_pairwise=.*/wpa_pairwise=TKIP/" ${i}
            sed -i "s/^#rsn_pairwise=.*/rsn_pairwise=CCMP/" ${i}
       elif grep -q "^auth_algs=" ${i}; then
            sed -i "s/^auth_algs=.*/auth_algs=1/" ${i}
            sed -i "s/^wpa=.*/wpa=3/" ${i}
            sed -i "s/^wpa_passphrase=.*/wpa_passphrase=$OPT_E/" ${i}
            sed -i "s/^wpa_key_mgmt=.*/wpa_key_mgmt=WPA-PSK/" ${i}
            sed -i "s/^wpa_pairwise=.*/wpa_pairwise=TKIP/" ${i}
            sed -i "s/^rsn_pairwise=.*/rsn_pairwise=CCMP/" ${i}
       else
		echo "" >> ${i}
        echo "auth_algs=1" >> ${i}
		echo "wpa=3" >> ${i}
		echo "wpa_passphrase=$OPT_E" >> ${i}
		echo "wpa_key_mgmt=WPA-PSK" >> ${i}
        echo "wpa_pairwise=TKIP" >> ${i}
		echo "rsn_pairwise=CCMP" >> ${i}
        fi
    fi
    if [ $OPT_O != false ]; then
        if grep -q "^auth_algs=" ${i}; then
            sed -i "s/^auth_algs=.*/#auth_algs=1/" ${i}
            sed -i "s/^wpa=.*/#wpa=3/" ${i}
            sed -i "s/^wpa_passphrase=.*/#wpa_passphrase=$defaultPass/" ${i}
            sed -i "s/^wpa_key_mgmt=.*/#wpa_key_mgmt=WPA-PSK/" ${i}
            sed -i "s/^wpa_pairwise=.*/#wpa_pairwise=TKIP/" ${i}
            sed -i "s/^rsn_pairwise=.*/#rsn_pairwise=CCMP/" ${i}
        elif grep -q "^#auth_algs=" ${i}; then
            sed -i "s/^#auth_algs=.*/#auth_algs=1/" ${i}
            sed -i "s/^#wpa=.*/#wpa=3/" ${i}
            sed -i "s/^#wpa_passphrase=.*/#wpa_passphrase=$defaultPass/" ${i}
            sed -i "s/^#wpa_key_mgmt=.*/#wpa_key_mgmt=WPA-PSK/" ${i}
            sed -i "s/^#wpa_pairwise=.*/#wpa_pairwise=TKIP/" ${i}
            sed -i "s/^#rsn_pairwise=.*/#rsn_pairwise=CCMP/" ${i}
        else
            echo "" >> ${i}
            echo "#auth_algs=1" >> ${i}
            echo "#wpa=3" >> ${i}
            echo "#wpa_passphrase=$defaultPass" >> ${i}
            echo "#wpa_key_mgmt=WPA-PSK" >> ${i}
            echo "#wpa_pairwise=TKIP" >> ${i}
            echo "#rsn_pairwise=CCMP" >> ${i}
        fi
    fi
  else
    error_exit "No ${i} file found..."
  fi
done

### End main loop ###

### Apply Settings and restart all services

	APPLYSETTINGSQUIET

exit 1
