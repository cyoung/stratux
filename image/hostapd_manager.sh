#!/bin/bash
######################################################################
#                      STRATUX HOSTAPD MANAGER                      #
######################################################################

#Set Script Name variable
SCRIPT=`basename ${BASH_SOURCE[0]}`

#Initialize variables to default values.
OPT_S=false
OPT_C=false
OPT_E=false
OPT_O=false
OPT_P=false
OPT_Q=false
defaultPass="SquawkDirtyToMe!"

parm="*"
err="####"
att="+++"

#Set fonts for Help.
BOLD=$(tput bold)
STOT=$(tput smso)
UNDR=$(tput smul)
REV=$(tput rev)
RED=$(tput setaf 1)
GREEN=$(tput setaf 2)
YELLOW=$(tput setaf 3)
MAGENTA=$(tput setaf 5)
WHITE=$(tput setaf 7)
NORM=$(tput sgr0)
NORMAL=$(tput sgr0)

#Help function
function HELP {
  echo -e \\n"Help documentation for ${BOLD}${SCRIPT}.${NORM}"\\n
  echo -e "${REV}Basic usage:${NORM} ${BOLD}$SCRIPT -s ssid -c chan -p pass ${NORM}"\\n
  echo "The following command line switches are recognized."
  echo "${REV}-s${NORM}  --Sets the SSID to ${BOLD}ssid${NORM}. \"-s stratux\""
  echo "${REV}-c${NORM}  --Sets the channel to ${BOLD}chan${NORM}. \"-c 1\""
  echo "${REV}-o${NORM}  --Turns off encryption and sets network to open. Cannot be used with -e or -p."
  echo "${REV}-e${NORM}  --Turns on encryption with passphrase ${BOLD}$defaultPass{NORM}. Cannot be used with -o or -p"
  echo "${REV}-p${NORM}  --Turns on encryption with your chosen passphrase ${BOLD}pass${NORM}. 8-63 Printable Characters(ascii 32-126). Cannot be used with -o or -e. \"-p password!\""
  echo "${REV}-q${NORM}  --Run silently. Still a work in progress, but quieter."
  echo -e "${REV}-h${NORM}  --Displays this help message. No further functions are performed."\\n
  echo -e "Example: ${BOLD}$SCRIPT -s Stratux-N3558D -c 5 -p SquawkDirty!${NORM}"\\n
  exit 1
}

confirm() {
	# call with a prompt string or use a default
	read -r -p "$1 " response
	case "$response" in
		[yY][eE][sS]|[yY]) 
		true
	;;
		*)
		exit 1
	;;
	esac
}


#apply settings and restart all processes
function APPLYSETTINGSLOUD {
	echo "${RED}${BOLD} $att At this time the script will restart your WiFi services.${WHITE}${NORMAL}"
	echo "If you are connected to Stratux through the ${BOLD}192.168.10.1${NORMAL} interface then you will be disconnected"
	echo "Please wait 1 min and look for the new SSID on your wireless device."
	sleep 3
	echo "${YELLOW}$att Restarting Stratux WiFi Services... $att ${WHITE}"
	echo "${YELLOW}$att SSH will now  disconnect if connected to http://192.168.1.10 ... $att ${WHITE}"
	echo "Killing hostapd..."
	sleep 2
	/usr/bin/killall -9 hostapd hostapd-edimax
	echo "Killed..."
	echo ""
	echo "Killing DHCP Server..."
	echo ""
	/usr/sbin/service isc-dhcp-server stop
	sleep 0.5
	echo "Killed..."
	echo ""
	echo "ifdown wlan0..."
	ifdown wlan0
	sleep 0.5
	echo "ifup wlan0..."
	echo "Calling Stratux WiFI Start Script(stratux-wifi.sh)..."
	ifup wlan0
	sleep 0.5
	echo ""
	echo ""
	echo "All systems should be up and running and you should see your new SSID!"
}

function APPLYSETTINGSQUIET {
	sleep 2
	/usr/bin/killall -9 hostapd hostapd-edimax
	sleep 1
	/usr/sbin/service isc-dhcp-server stop
	sleep 0.5
	ifdown wlan0
	sleep 0.5
	ifup wlan0
	sleep 0.5
}

clear

echo ""
echo "#### Stratux HOSTAPD Settings ####"
echo ""

if [ $(whoami) != 'root' ]; then
	echo "${BOLD}${RED}This script must be executed as root, exiting...${WHITE}${NORMAL}"
	echo "${BOLD}${RED}USAGE${WHITE}${NORMAL}"
	exit 1
fi

#Check the number of arguments. If none are passed, print help and exit.
NUMARGS=$#
if [ $NUMARGS -eq 0 ]; then
  HELP
fi

### Start getopts code ###

#Parse command line flags
#If an option should be followed by an argument, it should be followed by a ":".
#Notice there is no ":" after "eoqh". The leading ":" suppresses error messages from
#getopts. This is required to get my unrecognized option code to work.
options=':s:c:p:eoqh'
#options=':s:c:h'
while getopts $options option; do
  case $option in
    s)  #set option "s"
      if [[ -z "${OPTARG}" || "${OPTARG}" == *[[:space:]]* || "${OPTARG}" == -* ]]; then
          echo "${BOLD}${RED}$err No SSID for -s, exiting...${WHITE}${NORMAL}"
          exit 1
      else
          OPT_S=$OPTARG
          echo "$parm SSID Option -s used: $OPT_S"
          echo "${GREEN}    SSID will now be ${BOLD}${UNDR}$OPT_S${NORMAL}.${WHITE}"
      fi
      ;;
    c)  #set option "c"
      if [[ -z "${OPTARG}" || "${OPTARG}" == *[[:space:]]* || "${OPTARG}" == -* ]]; then
          echo "${BOLD}${RED}$err Channel option(-c) used without value, exiting... ${WHITE}${NORMAL}"
          exit 1
      else
          OPT_C=$OPTARG
          echo "$parm Channel option -c used: $OPT_C"
          if [[ "$OPT_C" =~ ^[0-9]+$ ]] && [ "$OPT_C" -ge 1 -a "$OPT_C" -le 13  ]; then
          	echo "${GREEN}    Channel will now be set to ${BOLD}${UNDR}$OPT_C${WHITE}${NORMAL}."
          else
            echo "${BOLD}${RED}$err Channel is not within acceptable values, exiting...${WHITE}${NORMAL}"
            exit 1
          fi
      fi
      ;;
    e)  #set option "e" with default passphrase
		if [[ -z "${OPTARG}" || "${OPTARG}" == *[[:space:]]* || "${OPTARG}" == -* ]]; then
          echo "$parm Encrypted WiFI Option -e used."
          OPT_E=$defaultPass
		  echo "${GREEN}     WiFi will be encrypted using ${BOLD}${UNDR}$OPT_E${NORMAL}${GREEN} as the passphrase!${WHITE}${NORMAL}"
      else
          echo "${BOLD}${RED}$err Option -e does not require arguement.${WHITE}${NORMAL}"
          exit 1
      fi
      ;;
	p) #set encryption with user specified passphrase
		if [[ -z "${OPTARG}" || "${OPTARG}" =~ ^[[:space:]]*$ || "${OPTARG}" == -* ]]; then
			echo "${BOLD}${RED}$err Encryption option(-p) used without passphrase!${WHITE}${NORMAL}"
			echo "${BOLD}${RED}$err Encryption option(-p) required an arguement \"-p passphrase\" ${WHITE}${NORMAL}"
		else
			OPT_P=$OPTARG
		fi
		echo "$parm Encryption option -p used:"
		if [ -z `echo $OPT_P | tr -d "[:print:]"` ] && [ ${#OPT_P} -ge 8 ]  && [ ${#OPT_P} -le 63 ]; then
			echo "${GREEN}     WiFi will be encrypted using ${BOLD}${UNDR}$OPT_P${NORMAL}${GREEN} as the passphrase!${WHITE}${NORMAL}"
			else
			echo  "${BOLD}${RED}$err Invalid PASSWORD: 8 - 63 printable characters, exiting...${WHITE}${NORMAL}"
		exit 1
		fi
    ;;
    o)  #set option "o"
      if [[ -z "${OPTARG}" || "${OPTARG}" == *[[:space:]]* || "${OPTARG}" == -* ]]; then
          echo "$parm Open WiFI Option -o used."
          echo "${GREEN}    WiFi will be set to ${BOLD}${UNDR}OPEN${NORMAL}${GREEN} or ${BOLD}${UNDR}UNSECURE${WHITE}${NORMAL}"
          OPT_O=true
      else
          echo "${BOLD}${RED}$err Option -o does not require arguement.${WHITE}${NORMAL}"
          exit 1
      fi
      ;;
    q)	 #set Quiet mode
		OPT_Q=true
      ;;
    h)  #show help
      HELP
      ;;
   \?) # invalid option
     echo "${BOLD}${RED}$err Invalid option -$OPTARG ${WHITE}${NORMAL}" >&2
     HELP
     exit 1
     ;;
   :) # Missing Arg 
     echo "${BOLD}${RED}$err Missing option for argument -$OPTARG ${WHITE}${NORMAL}" >&2
     HELP
     exit 1
     ;;
   *) # Invalid
     echo "${BOLD}${RED}$err Unimplemented option -$OPTARG ${WHITE}${NORMAL}" >&2
     HELP
     exit 1
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
  echo "${BOLD}${RED}$err Option -e , -p and -o cannot be used simultaneously ${WHITE}${NORMAL}"
  exit 1
fi

if [ $OPT_P != false ] && [ $OPT_E != false ]; then
  echo "${BOLD}${RED}$err Option -e and -p cannot be used simultaneously ${WHITE}${NORMAL}"
  exit 1
fi

echo ""
echo "${BOLD}No errors found. Continuning...${NORMAL}"
echo ""

if [ $OPT_Q == false ]; then
	confirm "Are you ready to apply these settings? [y/n]"
fi

# files to edit
HOSTAPD=('/etc/hostapd/hostapd.user')

####
#### File modification loop
####
for i in "${HOSTAPD[@]}"
do
  if [ -f ${i} ]; then
    echo "Working on $i..."
    if [ $OPT_S != false ]; then
    	echo "${MAGENTA}Setting ${YELLOW}SSID${MAGENTA} to ${YELLOW}$OPT_S ${MAGENTA}in $i...${WHITE}"
        if grep -q "^ssid=" ${HOSTAPD[$x]}; then
        sed -i "s/^ssid=.*/ssid=${OPT_S}/" ${i}
      else
        echo ${OPT_S} >> ${i}
      fi
    fi

    if [ $OPT_C != false ]; then
    	echo "${MAGENTA}Setting ${YELLOW}Channel${MAGENTA} to ${YELLOW}$OPT_C ${MAGENTA}in $i...${WHITE}"
        if grep -q "^channel=" ${i}; then
            sed -i "s/^channel=.*/channel=${OPT_C}/" ${i}
        else
            echo ${OPT_C} >> ${i}
        fi
    fi

    if [ $OPT_E != false ]; then
    	echo "${MAGENTA}Adding WPA encryption with passphrase: ${YELLOW}$OPT_E ${MAGENTA}to $i...${WHITE}"
        if grep -q "^#auth_algs=" ${i}; then
        	#echo "uncomenting wpa"
            sed -i "s/^#auth_algs=.*/auth_algs=1/" ${i}
            sed -i "s/^#wpa=.*/wpa=3/" ${i}
            sed -i "s/^#wpa_passphrase=.*/wpa_passphrase=$OPT_E/" ${i}
            sed -i "s/^#wpa_key_mgmt=.*/wpa_key_mgmt=WPA-PSK/" ${i}
            sed -i "s/^#wpa_pairwise=.*/wpa_pairwise=TKIP/" ${i}
            sed -i "s/^#rsn_pairwise=.*/rsn_pairwise=CCMP/" ${i}
       elif grep -q "^auth_algs=" ${i}; then
        	#echo "rewriting existing wpa"
            sed -i "s/^auth_algs=.*/auth_algs=1/" ${i}
            sed -i "s/^wpa=.*/wpa=3/" ${i}
            sed -i "s/^wpa_passphrase=.*/wpa_passphrase=$OPT_E/" ${i}
            sed -i "s/^wpa_key_mgmt=.*/wpa_key_mgmt=WPA-PSK/" ${i}
            sed -i "s/^wpa_pairwise=.*/wpa_pairwise=TKIP/" ${i}
            sed -i "s/^rsn_pairwise=.*/rsn_pairwise=CCMP/" ${i}
       else
#      		#echo "adding wpa"
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
       	echo "${MAGENTA}Removing WPA encryption in $i...${WHITE}"
        if grep -q "^auth_algs=" ${i}; then
        	#echo "comenting out wpa"
            sed -i "s/^auth_algs=.*/#auth_algs=1/" ${i}
            sed -i "s/^wpa=.*/#wpa=3/" ${i}
            sed -i "s/^wpa_passphrase=.*/#wpa_passphrase=$defaultPass/" ${i}
            sed -i "s/^wpa_key_mgmt=.*/#wpa_key_mgmt=WPA-PSK/" ${i}
            sed -i "s/^wpa_pairwise=.*/#wpa_pairwise=TKIP/" ${i}
            sed -i "s/^rsn_pairwise=.*/#rsn_pairwise=CCMP/" ${i}
        elif grep -q "^#auth_algs=" ${i}; then
        	#echo "rewriting comentied out wpa"
            sed -i "s/^#auth_algs=.*/#auth_algs=1/" ${i}
            sed -i "s/^#wpa=.*/#wpa=3/" ${i}
            sed -i "s/^#wpa_passphrase=.*/#wpa_passphrase=$defaultPass/" ${i}
            sed -i "s/^#wpa_key_mgmt=.*/#wpa_key_mgmt=WPA-PSK/" ${i}
            sed -i "s/^#wpa_pairwise=.*/#wpa_pairwise=TKIP/" ${i}
            sed -i "s/^#rsn_pairwise=.*/#rsn_pairwise=CCMP/" ${i}
        else
        	#echo "adding commented out WPA"
        	echo "" >> ${i}
        	echo "#auth_algs=1" >> ${i}
			echo "#wpa=3" >> ${i}
			echo "#wpa_passphrase=$defaultPass" >> ${i}
			echo "#wpa_key_mgmt=WPA-PSK" >> ${i}
            echo "#wpa_pairwise=TKIP" >> ${i}
			echo "#rsn_pairwise=CCMP" >> ${i}
        fi
    fi

	echo "${GREEN}Modified ${i}...done${WHITE}"
	echo ""
  else
   	echo "${MAGENTA}No ${i} file found...${WHITE}${NORMAL}"
	echo ""
  fi
done



### End main loop ###

### Apply Settings and restart all services

if [ $OPT_Q == false ]; then
	APPLYSETTINGSLOUD
else
	APPLYSETTINGSQUIET
fi

exit 0
