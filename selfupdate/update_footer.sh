cp -f gen_gdl90 /usr/bin/gen_gdl90
cp -f libdump978.so /usr/lib/libdump978.so
cp -f linux-mpu9150/libimu.so /usr/lib/libimu.so


#Startup script.
cp -f init.d-stratux /etc/init.d/stratux
chmod 755 /etc/init.d/stratux
ln -s /etc/init.d/stratux /etc/rc2.d/S01stratux
ln -s /etc/init.d/stratux /etc/rc6.d/K01stratux

cp -f dump1090 /usr/bin/
