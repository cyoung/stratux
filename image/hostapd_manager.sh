#!/bin/bash
<<<<<<< HEAD
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
    echo "usage: $0 quiet ssid channel passphrase"
    echo "  quiet           will the script output messages 1 = yes"
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
if [ ! -z "$3" ]; then
	if [ -z `echo $4 | tr -d "[:print:]"` ] && [ ${#4} -ge 8 ]  && [ ${#4} -le 63 ]; then
  		PASS=wpa_passphrase=$3
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


    if [ ! -z "$3" ]; then
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


    if [ ! -z "$3" ]; then
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
=======

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

#defaultPass="Squawk1200"

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
  echo -e "${REV}Basic usage:${NORM} ${BOLD}$SCRIPT -s ssid -c chan -e pass ${NORM}"\\n
  echo "Command line switches are optional. The following switches are recognized."
  echo "${REV}-s${NORM}  --Sets the SSID to ${BOLD}ssid${NORM}. \"-s stratux\""
  echo "${REV}-c${NORM}  --Sets the channel to ${BOLD}chan${NORM}. \"-c 1\""
 # echo "${REV}-e${NORM}  --Turns on encryption with passphrase ${BOLD}pass${NORM}. 8-63 Printable Characters(ascii 32-126). Cannot be used with -o. \"-e password!\""
 # echo "${REV}-o${NORM}  --Turns off encryption and sets network to open. Cannot be used with -e."
 # echo "${REV}-q${NORM}  --Run silently."
  echo -e "${REV}-h${NORM}  --Displays this help message. No further functions are performed."\\n
  echo -e "Example: ${BOLD}$SCRIPT -s Stratux-N3558D -c 5${NORM}"\\n
  exit 1
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
#Notice there is no ":" after "oqh". The leading ":" suppresses error messages from
#getopts. This is required to get my unrecognized option code to work.
#options=':s:c:e:oqh'
options=':s:c:h'
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
#    e)  #set option "e"
#      if [[ -z "${OPTARG}" || "${OPTARG}" == *[[:space:]]* || "${OPTARG}" == -* ]]; then
#          echo "${BOLD}${RED}$err Encryption option(-e) used without passphrase!${WHITE}${NORMAL}"
#          OPT_E=$defaultPass
#      else
#        OPT_E=$OPTARG
#        echo "$parm Encryption option -e used:"
#        if [ -z `echo $OPT_E | tr -d "[:print:]"` ] && [ ${#OPT_E} -ge 8 ]  && [ ${#OPT_E} -le 63 ]; then
#          echo "${GREEN}    Passphrase will now be ${BOLD}${UNDR}$OPT_E${NORMAL}.${WHITE}${NORMAL}"
#		else
#    	  echo  "${BOLD}${RED}$err Invalid PASSWORD: 8 - 63 printable characters, exiting...${WHITE}${NORMAL}"
#          exit
#        fi   
#      fi
#      ;;
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
    h)  #show help
      HELP
      ;;
   \?) # invalid option
     echo "${BOLD}${RED}$err Invalid option -$OPTARG ${WHITE}${NORMAL}" >&2
     exit 1
     ;;
   :) # Missing Arg 
     echo "${BOLD}${RED}$err Missing option for argument -$OPTARG ${WHITE}${NORMAL}" >&2
     exit 1
     ;;
   *) # Invalid
     echo "${BOLD}${RED}$err Unimplemented option -$OPTARG ${WHITE}${NORMAL}" >&2
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

if [ $OPT_O = true ] && [ $OPT_E != false ]; then
  echo "${BOLD}${RED}$err Option -e and -o cannot be used simultaneously ${WHITE}${NORMAL}"
  exit 1
fi

echo ""
echo "${BOLD}No errors found. Continuning...${NORMAL}"
echo ""

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
    
#    if [ $OPT_E != false ]; then
#    	echo "${MAGENTA}Adding WPA encryption with passphrase: ${YELLOW}$OPT_E ${MAGENTA}to $i...${WHITE}"
#        if grep -q "^#auth_algs=" ${i}; then
#        	#echo "uncomenting wpa"
#            sed -i "s/^#auth_algs=.*/auth_algs=1/" ${i}
#            sed -i "s/^#wpa=.*/wpa=3/" ${i}
#            sed -i "s/^#wpa_passphrase=.*/wpa_passphrase=$OPT_E/" ${i}
#            sed -i "s/^#wpa_key_mgmt=.*/wpa_key_mgmt=WPA-PSK/" ${i}
#            sed -i "s/^#wpa_pairwise=.*/wpa_pairwise=TKIP/" ${i}
#            sed -i "s/^#rsn_pairwise=.*/rsn_pairwise=CCMP/" ${i}
#       elif grep -q "^auth_algs=" ${i}; then
#        	#echo "rewriting existing wpa"
#            sed -i "s/^auth_algs=.*/auth_algs=1/" ${i}
#            sed -i "s/^wpa=.*/wpa=3/" ${i}
#            sed -i "s/^wpa_passphrase=.*/wpa_passphrase=$OPT_E/" ${i}
#            sed -i "s/^wpa_key_mgmt=.*/wpa_key_mgmt=WPA-PSK/" ${i}
#            sed -i "s/^wpa_pairwise=.*/wpa_pairwise=TKIP/" ${i}
#            sed -i "s/^rsn_pairwise=.*/rsn_pairwise=CCMP/" ${i}
#       else
#       		#echo "adding wpa"
#       		echo "" >> ${i}
#            echo "auth_algs=1" >> ${i}
#			echo "wpa=3" >> ${i}
#			echo "wpa_passphrase=$OPT_E" >> ${i}
#			echo "wpa_key_mgmt=WPA-PSK" >> ${i}
#            echo "wpa_pairwise=TKIP" >> ${i}
#			echo "rsn_pairwise=CCMP" >> ${i}
#        fi
#    fi
#    if [ $OPT_O != false ]; then
#       	echo "${MAGENTA}Removing WPA encryption in $i...${WHITE}"
#        if grep -q "^auth_algs=" ${i}; then
#        	#echo "comenting out wpa"
#            sed -i "s/^auth_algs=.*/#auth_algs=1/" ${i}
#            sed -i "s/^wpa=.*/#wpa=3/" ${i}
#            sed -i "s/^wpa_passphrase=.*/#wpa_passphrase=$defaultPass/" ${i}
#            sed -i "s/^wpa_key_mgmt=.*/#wpa_key_mgmt=WPA-PSK/" ${i}
#            sed -i "s/^wpa_pairwise=.*/#wpa_pairwise=TKIP/" ${i}
#            sed -i "s/^rsn_pairwise=.*/#rsn_pairwise=CCMP/" ${i}
#        elif grep -q "^#auth_algs=" ${i}; then
#        	#echo "rewriting comentied out wpa"
#            sed -i "s/^#auth_algs=.*/#auth_algs=1/" ${i}
#            sed -i "s/^#wpa=.*/#wpa=3/" ${i}
#            sed -i "s/^#wpa_passphrase=.*/#wpa_passphrase=$defaultPass/" ${i}
#            sed -i "s/^#wpa_key_mgmt=.*/#wpa_key_mgmt=WPA-PSK/" ${i}
#            sed -i "s/^#wpa_pairwise=.*/#wpa_pairwise=TKIP/" ${i}
#            sed -i "s/^#rsn_pairwise=.*/#rsn_pairwise=CCMP/" ${i}
#        else
#        	#echo "adding commented out WPA"
#        	echo "" >> ${i}
#        	echo "#auth_algs=1" >> ${i}
#			echo "#wpa=3" >> ${i}
#			echo "#wpa_passphrase=$defaultPass" >> ${i}
#			echo "#wpa_key_mgmt=WPA-PSK" >> ${i}
#            echo "#wpa_pairwise=TKIP" >> ${i}
#			echo "#rsn_pairwise=CCMP" >> ${i}
#        fi
#    fi

	echo "${GREEN}Modified ${i}...done${WHITE}"
    echo ""
  else
   	echo "${MAGENTA}No ${i} file found...${WHITE}${NORMAL}"
    echo ""
  fi
done
echo "${RED}${BOLD} $att At this time the script will restart your WiFi services.${WHITE}${NORMAL}"
echo "If you are connected to Stratux through the ${BOLD}192.168.10.1${NORMAL} interface then you will be disconnected"
echo "Please wait 1 min and look for the new SSID on your wireless device."
sleep 3
echo "${YELLOW}$att Restarting Stratux WiFi Services... $att ${WHITE}"
echo "Killing hostapd..."
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

### End main loop ###

exit 0
>>>>>>> 47271df7647b4efa49d86409d98897400559b018
