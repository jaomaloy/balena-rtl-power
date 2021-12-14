#!/bin/sh

#Fetch and build rtl-sdr
echo "***** building rtl-sdr *****"
cd /usr/src/app
# clone the fork of librtlsdr for additional debug
git clone https://github.com/jaomaloy/librtlsdr.git rtl-sdr
cd rtl-sdr/
mkdir build
cd build
cmake ../ -DINSTALL_UDEV_RULES=ON -DDETACH_KERNEL_DRIVER=ON
make
sudo make install
sudo ldconfig
cd ../..
echo "***** finished building rtl-sdr *****"

#Disable the DVB-T driver, which would prevent the rtl_sdr tool from accessing the stick
#(if you want to use it for DVB-T reception later, you should undo this change):
echo "***** disabling DVB-T driver *****"
sudo bash -c 'echo -e "\n# for RTL-SDR:\nblacklist dvb_usb_rtl28xxu\n" >> /etc/modprobe.d/blacklist.conf'
