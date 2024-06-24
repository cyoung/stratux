#!/bin/bash
# Requires having platformio installed in your $PATH!

set -e

if ! command -v platformio; then
    echo "Please install platformio to your PATH first"
    exit 1
fi

cd "$(dirname "$0")"
STRATUX_OGN_DIR="$(pwd)"

curl https://raw.githubusercontent.com/pjalocha/esp32-ogn-tracker/master/esptool.py > esptool.py

cd ogn-tracker


function append_defaultconfig {
    echo \
'   -DTTGO_TBEAM
    -DWITH_ESP32
    -DWITH_OGN
    -DWITH_ADSL
    -DWITH_FANET
    -DWITH_PAW
    -DWITH_CONFIG     ; allow to change parameters via serial console
    -DWITH_GPS_NMEA_PASS
    -DWITH_OLED
    -DWITH_GPS_PPS    ; use the PPS of the GPS (not critical but gets betterr timing)
    ;-DWITH_GPS_CONFIG ; GPS can be adjusted for serial baud rate and navigation model
    -DWITH_BME280     ; recognizes automatically BMP280 or BME280
    ;-DWITH_BT_SPP     ; BT Standard Serial Port => connection to XCsoar, takes big resources in Flash and RAM
    -DWITH_LOOKOUT
    -DWITH_PFLAA
    ;-DWITH_AP         ; enable these 3 once AP_BUTTON is properly implemented
    ;-DWITH_AP_BUTTON
    ;-DWITH_HTTP
    -DRADIOLIB_GODMODE' >> platformio.ini
}

function package {
    fname=$1
    cd .pio/build/$fname
    cp $STRATUX_OGN_DIR/esptool.py .
    zip $fname.zip bootloader.bin firmware.bin partitions.bin esptool.py
    mv $fname.zip $STRATUX_OGN_DIR/
    cd -
}


function create_fw {
    name=$1
    board=$2
    flags=$3

    git checkout platformio.ini
    echo "
[env:$name]
board = $board
build_flags = $flags" >> platformio.ini
    append_defaultconfig
    platformio run -e $name
    package $name

    git checkout platformio.ini
}


# Build SX1276-tbeam-07
create_fw ogn-tracker-bin-tbeam07-sx1276 ttgo-lora32-v1 '
    -DWITH_TBEAM07
    -DWITH_SX1276
    -DWITH_GPS_UBX
    -DWITH_GPS_UBX_PASS'

# Build SX1276-tbeam-10
create_fw ogn-tracker-bin-tbeam10-sx1276 ttgo-lora32-v1 '
    -DWITH_TBEAM10
    -DWITH_SX1276
    -DWITH_GPS_UBX
    -DWITH_GPS_UBX_PASS'


# Build SX1276-tbeam-12
create_fw ogn-tracker-bin-tbeam12-sx1276 ttgo-lora32-v1 '
    -DWITH_TBEAM20
    -DWITH_SX1276
    -DWITH_GPS_UBX
    -DWITH_GPS_UBX_PASS'

# Build SX1262-tbeam-10
create_fw ogn-tracker-bin-tbeam10-sx1262 ttgo-lora32-v1 '
    -DWITH_TBEAM10
    -DWITH_SX1262
    -DWITH_GPS_UBX
    -DWITH_GPS_UBX_PASS'

# Build SX1262-tbeam-s3-mtk
create_fw ogn-tracker-bin-tbeams3-sx1262-mtk esp32-s3-devkitc-1 '
    -DWITH_TBEAMS3
    -DWITH_SX1262
    -DWITH_GPS_MTK
    -DWITH_GPS_ENABLE'

# Build SX1262-tbeam-s3-ubx
create_fw ogn-tracker-bin-tbeams3-sx1262-ubx esp32-s3-devkitc-1 '
    -DWITH_TBEAMS3
    -DWITH_SX1262
    -DWITH_GPS_UBX
    -DWITH_GPS_UBX_PASS'

exit 0

### TODO: This was our legacy build for old esp32-ogn-tracker. Keeping this here to show which options we used in the past

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



enable WITH_WIFI
enable WITH_AP
enable WITH_AP_BUTTON
enable WITH_HTTP

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
#enable WITH_AXP
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


