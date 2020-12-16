#!/bin/bash


if [[ ! -d esp-idf ]]; then
    git clone -b v4.0 --recursive https://github.com/espressif/esp-idf.git
fi

source esp-idf/export.sh
cd esp32-ogn-tracker
cd utils && make read_log && make serial_dump && cd ..


function disable {
    opt=$1
    sed -i "s/^\s*#define\s*$opt\s/\/\/#define $opt /g" main/config.h
}
function enable {
    opt=$1
    sed -i "s/^\s*\/\/\s*#define\s*$opt\s/#define $opt /g" main/config.h

    grep $opt -q main/config.h || echo "#define $opt" >> main/config.h # add option if it doesn't exist yet
}

## Initial basic configuration
disable WITH_FollowMe
disable WITH_U8G2_OLED
disable WITH_U8G2_SH1106
disable WITH_U8G2_FLIP
disable WITH_GPS_ENABLE
disable WITH_GPS_PPS
disable WITH_GPS_MTK
disable WITH_LORAWAN
disable WITH_SD
disable WITH_SDLOG

# ?? WITH_FANET, WITH_LORAWAN
enable WITH_GPS_UBX
enable WITH_GPS_UBX_PASS
enable WITH_GPS_NMEA_PASS
enable WITH_PAW


# First build for old T-Beams
disable WITH_TBEAM_V10
disable WITH_AXP
enable WITH_TBEAM

make -B -j16
source bin-arch.sh
rm -r stratux
mkdir stratux
cd stratux
tar xzf ../esp32-ogn-tracker-bin.tgz && zip -r esp32-ogn-tracker-bin-07.zip *
mv esp32-ogn-tracker-bin-07.zip ../../
cd ..
rm -r stratux


# Second build for new T-Beams
disable WITH_TBEAM
enable WITH_TBEAM_V10
enable WITH_AXP


sed -i 's/^#define WITH_TBEAM/#define WITH_TBEAM_V10/g' main/config.h
make -B -j16
source bin-arch.sh
mkdir stratux
cd stratux
tar xzf ../esp32-ogn-tracker-bin.tgz && zip -r esp32-ogn-tracker-bin-10+.zip *
mv esp32-ogn-tracker-bin-10+.zip ../../
cd ..
rm -r stratux

# Clean up
git checkout .
rm -r esp32-ogn-tracker-bin.tgz utils/read_log utils/serial_dump build