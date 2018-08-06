#!/bin/bash
######################################################################
#                      STRATUX HOSTAPD MANAGER                      #
######################################################################

#Logging Function
SCRIPT=`basename ${BASH_SOURCE[0]}`
STX_LOG="/var/log/stratux.log"
function wLog () {
	echo "$(date +"%Y/%m/%d %H:%m:%S")  - $SCRIPT - $1" >> $STX_LOG
}
wLog "Running Hostapd Manager Script."

# files to edit
HOSTAPD=('/etc/hostapd/hostapd.user')

# values to be added to hostapd.user for security.
HOSTAPD_SECURE_VALUES_DELETE=('auth_algs=1' 'wpa=3' 'wpa_passphrase=' 'wpa_key_mgmt=WPA-PSK' 'wpa_pairwise=TKIP' 'rsn_pairwise=CCMP')

# 'wpa_passphrase=' was left out of this to set it with the $wifiPass. I assume you can not evaluate a variable from within an array variable
HOSTAPD_SECURE_VALUES_WRITE=('auth_algs=1' 'wpa=3' 'wpa_key_mgmt=WPA-PSK' 'wpa_pairwise=TKIP' 'rsn_pairwise=CCMP')

#Initialize variables to default values.
OPT_S=false
OPT_C=false
OPT_E=false
OPT_O=false
OPT_P=false
wifiPass="SquawkDirtyToMe!"

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
  echo "${REV}-e${NORM}  --Turns on encryption with passphrase ${BOLD}$wifiPass${NORM}. Cannot be used with -o or -p"
  echo "${REV}-p${NORM}  --Turns on encryption with your chosen passphrase ${BOLD}pass${NORM}. 8-63 Printable Characters(ascii 32-126). Cannot be used with -o or -e. \"-p password!\""
  echo -e "${REV}-h${NORM}  --Displays this help message. No further functions are performed."\\n
  echo -e "Example: ${BOLD}$SCRIPT -s Stratux-N3558D -c 5 -p SquawkDirty!${NORM}"\\n
  exit 1
}

function confirm() {
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

function cleanhostapd () {
	wLog "Cleaning hostapd config at $1"
	for j in "${HOSTAPD_SECURE_VALUES_DELETE[@]}"
	do
		sed -i "/$j/ d" ${1}
	done
	sed -i '/^\s*$/d' ${1}
}

function writehostapd () {
	wLog "Writing hostapd config at $1"
	sed -i '/^\s*$/d' ${1}
	echo "" >> ${1}
	for j in "${HOSTAPD_SECURE_VALUES_WRITE[@]}"
	do
		echo "${j}" >> ${1}
	done
	echo "wpa_passphrase=$wifiPass" >> ${1}
}

#apply settings and restart all processes
function APPLYSETTINGS {
	wLog "Restarting all wifi settings."
	echo "${RED}${BOLD} $att At this time the script will restart your WiFi services.${WHITE}${NORMAL}"
	echo "If you are connected to Stratux through the ${BOLD}192.168.10.1${NORMAL} interface then you will be disconnected"
	echo "Please wait up to 1 min and look for the new SSID on your wireless device."
	sleep 3
	echo "${YELLOW}$att Restarting Stratux WiFi Services... $att ${WHITE}"
	echo "${YELLOW}$att SSH will now disconnect if connected to http://192.168.10.1 ... $att ${WHITE}"
	echo "ifdown wlan0..."
	ifdown wlan0
	sleep 0.5
	echo "ifup wlan0..."
	echo "Calling Stratux WiFI Start Script(stratux-wifi.sh) via ifup wlan0..."
	ifup wlan0
	sleep 0.5
	echo ""
	echo ""
	echo "All systems should be up and running and you should see your new SSID!"
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
options=':s:c:p:eoh'
#options=':s:c:h'
while getopts $options option; do
  case $option in
    s)  #set option "s"
      if [[ -z "${OPTARG}" || "${OPTARG}" == *[[:space:]]* || "${OPTARG}" == -* ]]; then
          echo "${BOLD}${RED}$err No SSID for -s, exiting...${WHITE}${NORMAL}"
		  wLog "No SSID for -s, exiting..."
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
		  wLog "Channel option(-c) used without value, exiting..."
          exit 1
      else
          OPT_C=$OPTARG
          echo "$parm Channel option -c used: $OPT_C"
          if [[ "$OPT_C" =~ ^[0-9]+$ ]] && [ "$OPT_C" -ge 1 -a "$OPT_C" -le 13  ]; then
          	echo "${GREEN}    Channel will now be set to ${BOLD}${UNDR}$OPT_C${WHITE}${NORMAL}."
          else
            echo "${BOLD}${RED}$err Channel is not within acceptable values, exiting...${WHITE}${NORMAL}"
			wLog "Channel is not within acceptable values, exiting..."
            exit 1
          fi
      fi
      ;;
    e)  #set option "e" with default passphrase
		if [[ -z "${OPTARG}" || "${OPTARG}" == *[[:space:]]* || "${OPTARG}" == -* ]]; then
          echo "$parm Encrypted WiFI Option -e used."
          OPT_E=$wifiPass
		  echo "${GREEN}     WiFi will be encrypted using ${BOLD}${UNDR}$OPT_E${NORMAL}${GREEN} as the passphrase!${WHITE}${NORMAL}"
      else
          echo "${BOLD}${RED}$err Option -e does not require argument. exiting...${WHITE}${NORMAL}"
		  wLog "Option -e does not require argument."
          exit 1
      fi
      ;;
	p) #set encryption with user specified passphrase
		if [[ -z "${OPTARG}" || "${OPTARG}" =~ ^[[:space:]]*$ || "${OPTARG}" == -* ]]; then
			echo "${BOLD}${RED}$err Encryption option(-p) used without passphrase!${WHITE}${NORMAL}"
			echo "${BOLD}${RED}$err Encryption option(-p) required an argument \"-p passphrase\". exiting...${WHITE}${NORMAL}"
			wLog "Encryption option(-p) used without passphrase!"
		else
			OPT_P=$OPTARG
			wifiPass=$OPTARG
		fi
		echo "$parm Encryption option -p used:"
		if [ -z `echo $OPT_P | tr -d "[:print:]"` ] && [ ${#OPT_P} -ge 8 ]  && [ ${#OPT_P} -le 63 ]; then
			echo "${GREEN}     WiFi will be encrypted using ${BOLD}${UNDR}$OPT_P${NORMAL}${GREEN} as the passphrase!${WHITE}${NORMAL}"
		else
			echo  "${BOLD}${RED}$err Invalid PASSWORD: 8 - 63 printable characters, exiting...${WHITE}${NORMAL}"
			wLog "Invalid PASSWORD: 8 - 63 printable characters, exiting..."
			exit 1
		fi
    ;;
    o)  #set option "o"
      if [[ -z "${OPTARG}" || "${OPTARG}" == *[[:space:]]* || "${OPTARG}" == -* ]]; then
          echo "$parm Open WiFI Option -o used."
          echo "${GREEN}    WiFi will be set to ${BOLD}${UNDR}OPEN${NORMAL}${GREEN} or ${BOLD}${UNDR}UNSECURE${WHITE}${NORMAL}"
          OPT_O=true
      else
          echo "${BOLD}${RED}$err Option -o does not require argument. exiting...${WHITE}${NORMAL}"
		  wLog "Option -o does not require argument. exiting..."
          exit 1
      fi
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
  echo "${BOLD}${RED}$err Option -e , -p and -o cannot be used simultaneously. Exiting... ${WHITE}${NORMAL}"
  wLog "Option -e , -p and -o cannot be used simultaneously."
  exit 1
fi

if [ $OPT_P != false ] && [ $OPT_E != false ]; then
  echo "${BOLD}${RED}$err Option -e and -p cannot be used simultaneously. Exiting... ${WHITE}${NORMAL}"
  wLog "Option -e and -p cannot be used simultaneously."
  exit 1
fi

echo ""
echo "${BOLD}No errors found. Continuing...${NORMAL}"
echo ""

confirm "Are you ready to apply these settings? [y/n]"

####
#### File modification loop
####
for i in "${HOSTAPD[@]}"
do
  if [ -f ${i} ]; then
    echo "Working on $i..."
    wLog "Working on $i..."
	if [ $OPT_S != false ]; then
		wLog "Writing SSID $OPT_S to file $i"
    	echo "${MAGENTA}Setting ${YELLOW}SSID${MAGENTA} to ${YELLOW}$OPT_S ${MAGENTA}in $i...${WHITE}"
        if grep -q "^ssid=" ${HOSTAPD[$x]}; then
        sed -i "s/^ssid=.*/ssid=${OPT_S}/" ${i}
      else
        echo ${OPT_S} >> ${i}
      fi
    fi

    if [ $OPT_C != false ]; then
		wLog "Writing channel $OPT_C to file $i"
    	echo "${MAGENTA}Setting ${YELLOW}Channel${MAGENTA} to ${YELLOW}$OPT_C ${MAGENTA}in $i...${WHITE}"
        if grep -q "^channel=" ${i}; then
            sed -i "s/^channel=.*/channel=${OPT_C}/" ${i}
        else
            echo ${OPT_C} >> ${i}
        fi
    fi

    if [ $OPT_E != false ] || [ $OPT_P  != false ]; then
		wLog "Writing security and setting passphrase to $wifiPass to file $i"
        echo "${MAGENTA}Adding WPA encryption with passphrase: ${YELLOW}$wifiPass ${MAGENTA}to $i...${WHITE}"
        cleanhostapd $i
		writehostapd $i
    fi

    if [ $OPT_O != false ]; then
		wLog "Removing WiFi security in file $i"
        echo "${MAGENTA}Removing WPA encryption in $i...${WHITE}"
        cleanhostapd $i
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
APPLYSETTINGS

exit 0
