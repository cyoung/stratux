---
layout: post
title:  "Nano SDRs Comparison Update"
date:   2017-03-09 13:37:00 +0000
categories: hardware sdrs
author: cyoung
---

Much hasn't changed since the [this comparison](http://stratux.me/hardware/sdrs/2017/03/23/new-low-power-v2-sdrs-and-comparisons.html) was made about a year ago.

The only real new product since then has been the NESDR Nano 3. It features a new "nano" sized SDR covered with thermal pads on both sides and inside a custom aluminum enclosure. Since the previous NESDR Nano series had serious issues with heat dissipation, I figured it would be instructive to do the same noise floor tests as before and see how it responds.

`rtl_power -f 24M:1.7G:1M -g 50 -i 15m -1 nf-nesdr_nano3.csv`

[Here](https://i.imgur.com/Vd3e1Wi.png) are the results.

Here's a summary of what I found:

1. The aluminum enclosure gets pretty hot--seems hot enough to burn.
2. Nano 3 has a lower noise floor than the Low Power v2 from 24 MHz - 110 MHz, then above 890 MHz (except for noise spurs). It was surprising to see the Nano 3 have issues in the 120 MHz-890 MHz range.
3. Low Power v2 has less (and less harsh) noise spurs above 900 MHz.
4. Despite better heat dissipation, the Nano 3 still had issues at 1.5 GHz+.
5. The generic nano SDR is still the low-noise winner.

