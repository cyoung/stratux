# app side integration

### Protocol & network information

Stratux uses GDL90 protocol over port 4000 UDP. All messages are sent as **unicast** messages. When a device is connected to the stratux Wi-Fi
network and a DHCP lease is issued to the device, the IP of the DHCP lease is added to the list of clients receiving GDL90 messages.


The GDL90 is "standard" with the exception of two non-standard GDL90-style messages: 0xCC (stratux heartbeat) and 0x4C (AHRS report).


### How to recognize stratux

In order of preference:

1. Look for 0xCC heartbeat message. This is sent at the same time as the GDL90 heartbeat (0x00) message.
2. Look for Wi-Fi network that **starts with** "stratux".
3. Detect 192.168.10.0/24 Wi-Fi connection, verify stratux status with JSON response from ws://192.168.10.1/status.

See main/gen_gdl90.go:makeStratuxHeartbeat() for heartbeat format.

### Sleep mode

Stratux makes use of of ICMP Echo/Echo Reply and ICMP Destination Unreachable packets to determine the state of the application receiving GDL90 messages.

*Queueable messages* includes all UAT uplink messages, and in general: any messages whose value will not degrade for having been delayed for
the typical "sleep mode" period, whose content remains accurate despite said delay, and whose content is sufficiently redundant such
that if the typical delay period is exceeded then it can be discarded.

When a client enters sleep mode, queueable messages are entered into a FIFO queue of fixed size which should be sufficient to hold 10-25 minutes of
data per client.

There are three cases that are used to detect the state of a client:

1. Responding to ICMP Echo AND no ICMP Destination Unreachable received for destination port => not sleeping.
2. Not responding to ICMP Echo => sleeping.
3. Responding to ICMP Echo AND ICMP Destination Unreachable received for destination port => sleeping.

It is important to note that NEXRAD frames, METARs, and Winds Aloft may be delayed in reception. The timestamp in the GDL90 message should always
be used to for the observation time, and not the time the message was received.

### Traffic handling

Stratux receives traffic updates from UAT downlink messages and/or 1090ES messages.

Traffic information roughly follows this path:

1. Traffic update is received (from any source). An entry for the target is created, indexed by the ICAO address of the target.
2. Any position/information updates received from any source for this ICAO address updates the entry created in #1.
3. If no updates are received from any source over 60 seconds, the target entry is deleted.
4. A GDL90 traffic report is sent for every target entry (having valid lat/lng) every 1 second.

When traffic information is being received both from UAT and 1090ES sources, it is not uncommon to see a flip/flop in tail numbers on targets.
Some 1090ES transponders will send the actual registration number of the aircraft, which then becomes a TIS-B target whose tail number may be
a squawk code.