echo "Setup OGN receiver binaries for operation with Raspberry PI GPU"
sudo apt-get -y install libjpeg-dev libconfig-dev fftw3-dev lynx telnet
sudo chown root gsm_scan
sudo chmod a+s  gsm_scan
sudo chown root ogn-rf
sudo chmod a+s  ogn-rf
sudo chown root rtlsdr-ogn
sudo chmod a+s  rtlsdr-ogn
# sudo mknod gpu_dev c 100 0
# mkfifo ogn-rf.fifo

