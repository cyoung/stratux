#!/bin/bash

# To run this, make sure that this is installed:
# sudo apt install --yes parted zip unzip zerofree
# If you want to build on x86 with aarch64 emulation, additionally install qemu-user-static qemu-system-arm
# Run this script as root.
# Run with argument "dev" to not clone the stratux repository from remote, but instead copy this current local checkout onto the image
set -x
BASE_IMAGE_URL="https://downloads.raspberrypi.com/raspios_lite_arm64/images/raspios_lite_arm64-2024-07-04/2024-07-04-raspios-bookworm-arm64-lite.img.xz"
ZIPNAME="$(basename $BASE_IMAGE_URL)"
IMGNAME="$(basename $ZIPNAME .xz)"
TMPDIR="$HOME/stratux-tmp"
# REMOTE_ORIGIN=$(git config --get remote.origin.url) # would be nicer, but doesn't work with ssh clone..
REMOTE_ORIGIN="https://github.com/b3nn0/stratux.git"

die() {
    echo $1
    exit 1
}

if [ "$#" -lt 2 ]; then
    echo "Usage: " $0 " dev|prod branch [us]"
    echo "if \"us\" is given, an image with US-like pre-configuration and without developer mode enabled will be created as well"
    exit 1
fi

# cd to script directory
cd "$(dirname "$0")"
SRCDIR="$(realpath $(pwd)/..)"
mkdir -p $TMPDIR
cd $TMPDIR

# Download/extract image
wget -c $BASE_IMAGE_URL || die "Download failed"
unxz -k $ZIPNAME || die "Extracting image failed"

# Check where in the image the root partition begins:
bootoffset=$(parted $IMGNAME unit B p | grep fat32 | awk -F ' ' '{print $2}')
bootoffset=${bootoffset::-1}
partoffset=$(parted $IMGNAME unit B p | grep ext4 | awk -F ' ' '{print $2}')
partoffset=${partoffset::-1}

# Original image partition is too small to hold our stuff.. resize it to 2.5gb
# Append one GB and truncate to size
truncate -s 3500M $IMGNAME || die "Image resize failed"
lo=$(losetup -f)
losetup $lo $IMGNAME
partprobe $lo
e2fsck -y -f ${lo}p2
parted --script ${lo} resizepart 2 100%
partprobe $lo || die "Partprobe failed failed"
resize2fs -p ${lo}p2 || die "FS resize failed"



# Mount image locally, clone our repo, install packages..
mkdir -p mnt
mount -t ext4 ${lo}p2 mnt/ || die "root-mount failed"
mount -t vfat ${lo}p1 mnt/boot/firmware || die "boot-mount failed"


cd mnt/root/
if [ "$1" == "dev" ]; then
    rsync -av --progress --exclude=ogn/esp-idf --exclude="**/*.mbtiles" --exclude=esp32-ogn-tracker $SRCDIR ./
    cd stratux && git checkout $2 && cd ..
else
    git clone --recursive -b $2 $REMOTE_ORIGIN
fi
cd ../../

# Use latest qemu-aarch64-static version, since aarch64 doesn't seem to be that stable yet..
if [ "$(arch)" != "aarch64" ]; then
    wget -P mnt/usr/bin/ https://github.com/multiarch/qemu-user-static/releases/download/v7.2.0-1/qemu-aarch64-static
    chmod +x mnt/usr/bin/qemu-aarch64-static
    unshare -mpfu chroot mnt qemu-aarch64-static -cpu cortex-a72 /bin/bash -l -c /root/stratux/image/mk_europe_edition_device_setup64.sh
else
    unshare -mpfu chroot mnt /bin/bash -l -c /root/stratux/image/mk_europe_edition_device_setup64.sh
fi
mkdir -p out

# Move the selfupdate file out of there..
mv mnt/root/update-*.sh out

umount mnt/boot/firmware
umount mnt

# Shrink the image to minimum size.. it's still larger than it really needs to be, but whatever
minsize=$(resize2fs -P ${lo}p2 | rev | cut -d' ' -f 1 | rev)
minsizeBytes=$(($minsize * 4096))
e2fsck -f ${lo}p2
resize2fs -p ${lo}p2 $minsize

zerofree ${lo}p2 # for smaller zip

bytesEnd=$(($partoffset + $minsizeBytes))

losetup -d ${lo}

# parted --script $IMGNAME resizepart 2 ${bytesEnd}B Yes doesn't seem tow rok any more... echo yes | parted .. neither. So we re-create partition with proper size
parted --script $IMGNAME rm 2
parted --script $IMGNAME unit B mkpart primary ext4 ${partoffset}B ${bytesEnd}B
truncate -s $(($bytesEnd + 4096)) $IMGNAME


cd $SRCDIR
outname="stratux-$(git describe --tags --abbrev=0)-$(git log -n 1 --pretty=%H | cut -c 1-8).img"
outname_us="stratux-$(git describe --tags --abbrev=0)-$(git log -n 1 --pretty=%H | cut -c 1-8)-us.img"
cd $TMPDIR

# Rename and zip EU version
mv $IMGNAME $outname
zip out/$outname.zip $outname


# Now create US default config into the image and create the eu-us version..
if [ "$3" == "us" ]; then
    mount -t vfat -o offset=$bootoffset $outname mnt/ || die "boot-mount failed"
    echo '{"UAT_Enabled": true,"OGN_Enabled": false,"DeveloperMode": false}' > mnt/stratux.conf
    umount mnt
    mv $outname $outname_us
    zip out/$outname_us.zip $outname_us
fi


echo "Final images has been placed into $TMPDIR/out. Please install and test the image."
