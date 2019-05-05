#!/bin/bash

# To run this, make sure that qemu-user-static and gparted is installed.
# Run this script as root.
# Run with argument "dev" to not clone the stratux repository from remote, but instead copy this current local checkout onto the image

BASE_IMAGE_URL="https://github.com/cyoung/stratux/releases/download/v1.5b1/stratux-v1.5b1-3d168d0c6c.img.zip"
TMPDIR="/tmp/stratux-tmp"


# cd to script directory
cd "$(dirname "$0")"
SRCDIR="$(realpath $(pwd)/..)"
mkdir -p $TMPDIR
cd $TMPDIR

# Download/extract image
wget -c $BASE_IMAGE_URL
unzip stratux-*.img.zip 

# Original image partition is too small to hold our stuff.. resize it to 2.5gb
# Append one GB and truncate to size
#truncate -s 2600M stratux-*.img
qemu-img resize stratux-*.img 2500M
losetup -f
losetup /dev/loop0 stratux-*.img
partprobe /dev/loop0
e2fsck -f /dev/loop0p2
gparted /dev/loop0
# Any nice way to do this scripted? Resize partition, then FS via:
#resize2fs -p /dev/loop0p2 2430M
losetup -d /dev/loop0

# Check where in the image the root partition begins:
partoffset=$(fdisk -l stratux-*.img | tail -n1 | awk -F ' ' '{print $2}')
partoffset=$(( 512*partoffset ))
# Mount image locally, clone our repo, install packages..
mount -t ext4 -o offset=$partoffset stratux-*.img mnt/
cp $(which qemu-arm-static) mnt/usr/bin

cd mnt/root
wget https://dl.google.com/go/go1.10.4.linux-armv6l.tar.gz
tar xzf go1.10.4.linux-armv6l.tar.gz

if [ "$1" == "dev" ]; then
    git clone https://github.com/b3nn0/stratux.git
else
    cp -r $SRCDIR .
fi
cd ../..

chroot mnt qemu-arm-static /bin/bash -c /root/stratux/image/mk_europe_edition_device_setup.sh
