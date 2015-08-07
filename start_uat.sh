#!/bin/bash

rtl_sdr -f 978000000 -s 2083334 -g 48 - | dump978 | gen_gdl90

