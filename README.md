# Stratux - European edition
**Users from the US, see here: https://github.com/b3nn0/stratux/wiki/US-configuration**

This is a fork of the original Stratux version, incorperating many contributions by the community to create a
nice, full featured Stratux-OGN image that works well for europe.
![Data flow diagram](https://user-images.githubusercontent.com/60190549/94661904-f1201c80-0307-11eb-9d8d-3af2020583a8.png)
(see https://github.com/b3nn0/stratux/wiki/Stratux-EU-Structure)

## Disclaimer
This repository offers code and binaries that can help you to build your own traffic awareness device. We do not take any responsibility for what you do with this code. When you build a device, you are responsible for what it does. There is no warrenty of any kind provided with the information, code and binaries you can find here. You are solely responsible for the device you build.

## Main differences to original Stratux
* Original Stratux: https://github.com/cyoung/stratux
* Added OGN receiver functionality to receive several protocols on the 868Mhz frequency band, comparable to what the OpenGliderNetwork does
* Several improvements and bug fixes to GPS handling and chip configuration (by [VirusPilot](https://github.com/VirusPilot)
* Support for transmitting OGN via a TTGO T-Beam
* If no pressure sensor is present, Stratux EU will try to estimate your pressure altitude with atmospheric information received from other aircraft. We still recommend using some kind of barometric sensor (e.g. Stratux AHRS module). More information can be found [here](https://github.com/b3nn0/stratux/wiki/Altitudes-in-Stratux-EU)
* By default, OGN and DeveloperMode is enabled, UAT is disabled
* Several new features: Integrated traffic radar (by [TomBric](https://github.com/TomBric), online and offline traffic map and much more
* Updated RaspberryPi OS to Buster 64 bit, to support newer RaspberryPis as well
* Added a special "Skydemon wonky GDL90 parser" workaround to reduce Skydemons constant detection of very short disconnects (see below)
* Support for NMEA output (including PFLAA/PFLAU traffic messages) via TCP Port 2000 and [serial](https://github.com/b3nn0/stratux/wiki/Stratux-Serial-output-for-EFIS's-that-support-GDL90-or-Flarm-NMEA-over-serial)
* Estimation of Mode C/S target distance by signal strength, transmission of bearingless targets via NMEA and GDL90
* Support for changing the Stratux's IP address
* Possibility to enter multiple ownship transponder HEX codes, Stratux will automatically decide which of these are actually you. This is useful if you have multiple aircraft that you regularly fly with (e.g. add all club aircraft)
* X-Plane 11 compatible output for EFBs that support simulator input (experimental, unsupported. Might make it possible to connect Garmin Pilot). Based on original work by 0x74-0x62
* Support for WiFi Direct connection to make it possible to let Android have mobile data connection while connected to the Stratux
* Many more small features, bug fixes and tweaks all over the place

## Building the Europe Edition
Due to the modular nature of Stratux, there are many possibilities how you can build it to your needs.
You can find three popular variations in the form of complete build guides [here](https://github.com/b3nn0/stratux/wiki/Building-Stratux-Europe-Edition).
It also shows how you can modify your pre-built Stratux US version to run the EU version.

If you want to customize beyond that, or have different needs, you can find a full list of supported hardware/attachments [here](https://github.com/b3nn0/stratux/wiki/Supported-Hardware).



## Notes to SkyDemon Users
SkyDemon is probably the most popular EFB in Europe, and we are trying hard to make Stratux work as good as possible in SkyDemon, which is not always easy. Most notably, with original Stratux on a RaspberryPI 2b, you can often observe disconnects, which will show as many red dots in your track log.

Thorough analysis has shown that this is caused by a mix of
- RaspberryPI's brcmfmac wifi driver and its behaviour when UDP package delivery is slow
- Androids/iOSs handling of UDP packets under load
- A wonky GDL90 implementation in SkyDemon (which is not very error tolerant, even though the UDP RFC explicitly says that applications should expect errors and work around them).

If you will suffer from these problems depends on many factors, but it is certainly possible.
The real solution would be, that SkyDemon behaves more error tolerant, but they seem to be resiliant to do so.
As of version 1.5b2-eu004, the web interface has a settings switch labeled "SkyDemon Android disconnect bug workaround". Enabling this will cause Stratux to send position reports to the EFB every 150ms instead of every second.
Experiments show that SkyDemon handles this relatively well and will show disconnects much rarer.
Note that this is an ugly hack and does not conform the GDL90 specification, but it seems to do the job for SkyDemon.


