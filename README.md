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
* Added a special "Skydemon wonky GDL90 parser" workaround to reduce Skydemons constant detection of very short disconnects (see below)

## Building the Europe Edition
Building the european Edition is practically the same as the official Stratux. More information can be found here:
http://stratux.me/
You can also buy a prebuilt unit.
The only difference: It is highly recommended to purchase a 868 Mhz antenna for FLARM reception. The standard 978 Mhz antenna can receive some FLARM targets, but the range will be very limited.
Additionally, you will need a PC with an SD Card reader.
Download the latest image here: https://github.com/b3nn0/stratux/releases
and use an arbitrary tool to burn the image to your Micro SD Card (e.g. "Etcher", see https://www.raspberrypi.org/documentation/installation/installing-images/).



## Notes to SkyDemon Android Users
SkyDemon is probably the most popular EFB in Europe, and we are trying hard to make Stratux work as good as possible in SkyDemon, which is not always easy. Most notably, with original Stratux on a RaspberryPI 2b, you can often oberve Disconnects, which will show as many red dots in your track log.

Thorough analysis has shown that this is caused by a mix of
- RaspberryPI's brcmfmac wifi driver and its behaviour when UDP package delivery is slow
- Androids handling of UDP packets under load - namely the fact that it will delay them
- A wonky GDL90 implementation in SkyDemon (which is not very error tolerant, even though the UDP RFC explicitly says that applications should expect errors and work around them).

If you will suffer from these problems depends on many factors, but it is certainly possible.
There are two known ways to get around these errors:
### 1) Use the workaround
As of version Stratux 1.5b2eu004, the web interface has a settings switch labeled "SkyDemon Android disconnect bug workaround". Enabling this will cause Stratux to send position reports to the EFB every 150ms instead of every second.
Experiments show that SkyDemon handles this well and will show disconnects only very rarely, if ever.
Note that this is an ugly hack and does not conform the GDL90 specification, but it seems to do the job for SkyDemon

### 2) Different WiFi stick for Stratux
If you want a better solution, you might want to consider investing in a different WiFi chip for your Stratux.
The EDIMAX EW-7811UN has shown to work decently, is cheap, and small enough to fit nicely in any case.
Plugging this in and rebooting your Stratux should automatically use this device for the hotspot, instead of the builtin wifi.
The driver for this chip seems to have a different retransmit behaviour, which will not confuse SkyDemon as much, hence the apparent disconnects are gone.
Note: If you have the internal GPS with the large antenna, you might have to cut off one edge of your GPS PCB to make it fit.


