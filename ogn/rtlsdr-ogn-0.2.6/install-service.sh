echo "Install rtlsdr-ogn to run OGN receiver as a service"
sudo apt-get -y install procserv nano
sudo cp -v rtlsdr-ogn /etc/init.d/rtlsdr-ogn
sudo cp -v rtlsdr-ogn.conf /etc/rtlsdr-ogn.conf
sudo update-rc.d rtlsdr-ogn defaults
echo "Now edit /etc/rtlsdr-ogn.conf and put there correct username and directory"
 
