#!/bin/bash


cat $1 | grep -a -v ^START | grep -a -v ^PAUSE | grep -a -v ^UNPAUSE | grep -a PUBX,00 | cut -d, -f1,5,6,7,8,9