#!/bin/bash

######################################################################
#                      STRATUX SDR MANAGER                           #
######################################################################

#Set Script Name variable
SCRIPT=`basename ${BASH_SOURCE[0]}`

# rtl_eeprom -d 0 -s <String>:<Freq>:<PPM>
#Initialize variables to default values.
SERVICE=stratux.service
WhichSDR=1090
FallBack=true
PPMValue=0

parm="*"
err="####"
att="+++"

#Set fonts for Help.
BOLD=$(tput bold)
STOT=$(tput smso)
DIM=$(tput dim)
UNDR=$(tput smul)
REV=$(tput rev)
RED=$(tput setaf 1)
GREEN=$(tput setaf 2)
YELLOW=$(tput setaf 3)
MAGENTA=$(tput setaf 5)
WHITE=$(tput setaf 7)
NORM=$(tput sgr0)
NORMAL=$(tput sgr0)

#This is the Title
function HEAD {
	clear
	echo "######################################################################"
	echo "#                     STRATUX SDR SERIAL TOOL                        #"
	echo "######################################################################"
	echo " "
}

function STOPSTRATUX {
	HEAD
	echo "Give me a few seconds to check if STRATUX is running..."
	# The service we want to check (according to systemctl)
	if [ "`systemctl is-active $SERVICE`" = "active" ] 
	then
	    echo "$SERVICE is currently running"
	    echo "Stopping..."
	    SDRs=`systemctl stop stratux.service`
	fi
	sleep 3
}

function STARTSTRATUX {
	HEAD
	echo "Give me a few seconds to get STRATUX running again..."
	SDRs=`systemctl start stratux.service`
	sleep 3
	if [ "`systemctl is-active $SERVICE`" = "active" ] 
	then
	    echo "$SERVICE is now running"
	else 
		echo "$SERVICE did not restart. Try 'reboot' to restart your RaspberryPI"
	fi
}

#Function to set the serial function
function SETSDRSERIAL {
	HEAD
    echo "#            Setting ${WhichSDR}mhz SDR Serial Data                  #"
	#Build this string
	# rtl_eeprom -d 0 -s <String>:<Freq>:<PPM>
	echo " SETTING SERIAL: "
	echo " rtl_eeprom -d 0 -s stx:${WhichSDR}:${PPMValue} "
    echo " "
    echo "${REV}Answer 'y' to the qustion: 'Write new configuration to device [y/n]?'${NORM}"
    echo " "
    SDRs=`rtl_eeprom -d 0 -s stx:${WhichSDR}:${PPMValue}`
    sleep 2
    echo " "
    echo "Do you have another SDR to program?"
    echo "     'Yes' will shutdown your STRATUX and allow you to swap SDRs."
    echo "     'No' will reboot your STRATUX and return your STRATUX to normal operation."
    echo "     'exit' will exit the script and return you to your shell prompt"
    choices=( 'Yes' 'No' 'exit' )
	# Present the choices.
	# The user chooses by entering the *number* before the desired choice.
	select choice in "${choices[@]}"; do

  		# If an invalid number was chosen, $choice will be empty.
  		# Report an error and prompt again.
		[[ -n $choice ]] || { echo "Invalid choice." >&2; continue; }	

  		case $choice in
    		'Yes')
            		echo "Shutting down..."
    	    		SDRs=`shutdown -h now`
    	    	;;
    		'No')
    	  		echo "Rebooting..."
                	SDRs=`reboot`
    	  	;;
    		exit)
			STARTSTRATUX
    	  		echo "Exiting. "
    	  	exit 0
  		esac
    break
    done
}


function SDRInfo {
	HEAD
    echo "#              Building ${WhichSDR}mhz SDR Serial                     #"
    echo " "
    echo "Do you have a PPM value to enter?"
    echo "If not, its ok... Just choose 'No'"
    choices=( 'Yes' 'No' 'exit' )
	# Present the choices.
	# The user chooses by entering the *number* before the desired choice.
	select choice in "${choices[@]}"; do

  		# If an invalid number was chosen, $choice will be empty.
  		# Report an error and prompt again.
		[[ -n $choice ]] || { echo "Invalid choice." >&2; continue; }	

  		case $choice in
    		'Yes')
    	    	echo "Please enter your PPM value for your ${WhichSDR}mhz SDR:"
				read PPMValue
    	    ;;
    		'No')
    	  		echo " "
    	  	;;
    		exit)
			STARTSTRATUX
    	  		echo "Exiting. "
    	  	exit 0
  		esac
    break
    done
	SETSDRSERIAL
}

function PICKFALLBACK {
	HEAD
    echo "#             Gathering ${WhichSDR}mhz SDR Serial                    #"
    echo " "
    echo "${RED}${BOLD}IMPORTANT INFORMATION: READ CAREFULLY${NORM}"
    echo "${BOLD}DO you want to set the 1090mhz SDR to Fall Back to 978mhz in the event of the 978mhx SDR failing inflight?${NORM}"
    echo "If no serials are set on any of the attached SDRs then STRATUX will assign 978mhz to the first SDR found and 1090mhz to the remaining SDR. This is a safety featre of STRATUX to always allow users to always have access to WEATHER and METAR data in the event of one SDR failing in flight. "
    echo " "
    echo "When a user assigns a frequency to an SDR, via setting serials, STRATUX will always assign that frequency. NO MATTER WHAT."
    echo "This could cause issues if an SDR fails in flight. If the 978mhz SDR fails in flight and the other SDR is assigned the 1090 serial this SDR will never be set to 978mhz and the user will not have access to WEATHER and METAR data" 
    echo " "
    echo "Choosing the Fall Back mode will allow the remaining SDR to be assigned to 978mhz while keeping the PPM value, allowing the user to continue to receive WEATHER and METAR data."
    echo "Fall Back mode is reccomended!"
    
    choices=( 'FallBack' '1090mhz' 'exit' )
	# Present the choices.
	# The user chooses by entering the *number* before the desired choice.
	select choice in "${choices[@]}"; do

  		# If an invalid number was chosen, $choice will be empty.
  		# Report an error and prompt again.
		[[ -n $choice ]] || { echo "Invalid choice." >&2; continue; }	

  		case $choice in
    		'FallBack')
            	WhichSDR=0
    	    ;;
    		'1090mhz')
				echo " "
    	  	;;
    		exit)
			STARTSTRATUX
    	  		echo "Exiting. "
    	  	exit 0
  		esac
    break
    done

}

function PICKFREQ {
	HEAD
    echo "#                Selecting Radio to set Serial                       #"
    echo " "
    echo "${BOLD}Which SDR are you setting up?${NORM}"
    echo "${DIM}If you have tuned antennas make sure you have the correct SDR and antenna combination hooked up at this time and remember which antenna connection is for which antenna.${NORM}"
    choices=( '868mhz' '978mhz' '1090mhz' 'exit' )
	# Present the choices.
	# The user chooses by entering the *number* before the desired choice.
	select choice in "${choices[@]}"; do

  		# If an invalid number was chosen, $choice will be empty.
  		# Report an error and prompt again.
		[[ -n $choice ]] || { echo "Invalid choice." >&2; continue; }	

  		case $choice in
			'868mhz')
				WhichSDR=868
				SDRInfo
			;;
    		'978mhz')
            	WhichSDR=978
    	    	SDRInfo
    	    ;;
    		'1090mhz')
                PICKFALLBACK
    	  		SDRInfo
    	  	;;
    		exit)
    	  		STARTSTRATUX
			echo "Exiting. "
    	  	exit 0
  		esac
    break
    done
}

function MAINMENU {
	HEAD
	echo "Loading SDR info..."
    	sleep 2
	HEAD
	echo "#                CONFIRM ONLY ONE SDR INSTALLED                      #"
	echo "----------------------------------------------------------------------"
	SDRs=`rtl_eeprom`
    echo "----------------------------------------------------------------------"
	echo " "
	echo "${BOLD}${RED}Read the lines above.${NORM}"
	echo "${BOLD}How many SDRs were found?${NORM}"

	# Define the choices to present to the user, which will be
	# presented line by line, prefixed by a sequential number
	# (E.g., '1) copy', ...)
	choices=( 'Only 1' '2 or more' 'exit' )
	# Present the choices.
	# The user chooses by entering the *number* before the desired choice.
	select choice in "${choices[@]}"; do
	
		# If an invalid number was chosen, $choice will be empty.
  		# Report an error and prompt again.
  		[[ -n $choice ]] || { echo "Invalid choice." >&2; continue; }

		case $choice in
    		'Only 1')
		      PICKFREQ
      		;;
		    '2 or more')
				echo "#####################################################################################"
				echo "#      ${RED}Too Many SDRs Plugged in. Unplug all SDRs except one and try again!!${NORM}         #"
				echo "#####################################################################################"
				STARTSTRATUX
				echo "Exiting. "
		      exit 0
		      ;;
		    exit)
		    	STARTSTRATUX
		    	echo "Exiting... "
		    	exit 0
		  esac
 	# Getting here means that a valid choice was made,
  	# so break out of the select statement and continue below,
  	# if desired.
  	# Note that without an explicit break (or exit) statement, 
  	# bash will continue to prompt.
  	break
	done
}

function START {
  echo "Help documentation for ${BOLD}${SCRIPT}.${NORM}"
  echo " "
  echo "This script will help you in setting your SDR serials. Please read carefully before continuing.  There are many options in settings the SDR serials. Programming the SDR serials does 2 things. "
  echo " "
  echo "${BOLD}First:${NORM}"
  echo "Setting the serials will tell your STRATUX which SDR is attached to which tuned antenna."
  echo " "
  echo "${BOLD}Second:${NORM}"
  echo "Setting the PPM value will enhance the reception of your SDR by correcting the Frequency Error in each SDR.  Each PPM value is unique to each SDR. For more info on this please refer to the Settings page in the WebUI and click on the Help in the top right."
  echo " "
  echo "Steps we will take:"
  echo "1) Make sure you have ${BOLD}${REV}ONLY ONE${NORM} SDR plugged in at a time. Plugging in one SDR at a time will ensure they are not mixed up."
  echo "2) Select which SDR we are setting the serial for."
  echo "3) Add a PPM value. If you do not know or do not want to set this value this will be set to 0. "
  echo "4) Write the serial to the SDR."
  echo " "
  echo "If you are ready to begin choose ${BOLD}Continue${NORM} to begin."
  echo "     Continuing will stop the STRATUX service to release the SDRs for setting the serials"
   
	choices=( 'Continue' 'Exit' )
	# Present the choices.
	# The user chooses by entering the *number* before the desired choice.
	select choice in "${choices[@]}"; do

  		# If an invalid number was chosen, $choice will be empty.
  		# Report an error and prompt again.
		[[ -n $choice ]] || { echo "Invalid choice." >&2; continue; }	

  		case $choice in
    		'Continue')
				STOPSTRATUX
    	    	MAINMENU
    	    ;;
    		exit)
			STARTSTRATUX
    	  		echo "Exiting. "
    	  	exit 0
  		esac
    break
    done
}

HEAD
START
