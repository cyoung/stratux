# dump978

Experimental demodulator/decoder for 978MHz UAT signals.

## A note about future development

I'm in Europe which doesn't use UAT, so there won't be much spontaneous
development going on now that the demodulator is at a basic "it works" stage.

I'm happy to look at signal or message captures and help with further
development, but it really needs to be driven by whoever is actually using the
code to receive UAT!

## Demodulator

dump978 is the demodulator. It expects 8-bit I/Q samples on stdin at
2.083334MHz, for example:

````
$ rtl_sdr -f 978000000 -s 2083334 -g 48 - | ./dump978
````

It outputs one one line per demodulated message, in the form:

````
+012345678..; this is an uplink message
-012345678..; this is a downlink message
````

For parsers: ignore everything between the first semicolon and newline that
you don't understand, it will be used for metadata later. See reader.[ch] for
a reference implementation.

## Decoder

To decode messages into a readable form use uat2text:

````
$ rtl_sdr -f 978000000 -s 2083334 -g 48 - | ./dump978 | ./uat2text
````

## Sample data

Around 1100 sample messages are in the file sample-data.txt.gz. They are the
output of the demodulator from various RF captures I have on hand. This file
can be fed to uat2text etc:

$ zcat sample-data.txt.gz | ./uat2text

When testing, this is much easier on your CPU (and disk space!) than starting
from the raw RF captures.

## Filtering for just uplink or downlink messages

As the uplink and downlink messages start with different characters, you can
filter for just one type of message very easily with grep:

````
  # Uplink messages only:
$ zcat sample-data.txt.gz | grep "^+" | ./uat2text
  # Downlink messages only:
$ zcat sample-data.txt.gz | grep "^-" | ./uat2text
````

## Map generation via uat2json

uat2json writes aircraft.json files in the format expected by dump1090's
map html/javascript.

To set up a live map feed:

1) Get a copy of dump1090, we're going to reuse its mapping html/javascript:

````
$ git clone https://github.com/mutability/dump1090 dump1090-copy
````

2) Put the html/javascript somewhere your webserver can reach:

````
$ mkdir /var/www/dump978map
$ cp -a dump1090-copy/public_html/* /var/www/dump978map/
````

3) Create an empty "data" subdirectory

````
$ mkdir /var/www/dump978map/data
````

4) Feed uat2json from dump978:

````
$ rtl_sdr -f 978000000 -s 2083334 -g 48 - | \
  ./dump978 | \
  ./uat2json /var/www/dump978map/data
````

5) Go look at http://localhost/dump978map/

## uat2esnt: convert UAT ADS-B messages to Mode S ADS-B messages.

Warning: This one is particularly experimental.

uat2esnt accepts 978MHz UAT downlink messages on stdin and
generates 1090MHz Extended Squitter messages on stdout.

The generated messages mostly use DF18 with CF=6, which is
for rebroadcasts of ADS-B messages (ADS-R).

The output format is the "AVR" text format; this can be
fed to dump1090 on port 30001 by default. Other ADS-B tools
may accept it too - e.g. VRS seems to accept most of it (though
it ignores DF18 CF=5 messages which are generated for
non-ICAO-address callsign/squawk information.

You'll want a pipeline like this:

````
$ rtl_sdr -f 978000000 -s 2083334 -g 48 - | \
  ./dump978 | \
  ./uat2esnt | \
  nc -q1 localhost 30001
````
