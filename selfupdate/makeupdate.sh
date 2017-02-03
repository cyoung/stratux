#!/bin/bash

#apt-get install -y dh-make

stratuxVersion=`git describe --tags --abbrev=0`
stratuxBuild=`git log -n 1 --pretty=%H`

echo
echo
echo "Packaging ${stratuxVersion} (${stratuxBuild})."
echo
echo

cd ..
make
rm -rf work
mkdir -p work/bin
cp gen_gdl90 work/bin/
cp libdump978.so work/bin/
cp linux-mpu9150/libimu.so work/bin/
cp __lib__systemd__system__stratux.service work/bin/
cp __root__stratux-pre-start.sh work/bin/
cp dump1090/dump1090 work/bin/
cp -r web work/bin/
cp image/hostapd.conf work/bin/
cp image/hostapd-edimax.conf work/bin/
cp image/config.txt work/bin/
cp image/rtl-sdr-blacklist.conf work/bin/
cp image/bashrc.txt work/bin/
cp image/modules.txt work/bin/
cp image/stxAliases.txt work/bin/
cp image/hostapd_manager.sh work/bin/
cp image/sdr-tool.sh work/bin/
cp image/10-stratux.rules work/bin/
cp image/99-uavionix.rules work/bin/
cp image/motd work/bin/
cp image/fancontrol.py work/bin/
cp image/stratux-wifi.sh work/bin/

#TODO: librtlsdr.
cd work/
cat ../selfupdate/update_header.sh >update.sh

echo "stratuxVersion=${stratuxVersion}" >>update.sh
echo "stratuxBuild=${stratuxBuild}" >>update.sh


find bin/ -type d | sed -e 's/^bin\///' | grep -v '^$' | while read dn; do
	echo "mkdir -p $dn" >>update.sh
done
find bin/ -type f | while read fn; do
	echo -n "packaging $fn... "
	UPFN=`echo $fn | sed -e 's/^bin\///'`
	echo "cat >${UPFN}.b64 <<__EOF__" >>update.sh
	gzip -c $fn | base64 >>update.sh
	echo "__EOF__" >>update.sh
	echo "base64 -d ${UPFN}.b64 | gzip -d -c >${UPFN}" >>update.sh
	echo "rm -f ${UPFN}.b64" >>update.sh
	echo "done"
done
cat ../selfupdate/update_footer.sh >>update.sh

chmod +x update.sh

OUTF="update-stratux-${stratuxVersion}-${stratuxBuild:0:10}.sh"
mv update.sh $OUTF


echo
echo
echo "$OUTF ready."
echo
echo
