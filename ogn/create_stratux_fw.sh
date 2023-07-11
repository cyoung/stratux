#!/bin/bash
set -e

cd "$(dirname "$0")"

if [[ ! -d esp-idf ]]; then
    echo
    read -p "Install esp-idf? [y/n]" -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        git clone -b v4.4.4 --recursive https://github.com/espressif/esp-idf.git
        cd esp-idf && ./install.sh
        cd ..
    else
        echo "esp-idf required to build firmware"
        exit 1
    fi
fi

source esp-idf/export.sh
cd esp32-ogn-tracker
cd utils && make read_log && make serial_dump && cd ..


function disable {
    opt=$1
    sed -i "s~^\s*#define\s*$opt\$~// #define $opt~g" main/config.h
}
function enable {
    opt=$1
    sed -i "s~^\s*//\s*#define\s*$opt\$~#define $opt~g" main/config.h

    grep $opt -q main/config.h || echo "#define $opt" >> main/config.h # add option if it doesn't exist yet
}

git checkout main/config.h
# to simplify our regexes, remove all the comments..
sed -i "s~\s\s\s*//.*~~g" main/config.h

## Initial basic configuration
disable WITH_FollowMe
disable WITH_U8G2_OLED
disable WITH_U8G2_SH1106
disable WITH_U8G2_FLIP
disable WITH_GPS_ENABLE
disable WITH_GPS_MTK
disable WITH_SD
disable WITH_SDLOG
disable WITH_AP
disable WITH_HTTP


# ?? WITH_FANET, WITH_LORAWAN
enable WITH_GPS_UBX
enable WITH_GPS_UBX_PASS
enable WITH_GPS_NMEA_PASS
enable WITH_BME280
enable WITH_PAW
enable WITH_LORAWAN
enable WITH_ADSL
enable WITH_FANET

rm -rf stratux # cleanup of old build


# First build for old T-Beams
disable WITH_TBEAM_V10
disable WITH_AXP
disable WITH_GPS_PPS
enable WITH_TBEAM
make -B -j16 > /dev/null
source bin-arch.sh
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
enable WITH_GPS_PPS

make -B -j16 > /dev/null
source bin-arch.sh
mkdir stratux
cd stratux
tar xzf ../esp32-ogn-tracker-bin.tgz && zip -r esp32-ogn-tracker-bin-10+.zip *
mv esp32-ogn-tracker-bin-10+.zip ../../
cd ..
rm -r stratux


# Third build SX1262 variant
enable WITH_SX1262
disable WITH_LORAWAN
disable WITH_RFM95

make -B -j16 > /dev/null
source bin-arch.sh
mkdir stratux
cd stratux
tar xzf ../esp32-ogn-tracker-bin.tgz && zip -r esp32-ogn-tracker-bin-10+-sx1262.zip *
mv esp32-ogn-tracker-bin-10+-sx1262.zip ../../
cd ..
rm -r stratux

# Clean up
git checkout .
rm -r esp32-ogn-tracker-bin.* utils/read_log utils/serial_dump build


