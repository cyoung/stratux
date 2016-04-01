#!/bin/bash
# Log in as root and copy this file then pass the
# desired ssid as a command line option, e.g.:
#
#
# bash hostapd-mgr.sh ssid chan passphrase
#
# bash ssid-change.sh stratux-100 6 secrst
#

RED=$(tput setaf 1)
YELLOW=$(tput setaf 3)
MAGENTA=$(tput setaf 5)
WHITE=$(tput setaf 7)
NORMAL=$(tput sgr0)

function USAGE {
	echo ""
    echo "usage: $0 ssid channel passphrase"
    echo "	ssid			current or new SSID -required-"
    echo "	channel			the channel you want the WIFI to operate on -required-"
    echo "	passphrase		code to login to wifi. If not provided then security will be turned off"
    echo ""
    exit 1
}

#echo ${#4}



#### root user check
if [ $(whoami) != 'root' ]; then
    echo "${BOLD}${RED}This script must be executed as root, exiting...${WHITE}${NORMAL}"
    echo "${BOLD}${RED}USAGE${WHITE}${NORMAL}"
    exit
fi

if [ $# -eq 0 ]; then
	USAGE
    exit
fi

#### ssid option check
####
SSID=
if [ "$1" = '' ]; then
    echo "${BOLD}${RED}Missing SSID option, exiting...${WHITE}${NORMAL}"
    echo "${BOLD}${RED}USAGE${WHITE}${NORMAL}"
    exit
else
    SSID=ssid=$1
fi

#### channel option check
####
CHAN=
if [[ "$2" =~ ^[0-9]+$ ]] && [ "$2" -ge 1 -a "$2" -le 13  ]; then
    	CHAN=channel=$2
else
    echo "${BOLD}${RED}Incorrect CHANNEL(number from 1 to 13), exiting...${WHITE}${NORMAL}"
    echo "${BOLD}${RED}USAGE${WHITE}${NORMAL}"
    exit
fi

#### encription option check
####
PASS=
if [ ! -z "$3" ]; then
	if [ -z `echo $3 | tr -d "[:print:]"` ] && [ ${#3} -ge 8 ]  && [ ${#3} -le 63 ]; then
  		PASS=wpa_passphrase=$3
	else
		echo  "${BOLD}${RED}Invalid PASSWORD: 8 - 63 printable characters, exiting...${WHITE}${NORMAL}"
        echo "${BOLD}${RED}USAGE${WHITE}${NORMAL}"
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
    echo "${MAGENTA}Setting ${YELLOW}SSID${MAGENTA} to ${YELLOW}$1 ${MAGENTA}in $HOSTAPD...${WHITE}"

    if grep -q "^ssid=" ${HOSTAPD}; then
        sed -i "s/^ssid=.*/${SSID}/" ${HOSTAPD}
    else
        echo ${SSID} >> ${HOSTAPD}
    fi
    
    echo "${MAGENTA}Setting Channel to ${YELLOW}$2 ${MAGENTA}in $HOSTAPD...${WHITE}"

    if grep -q "^channel=" ${HOSTAPD}; then
        sed -i "s/^channel=.*/${CHAN}/" ${HOSTAPD}
    else
        echo ${CHAN} >> ${HOSTAPD}
    fi


    if [ ! -z "$3" ]; then
        echo "${MAGENTA}Adding WPA encryption with passphrase: ${YELLOW}$4 ${MAGENTA}to $HOSTAPD...${WHITE}"
        if grep -q "^#auth_algs=" ${HOSTAPD}; then
        	echo "uncomenting wpa"
            sed -i "s/^#auth_algs=.*/auth_algs=1/" ${HOSTAPD}
            sed -i "s/^#wpa=.*/wpa=3/" ${HOSTAPD}
            sed -i "s/^#wpa_passphrase=.*/wpa_passphrase=$3/" ${HOSTAPD}
            sed -i "s/^#wpa_key_mgmt=.*/wpa_key_mgmt=WPA-PSK/" ${HOSTAPD}
            sed -i "s/^#wpa_pairwise=.*/wpa_pairwise=TKIP/" ${HOSTAPD}
            sed -i "s/^#rsn_pairwise=.*/rsn_pairwise=CCMP/" ${HOSTAPD}
       elif grep -q "^auth_algs=" ${HOSTAPD}; then
        	echo "rewriting existing wpa"
            sed -i "s/^auth_algs=.*/auth_algs=1/" ${HOSTAPD}
            sed -i "s/^wpa=.*/wpa=3/" ${HOSTAPD}
            sed -i "s/^wpa_passphrase=.*/wpa_passphrase=$3/" ${HOSTAPD}
            sed -i "s/^wpa_key_mgmt=.*/wpa_key_mgmt=WPA-PSK/" ${HOSTAPD}
            sed -i "s/^wpa_pairwise=.*/wpa_pairwise=TKIP/" ${HOSTAPD}
            sed -i "s/^rsn_pairwise=.*/rsn_pairwise=CCMP/" ${HOSTAPD}
       else
       		echo "adding wpa"
       		echo "" >> ${HOSTAPD}
            echo "auth_algs=1" >> ${HOSTAPD}
			echo "wpa=3" >> ${HOSTAPD}
			echo "wpa_passphrase=$3" >> ${HOSTAPD}
			echo "wpa_key_mgmt=WPA-PSK" >> ${HOSTAPD}
            echo "wpa_pairwise=TKIP" >> ${HOSTAPD}
			echo "rsn_pairwise=CCMP" >> ${HOSTAPD}
        fi
    else
        echo "${MAGENTA}Removing WPA encryption in $HOSTAPD...${WHITE}"
        if grep -q "^auth_algs=" ${HOSTAPD}; then
        	echo "comenting out wpa"
            sed -i "s/^auth_algs=.*/#auth_algs=1/" ${HOSTAPD}
            sed -i "s/^wpa=.*/#wpa=3/" ${HOSTAPD}
            sed -i "s/^wpa_passphrase=.*/#wpa_passphrase=Clearedfortakeoff/" ${HOSTAPD}
            sed -i "s/^wpa_key_mgmt=.*/#wpa_key_mgmt=WPA-PSK/" ${HOSTAPD}
            sed -i "s/^wpa_pairwise=.*/#wpa_pairwise=TKIP/" ${HOSTAPD}
            sed -i "s/^rsn_pairwise=.*/#rsn_pairwise=CCMP/" ${HOSTAPD}
        elif grep -q "^#auth_algs=" ${HOSTAPD}; then
        	echo "rewriting comentied out wpa"
            sed -i "s/^#auth_algs=.*/#auth_algs=1/" ${HOSTAPD}
            sed -i "s/^#wpa=.*/#wpa=3/" ${HOSTAPD}
            sed -i "s/^#wpa_passphrase=.*/#wpa_passphrase=Clearedfortakeoff/" ${HOSTAPD}
            sed -i "s/^#wpa_key_mgmt=.*/#wpa_key_mgmt=WPA-PSK/" ${HOSTAPD}
            sed -i "s/^#wpa_pairwise=.*/#wpa_pairwise=TKIP/" ${HOSTAPD}
            sed -i "s/^#rsn_pairwise=.*/#rsn_pairwise=CCMP/" ${HOSTAPD}
        else
        	echo "adding commented out WPA"
        	echo "" >> ${HOSTAPD}
        	echo "#auth_algs=1" >> ${HOSTAPD}
			echo "#wpa=3" >> ${HOSTAPD}
			echo "#wpa_passphrase=Clearedfortakeoff" >> ${HOSTAPD}
			echo "#wpa_key_mgmt=WPA-PSK" >> ${HOSTAPD}
            echo "#wpa_pairwise=TKIP" >> ${HOSTAPD}
			echo "#rsn_pairwise=CCMP" >> ${HOSTAPD}
        fi
        
    fi

    echo "${GREEN}...done${WHITE}"
    STATUS="${YELLOW}Don't forget to reboot...${WHITE}"
else
    echo "${MAGENTA}${RED}No ${HOSTAPD} file found...${WHITE}${NORMAL}"
fi

####
#### /etc/hostapd/hostapd-edimax.conf
####
if [ -f "$HOSTAPDEDIMAX" ]; then
    echo "${MAGENTA}Setting ${YELLOW}SSID ${MAGENTA}to ${YELLOW}$1 ${MAGENTA}in $HOSTAPD...${WHITE}"

    if grep -q "^ssid=" "${HOSTAPDEDIMAX}"; then
        sed -i "s/^ssid=.*/${SSID}/" ${HOSTAPDEDIMAX}
    else
        echo ${SSID} >> ${HOSTAPDEDIMAX}
    fi

   echo "${MAGENTA}Setting Channel to ${YELLOW}$2 ${MAGENTA}in $HOSTAPD...${WHITE}"

    if grep -q "^channel=" ${HOSTAPD}; then
        sed -i "s/^channel=.*/${CHAN}/" ${HOSTAPD}
    else
        echo ${CHAN} >> ${HOSTAPD}
    fi


    if [ ! -z "$3" ]; then
        echo "${MAGENTA}Adding WPA encryption with passphrase: ${YELLOW}$4 ${MAGENTA}to $HOSTAPD...${WHITE}"
        if grep -q "^#auth_algs=" ${HOSTAPD}; then
        	echo "uncomenting wpa"
            sed -i "s/^#auth_algs=.*/auth_algs=1/" ${HOSTAPD}
            sed -i "s/^#wpa=.*/wpa=3/" ${HOSTAPD}
            sed -i "s/^#wpa_passphrase=.*/wpa_passphrase=$3/" ${HOSTAPD}
            sed -i "s/^#wpa_key_mgmt=.*/wpa_key_mgmt=WPA-PSK/" ${HOSTAPD}
            sed -i "s/^#wpa_pairwise=.*/wpa_pairwise=TKIP/" ${HOSTAPD}
            sed -i "s/^#rsn_pairwise=.*/rsn_pairwise=CCMP/" ${HOSTAPD}
       elif grep -q "^auth_algs=" ${HOSTAPD}; then
        	echo "rewriting existing wpa"
            sed -i "s/^auth_algs=.*/auth_algs=1/" ${HOSTAPD}
            sed -i "s/^wpa=.*/wpa=3/" ${HOSTAPD}
            sed -i "s/^wpa_passphrase=.*/wpa_passphrase=$3/" ${HOSTAPD}
            sed -i "s/^wpa_key_mgmt=.*/wpa_key_mgmt=WPA-PSK/" ${HOSTAPD}
            sed -i "s/^wpa_pairwise=.*/wpa_pairwise=TKIP/" ${HOSTAPD}
            sed -i "s/^rsn_pairwise=.*/rsn_pairwise=CCMP/" ${HOSTAPD}
       else
       		echo "adding wpa"
       		echo "" >> ${HOSTAPD}
            echo "auth_algs=1" >> ${HOSTAPD}
			echo "wpa=3" >> ${HOSTAPD}
			echo "wpa_passphrase=$3" >> ${HOSTAPD}
			echo "wpa_key_mgmt=WPA-PSK" >> ${HOSTAPD}
            echo "wpa_pairwise=TKIP" >> ${HOSTAPD}
			echo "rsn_pairwise=CCMP" >> ${HOSTAPD}
        fi
    else
        echo "${MAGENTA}Removing WPA encryption in $HOSTAPD...${WHITE}"
        if grep -q "^auth_algs=" ${HOSTAPD}; then
        	echo "comenting out wpa"
            sed -i "s/^auth_algs=.*/#auth_algs=1/" ${HOSTAPD}
            sed -i "s/^wpa=.*/#wpa=3/" ${HOSTAPD}
            sed -i "s/^wpa_passphrase=.*/#wpa_passphrase=Clearedfortakeoff/" ${HOSTAPD}
            sed -i "s/^wpa_key_mgmt=.*/#wpa_key_mgmt=WPA-PSK/" ${HOSTAPD}
            sed -i "s/^wpa_pairwise=.*/#wpa_pairwise=TKIP/" ${HOSTAPD}
            sed -i "s/^rsn_pairwise=.*/#rsn_pairwise=CCMP/" ${HOSTAPD}
        elif grep -q "^#auth_algs=" ${HOSTAPD}; then
        	echo "rewriting comentied out wpa"
            sed -i "s/^#auth_algs=.*/#auth_algs=1/" ${HOSTAPD}
            sed -i "s/^#wpa=.*/#wpa=3/" ${HOSTAPD}
            sed -i "s/^#wpa_passphrase=.*/#wpa_passphrase=Clearedfortakeoff/" ${HOSTAPD}
            sed -i "s/^#wpa_key_mgmt=.*/#wpa_key_mgmt=WPA-PSK/" ${HOSTAPD}
            sed -i "s/^#wpa_pairwise=.*/#wpa_pairwise=TKIP/" ${HOSTAPD}
            sed -i "s/^#rsn_pairwise=.*/#rsn_pairwise=CCMP/" ${HOSTAPD}
        else
        	echo "adding commented out WPA"
        	echo "" >> ${HOSTAPD}
        	echo "#auth_algs=1" >> ${HOSTAPD}
			echo "#wpa=3" >> ${HOSTAPD}
			echo "#wpa_passphrase=Clearedfortakeoff" >> ${HOSTAPD}
			echo "#wpa_key_mgmt=WPA-PSK" >> ${HOSTAPD}
            echo "#wpa_pairwise=TKIP" >> ${HOSTAPD}
			echo "#rsn_pairwise=CCMP" >> ${HOSTAPD}
        fi
    fi

    echo "${GREEN}...done${WHITE}"
    STATUS="${YELLOW}Don't forget to reboot...${WHITE}"
else
    echo "${MAGENTA}${RED}No ${HOSTAPDEDIMAX} found...${WHITE}${NORMAL}"
fi

echo
echo $STATUS
echo
