#!/bin/bash

/usr/local/bin/rtl_sdr -f 978000000 -s 2083334 -g 48 -d 0 - | /usr/bin/dump978 | /usr/bin/gen_gdl90

