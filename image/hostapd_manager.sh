#!/bin/bash
# Log in as root and copy this file then pass the
# desired ssid as a command line option, e.g.:
#
#
# bash hostapd_manager.sh 1 ssid chan passphrase
#
# bash hostapd_manager.sh 0 stratux-100 6 secret
#

RED=$(tput setaf 1)
YELLOW=$(tput setaf 3)
MAGENTA=$(tput setaf 5)
WHITE=$(tput setaf 7)
NORMAL=$(tput sgr0)

function USAGE {
	echo ""
    echo "usage: $0 quiet ssid channel passphrase"
    echo "      quiet                   will the script output messages 1 = silent"
    echo "	ssid			current or new SSID -required-"
    echo "	channel			the channel you want the WIFI to operate on -required-"
    echo "	passphrase		code to login to wifi. If not provided then security will be turned off"
    echo ""
    exit 1
}

#echo ${#4}



#### root user check
if [ $(whoami) != 'root' ]; then
    if [ $1 != "1" ]; then
    	echo "${BOLD}${RED}This script must be executed as root, exiting...${WHITE}${NORMAL}"
    	echo "${BOLD}${RED}USAGE${WHITE}${NORMAL}"
    fi
    exit
fi

if [ $# -eq 0 ]; then
	USAGE
    exit
fi

#### ssid option check
####
SSID=
if [ "$2" = '' ]; then
    if [ $1 != "1" ]; then
    	echo "${BOLD}${RED}Missing SSID option, exiting...${WHITE}${NORMAL}"
    	echo "${BOLD}${RED}USAGE${WHITE}${NORMAL}"
    fi
    exit
else
    SSID=ssid=$2
fi

#### channel option check
####
CHAN=
if [[ "$3" =~ ^[0-9]+$ ]] && [ "$3" -ge 1 -a "$3" -le 13  ]; then
    	CHAN=channel=$3
else
	if [ $1 != "1" ]; then
    	echo "${BOLD}${RED}Incorrect CHANNEL $3 not valid (number from 1 to 13), exiting...${WHITE}${NORMAL}"
    	echo "${BOLD}${RED}USAGE${WHITE}${NORMAL}"
    fi
   	exit
fi

#### encription option check
####
PASS=
if [ ! -z "$4" ]; then
	if [ -z `echo $4 | tr -d "[:print:]"` ] && [ ${#4} -ge 8 ]  && [ ${#4} -le 63 ]; then
  		PASS=wpa_passphrase=$4
	else
    	if [ $1 != "1" ]; then
			echo  "${BOLD}${RED}Invalid PASSWORD: 8 - 63 printable characters, exiting...${WHITE}${NORMAL}"
        	echo "${BOLD}${RED}USAGE${WHITE}${NORMAL}"
        fi
		exit
	fi 
fi

STATUS=
HOSTAPD=/etc/hostapd/hostapd.conf
HOSTAPDEDIMAX=/etc/hostapd/hostapd-edimax.conf

####
#### /etc/hostapd/hostapd.conf
####
if [ -f "$HOSTAPD" ]; then
    if [ $1 != "1" ]; then
    	echo "${MAGENTA}Setting ${YELLOW}SSID${MAGENTA} to ${YELLOW}$2 ${MAGENTA}in $HOSTAPD...${WHITE}"
    fi

    if grep -q "^ssid=" ${HOSTAPD}; then
        sed -i "s/^ssid=.*/${SSID}/" ${HOSTAPD}
    else
        echo ${SSID} >> ${HOSTAPD}
    fi
    
    if [ $1 != "1" ]; then
    	echo "${MAGENTA}Setting Channel to ${YELLOW}$3 ${MAGENTA}in $HOSTAPD...${WHITE}"
    fi

    if grep -q "^channel=" ${HOSTAPD}; then
        sed -i "s/^channel=.*/${CHAN}/" ${HOSTAPD}
    else
        echo ${CHAN} >> ${HOSTAPD}
    fi


    if [ ! -z "$4" ]; then
    	if [ $1 != "1" ]; then
        	echo "${MAGENTA}Adding WPA encryption with passphrase: ${YELLOW}$4 ${MAGENTA}to $HOSTAPD...${WHITE}"
        fi
        if grep -q "^#auth_algs=" ${HOSTAPD}; then
        	#echo "uncomenting wpa"
            sed -i "s/^#auth_algs=.*/auth_algs=1/" ${HOSTAPD}
            sed -i "s/^#wpa=.*/wpa=3/" ${HOSTAPD}
            sed -i "s/^#wpa_passphrase=.*/wpa_passphrase=$4/" ${HOSTAPD}
            sed -i "s/^#wpa_key_mgmt=.*/wpa_key_mgmt=WPA-PSK/" ${HOSTAPD}
            sed -i "s/^#wpa_pairwise=.*/wpa_pairwise=TKIP/" ${HOSTAPD}
            sed -i "s/^#rsn_pairwise=.*/rsn_pairwise=CCMP/" ${HOSTAPD}
       elif grep -q "^auth_algs=" ${HOSTAPD}; then
        	#echo "rewriting existing wpa"
            sed -i "s/^auth_algs=.*/auth_algs=1/" ${HOSTAPD}
            sed -i "s/^wpa=.*/wpa=3/" ${HOSTAPD}
            sed -i "s/^wpa_passphrase=.*/wpa_passphrase=$4/" ${HOSTAPD}
            sed -i "s/^wpa_key_mgmt=.*/wpa_key_mgmt=WPA-PSK/" ${HOSTAPD}
            sed -i "s/^wpa_pairwise=.*/wpa_pairwise=TKIP/" ${HOSTAPD}
            sed -i "s/^rsn_pairwise=.*/rsn_pairwise=CCMP/" ${HOSTAPD}
       else
       		#echo "adding wpa"
       		echo "" >> ${HOSTAPD}
            echo "auth_algs=1" >> ${HOSTAPD}
			echo "wpa=3" >> ${HOSTAPD}
			echo "wpa_passphrase=$3" >> ${HOSTAPD}
			echo "wpa_key_mgmt=WPA-PSK" >> ${HOSTAPD}
            echo "wpa_pairwise=TKIP" >> ${HOSTAPD}
			echo "rsn_pairwise=CCMP" >> ${HOSTAPD}
        fi
    else
    	if [ $1 != "1" ]; then
        	echo "${MAGENTA}Removing WPA encryption in $HOSTAPD...${WHITE}"
        fi
        if grep -q "^auth_algs=" ${HOSTAPD}; then
        	#echo "comenting out wpa"
            sed -i "s/^auth_algs=.*/#auth_algs=1/" ${HOSTAPD}
            sed -i "s/^wpa=.*/#wpa=3/" ${HOSTAPD}
            sed -i "s/^wpa_passphrase=.*/#wpa_passphrase=Clearedfortakeoff/" ${HOSTAPD}
            sed -i "s/^wpa_key_mgmt=.*/#wpa_key_mgmt=WPA-PSK/" ${HOSTAPD}
            sed -i "s/^wpa_pairwise=.*/#wpa_pairwise=TKIP/" ${HOSTAPD}
            sed -i "s/^rsn_pairwise=.*/#rsn_pairwise=CCMP/" ${HOSTAPD}
        elif grep -q "^#auth_algs=" ${HOSTAPD}; then
        	#echo "rewriting comentied out wpa"
            sed -i "s/^#auth_algs=.*/#auth_algs=1/" ${HOSTAPD}
            sed -i "s/^#wpa=.*/#wpa=3/" ${HOSTAPD}
            sed -i "s/^#wpa_passphrase=.*/#wpa_passphrase=Clearedfortakeoff/" ${HOSTAPD}
            sed -i "s/^#wpa_key_mgmt=.*/#wpa_key_mgmt=WPA-PSK/" ${HOSTAPD}
            sed -i "s/^#wpa_pairwise=.*/#wpa_pairwise=TKIP/" ${HOSTAPD}
            sed -i "s/^#rsn_pairwise=.*/#rsn_pairwise=CCMP/" ${HOSTAPD}
        else
        	#echo "adding commented out WPA"
        	echo "" >> ${HOSTAPD}
        	echo "#auth_algs=1" >> ${HOSTAPD}
			echo "#wpa=3" >> ${HOSTAPD}
			echo "#wpa_passphrase=Clearedfortakeoff" >> ${HOSTAPD}
			echo "#wpa_key_mgmt=WPA-PSK" >> ${HOSTAPD}
            echo "#wpa_pairwise=TKIP" >> ${HOSTAPD}
			echo "#rsn_pairwise=CCMP" >> ${HOSTAPD}
        fi
        
    fi
	if [ $1 != "1" ]; then
    	echo "${GREEN}...done${WHITE}"
    fi
    STATUS="${YELLOW}Don't forget to reboot...${WHITE}"
else
	if [ $1 != "1" ]; then
    	echo "${MAGENTA}${RED}No ${HOSTAPD} file found...${WHITE}${NORMAL}"
    fi
fi

####
#### /etc/hostapd/hostapd-edimax.conf
####
if [ -f "$HOSTAPDEDIMAX" ]; then
	if [ $1 != "1" ]; then
    	echo "${MAGENTA}Setting ${YELLOW}SSID ${MAGENTA}to ${YELLOW}$2 ${MAGENTA}in $HOSTAPD...${WHITE}"
    fi

    if grep -q "^ssid=" "${HOSTAPDEDIMAX}"; then
        sed -i "s/^ssid=.*/${SSID}/" ${HOSTAPDEDIMAX}
    else
        echo ${SSID} >> ${HOSTAPDEDIMAX}
    fi

	if [ $1 != "1" ]; then
    	echo "${MAGENTA}Setting Channel to ${YELLOW}$3 ${MAGENTA}in $HOSTAPD...${WHITE}"
    fi

    if grep -q "^channel=" ${HOSTAPD}; then
        sed -i "s/^channel=.*/${CHAN}/" ${HOSTAPD}
    else
        echo ${CHAN} >> ${HOSTAPD}
    fi


    if [ ! -z "$4" ]; then
    	if [ $1 != "1" ]; then
        	echo "${MAGENTA}Adding WPA encryption with passphrase: ${YELLOW}$4 ${MAGENTA}to $HOSTAPD...${WHITE}"
        fi
        if grep -q "^#auth_algs=" ${HOSTAPD}; then
        	#echo "uncomenting wpa"
            sed -i "s/^#auth_algs=.*/auth_algs=1/" ${HOSTAPD}
            sed -i "s/^#wpa=.*/wpa=3/" ${HOSTAPD}
            sed -i "s/^#wpa_passphrase=.*/wpa_passphrase=$4/" ${HOSTAPD}
            sed -i "s/^#wpa_key_mgmt=.*/wpa_key_mgmt=WPA-PSK/" ${HOSTAPD}
            sed -i "s/^#wpa_pairwise=.*/wpa_pairwise=TKIP/" ${HOSTAPD}
            sed -i "s/^#rsn_pairwise=.*/rsn_pairwise=CCMP/" ${HOSTAPD}
       elif grep -q "^auth_algs=" ${HOSTAPD}; then
        	#echo "rewriting existing wpa"
            sed -i "s/^auth_algs=.*/auth_algs=1/" ${HOSTAPD}
            sed -i "s/^wpa=.*/wpa=3/" ${HOSTAPD}
            sed -i "s/^wpa_passphrase=.*/wpa_passphrase=$4/" ${HOSTAPD}
            sed -i "s/^wpa_key_mgmt=.*/wpa_key_mgmt=WPA-PSK/" ${HOSTAPD}
            sed -i "s/^wpa_pairwise=.*/wpa_pairwise=TKIP/" ${HOSTAPD}
            sed -i "s/^rsn_pairwise=.*/rsn_pairwise=CCMP/" ${HOSTAPD}
       else
       		#echo "adding wpa"
       		echo "" >> ${HOSTAPD}
            echo "auth_algs=1" >> ${HOSTAPD}
			echo "wpa=3" >> ${HOSTAPD}
			echo "wpa_passphrase=$4" >> ${HOSTAPD}
			echo "wpa_key_mgmt=WPA-PSK" >> ${HOSTAPD}
            echo "wpa_pairwise=TKIP" >> ${HOSTAPD}
			echo "rsn_pairwise=CCMP" >> ${HOSTAPD}
        fi
    else
    	if [ $1 != "1" ]; then
        	echo "${MAGENTA}Removing WPA encryption in $HOSTAPD...${WHITE}"
        fi
        if grep -q "^auth_algs=" ${HOSTAPD}; then
        	#echo "comenting out wpa"
            sed -i "s/^auth_algs=.*/#auth_algs=1/" ${HOSTAPD}
            sed -i "s/^wpa=.*/#wpa=3/" ${HOSTAPD}
            sed -i "s/^wpa_passphrase=.*/#wpa_passphrase=Clearedfortakeoff/" ${HOSTAPD}
            sed -i "s/^wpa_key_mgmt=.*/#wpa_key_mgmt=WPA-PSK/" ${HOSTAPD}
            sed -i "s/^wpa_pairwise=.*/#wpa_pairwise=TKIP/" ${HOSTAPD}
            sed -i "s/^rsn_pairwise=.*/#rsn_pairwise=CCMP/" ${HOSTAPD}
        elif grep -q "^#auth_algs=" ${HOSTAPD}; then
        	#echo "rewriting comentied out wpa"
            sed -i "s/^#auth_algs=.*/#auth_algs=1/" ${HOSTAPD}
            sed -i "s/^#wpa=.*/#wpa=3/" ${HOSTAPD}
            sed -i "s/^#wpa_passphrase=.*/#wpa_passphrase=Clearedfortakeoff/" ${HOSTAPD}
            sed -i "s/^#wpa_key_mgmt=.*/#wpa_key_mgmt=WPA-PSK/" ${HOSTAPD}
            sed -i "s/^#wpa_pairwise=.*/#wpa_pairwise=TKIP/" ${HOSTAPD}
            sed -i "s/^#rsn_pairwise=.*/#rsn_pairwise=CCMP/" ${HOSTAPD}
        else
        	#echo "adding commented out WPA"
        	echo "" >> ${HOSTAPD}
        	echo "#auth_algs=1" >> ${HOSTAPD}
			echo "#wpa=3" >> ${HOSTAPD}
			echo "#wpa_passphrase=Clearedfortakeoff" >> ${HOSTAPD}
			echo "#wpa_key_mgmt=WPA-PSK" >> ${HOSTAPD}
            echo "#wpa_pairwise=TKIP" >> ${HOSTAPD}
			echo "#rsn_pairwise=CCMP" >> ${HOSTAPD}
        fi
    fi

    if [ $1 != "1" ]; then
    	echo "${GREEN}...done${WHITE}"
    fi
    STATUS="${YELLOW}Don't forget to reboot...${WHITE}"
else
    if [ $1 != "1" ]; then
    	echo "${MAGENTA}${RED}No ${HOSTAPDEDIMAX} found...${WHITE}${NORMAL}"
    fi
fi

if [ $1 != "1" ]; then
	echo
	echo $STATUS
	echo
fi
