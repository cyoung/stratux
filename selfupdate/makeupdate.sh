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
cp init.d-stratux work/bin/
cp dump1090/dump1090 work/bin/
#TODO: librtlsdr.
cd work/
cat ../selfupdate/update_header.sh >update.sh
find bin/ -type f | while read fn; do
	echo -n "packaging $fn... "
	UPFN=`echo $fn | cut -d/ -f2`
	echo "cat >${UPFN}.b64 <<__EOF__" >>update.sh
	gzip -c $fn | base64 >>update.sh
	echo "__EOF__" >>update.sh
	echo "base64 -d ${UPFN}.b64 | gzip -d -c >${UPFN}" >>update.sh
	echo "rm -f ${UPFN}.b64" >>update.sh
	echo "done"
done
cat ../selfupdate/update_footer.sh >>update.sh

chmod +x update.sh

OUTF="update-${stratuxVersion}.sh"
mv update.sh $OUTF


echo
echo
echo "$OUTF ready."
echo
echo
