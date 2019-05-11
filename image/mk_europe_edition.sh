#!/bin/bash

# To run this, make sure that this is installed:
# sudo apt install --yes qemu-user-static gparted qemu-system-arm
# Run this script as root.
# Run with argument "dev" to not clone the stratux repository from remote, but instead copy this current local checkout onto the image

BASE_IMAGE_URL="https://github.com/cyoung/stratux/releases/download/v1.5b1/stratux-v1.5b1-3d168d0c6c.img.zip"
IMGNAME="stratux-v1.5b1-3d168d0c6c.img"
TMPDIR="/tmp/stratux-tmp"


# cd to script directory
cd "$(dirname "$0")"
SRCDIR="$(realpath $(pwd)/..)"
mkdir -p $TMPDIR
cd $TMPDIR

# Download/extract image
wget -c $BASE_IMAGE_URL
unzip stratux-*.img.zip

# Check where in the image the root partition begins:
sector=$(fdisk -l $IMGNAME | grep Linux | awk -F ' ' '{print $2}')
partoffset=$(( 512*sector ))
bootoffset=$(fdisk -l $IMGNAME | grep W95 | awk -F ' ' '{print $2}')
bootoffset=$(( 512*bootoffset ))
sizelimit=$(fdisk -l $IMGNAME | grep W95 | awk -F ' ' '{print $4}')
sizelimit=$(( 512*sizelimit ))

# Original image partition is too small to hold our stuff.. resize it to 2.5gb
# Append one GB and truncate to size
#truncate -s 2600M $IMGNAME
qemu-img resize $IMGNAME 2500M
losetup -f
losetup /dev/loop0 $IMGNAME
partprobe /dev/loop0
e2fsck -f /dev/loop0p2
fdisk /dev/loop0 <<EOF
p
d
2
n
p
2
$sector

p
w
EOF
partprobe /dev/loop0
resize2fs -p /dev/loop0p2
losetup -d /dev/loop0




# Mount image locally, clone our repo, install packages..
mkdir -p mnt
mount -t ext4 -o offset=$partoffset $IMGNAME mnt/
mount -t vfat -o offset=$bootoffset,sizelimit=$sizelimit $IMGNAME mnt/boot
cp $(which qemu-arm-static) mnt/usr/bin

cd mnt/root
wget https://dl.google.com/go/go1.12.4.linux-armv6l.tar.gz
tar xzf go1.12.4.linux-armv6l.tar.gz
rm go1.12.4.linux-armv6l.tar.gz

if [ "$1" == "dev" ]; then
    cp -r $SRCDIR .
else
    git clone --recursive https://github.com/b3nn0/stratux.git
fi
cd ../..

# Now download a specific kernel to run raspbian images in qemu and boot it..
chroot mnt qemu-arm-static /bin/bash -c /root/stratux/image/mk_europe_edition_device_setup.sh
umount mnt/boot
umount mnt

mkdir -p $SRCDIR/image/out
mv $IMGNAME $SRCDIR/image/out/
cd $SRCDIR/image/out
outname="stratux-$(git describe --tags --abbrev=0)-$(git log -n 1 --pretty=%H | cut -c 1-8).img"
mv $IMGNAME $outname
zip $outname.zip $outname


echo "Final image has been placed into $SRCDIR/image/out. Please install and test the image."