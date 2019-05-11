# Stratux - European edition
This is a fork of the original Stratux version, incorperating many contributions by the community to create a
nice, full featured Stratux-Flarm image that works well for europe.
## Difference to original Stratux
* Original Stratux: https://github.com/cyoung/stratux
* Merged devel/flarm_receiver from PepperJo, which enables flarm reception based on the OpenGliderNetwork decoding stack (https://github.com/PepperJo/stratux)
* Merged VirusPilot's fixes and improvements for U-Blox 8 devices and Galileo/Glonass reception (https://github.com/VirusPilot/stratux)
* Changed DHCP Settings to not set a DNS server - this fixes the hangs that can be observed with current SkyDemon versions when not having an internet connection
* If no pressure sensor is present, report GPS Altitude as pressure altitude to make SkyDemon happy (NOT RECOMENDED!)
* By default, FLARM and DeveloperMode is enabled, UAT is disabled
* Merged Stratux Web-Radar for web-based traffic display by TomBric (https://github.com/TomBric/Radar-Stratux)
* Upgraded the RaspberryPi Debian system to the latest debian packages
* Hide Weather/Towers page if UAT is disabled
* Added a simple Flarm Status page, loading the ogn-rf and ogn-decode web pages as iFrames
