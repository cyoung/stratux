---
layout: post
title:  "New Low Power v2 SDRs and comparisons"
date:   2017-03-23 21:52:10 +0000
categories: hardware sdrs
author: cyoung
---

New Low Power v2 SDRs are evaluated along with other common radios to compare noise floor and heat dissipation properties.

The new [Low Power v2](http://a.co/bxLCOdk) are in, so I decided to go back and do some further testing on the four main SDRs we've used in the past year.

The changes in this version were a switch to a 4-layer PCB and improved layout. The changes actually made a big difference, the Low Power v2 is the coolest running "Nano" SDR yet. There were big improvements in sensitivity and further reduced power consumption in this version as well.

Here's what I've found, starting with the earliest used SDR (for stratux) to the newest.

Description|Power|Peak R820T2 Temp|Thermal Image|Noise Floor Scan
:--|:--|:--|:--|:--
NESDR Nano 2|1.362W|243&deg;F|[here](http://i.imgur.com/nzKxu6S.jpg)|[here](http://i.imgur.com/20WttLr.png)
Generic Nano|1.318W|202&deg;F|[here](http://i.imgur.com/hpkdV52.jpg)|[here](http://i.imgur.com/5gkaFQO.png)
Low Power v1|1.003W|200&deg;F|[here](http://i.imgur.com/caeAeFC.jpg)|[here](http://i.imgur.com/EIc1tsh.png)
Low Power v2|0.933W|186&deg;F|[here](http://i.imgur.com/Gn4vUv2.jpg)|[here](http://i.imgur.com/zzjHzIS.png)

For the noise floor graphs, the SDRs were hooked up to a 50 ohm dummy load and `rtl_power` run to get the values.
e.g.,

`rtl_power -f 24M:1.7G:1M -g 50 -i 15m -1 nf-nesdr_nano2.csv`

The thermal images were taken after each SDR was let run `rtl_test` for 10 minutes. Each one is centered on the R820T2 (so the Low Power SDRs are flipped over, since it's on the bottom).  The power measurements were taken with a cheap USB meter over the same period.

[Here](http://i.imgur.com/yce2747.png) is a plot showing all of the above four SDR noise floor scan results.

This gave some pretty interesting results:

1. The NESDR Nano 2 loses in pretty much every aspect except for noise floor on VHF frequencies compared against the Low Power v1.
2. You can see the effects of heat on the R820T2 above 1.4 GHz.
3. The "Generic Nano" was always a great performer in terms of sensitivity.
4. For ~0.8W (in a dual-band build) less power, the cost is 0.41 dB @ 1090 MHz and 0.64 dB @ 978 MHz (compared to the Generic Nano).

