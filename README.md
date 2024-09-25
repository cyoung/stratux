# Stratux - European edition [![](https://dcbadge.limes.pink/api/server/D9NQ6xe4nF)](https://discord.gg/D9NQ6xe4nF)

**Users from the US, see here: https://github.com/b3nn0/stratux/wiki/US-configuration**

This is a fork of the original cyoung/Stratux version, incorperating many contributions by the community to create a
nice, full featured Stratux-OGN image that works well for europe, the US, and the rest of the world.
![Data flow diagram](https://user-images.githubusercontent.com/60190549/94661904-f1201c80-0307-11eb-9d8d-3af2020583a8.png)
(see https://github.com/b3nn0/stratux/wiki/Stratux-EU-Structure)

## Disclaimer
This repository offers code and binaries that can help you to build your own traffic awareness device. We do not take any responsibility for what you do with this code. When you build a device, you are responsible for what it does. There is no warrenty of any kind provided with the information, code and binaries you can find here. You are solely responsible for the device you build.

## Main differences to original Stratux
* Original Stratux: https://github.com/cyoung/stratux
* Added OGN receiver functionality to receive several protocols on the 868Mhz frequency band, comparable to what the OpenGliderNetwork does
* Several improvements and bug fixes to GPS handling and chip configuration (by [VirusPilot](https://github.com/VirusPilot)
* Support for transmitting OGN via a TTGO T-Beam
* More robust sensor handling
* Traffic Radar and Map
* Support for traffic output via Bluetooth LE
* Estimation of Mode-S target distance
* Support for NMEA output (including PFLAA/PFLAU traffic messages) via TCP Port 2000 and [serial](https://github.com/b3nn0/stratux/wiki/Stratux-Serial-output-for-EFIS's-that-support-GDL90-or-Flarm-NMEA-over-serial)
* And a _lot_ more bug fixes and smaller features all over the place

## Building the Europe Edition
Due to the modular nature of Stratux, there are many possibilities how you can build it to your needs.
You can find three popular variations in the form of complete build guides [here](https://github.com/b3nn0/stratux/wiki/Building-Stratux-Europe-Edition).
It also shows how you can modify your pre-built Stratux US version to run the EU version.

If you want to customize beyond that, or have different needs, you can find a full list of supported hardware/attachments [here](https://github.com/b3nn0/stratux/wiki/Supported-Hardware).

