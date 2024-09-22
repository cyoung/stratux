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
build_flags = " >> platformio.ini
    append_defaultconfig
    echo "$flags" >> platformio.ini
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
    -DWITH_AXP
    -DWITH_GPS_UBX
    -DWITH_GPS_UBX_PASS'

# Build SX1262-tbeam-10
create_fw ogn-tracker-bin-tbeam10-sx1262 ttgo-lora32-v1 '
    -DWITH_TBEAM10
    -DWITH_SX1262
    -DWITH_AXP
    -DWITH_GPS_UBX
    -DWITH_GPS_UBX_PASS'


# Build SX1276-tbeam-12
create_fw ogn-tracker-bin-tbeam12-sx1276 ttgo-lora32-v1 '
    -DWITH_TBEAM20
    -DWITH_SX1276
    -DWITH_XPOWERS
    -DWITH_GPS_UBX
    -DWITH_GPS_UBX_PASS'


# Build SX1262-tbeam-s3-mtk
create_fw ogn-tracker-bin-tbeams3-sx1262-mtk esp32-s3-devkitc-1 '
    -DWITH_TBEAMS3
    -DWITH_SX1262
    -DWITH_XPOWERS
    -DWITH_GPS_MTK
    -DWITH_GPS_ENABLE
    -DARDUINO_USB_MODE=1        
    -DARDUINO_USB_CDC_ON_BOOT=1
board_build.mcu = esp32s3'

# Build SX1262-tbeam-s3-ubx
create_fw ogn-tracker-bin-tbeams3-sx1262-ubx esp32-s3-devkitc-1 '
    -DWITH_TBEAMS3
    -DWITH_SX1262
    -DWITH_XPOWERS
    -DWITH_GPS_UBX
    -DWITH_GPS_UBX_PASS
    -DARDUINO_USB_MODE=1        
    -DARDUINO_USB_CDC_ON_BOOT=1
board_build.mcu = esp32s3'

