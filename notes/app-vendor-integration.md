# app side integration

### Protocol & network information

Stratux uses GDL90 protocol over port 4000 UDP. All messages are sent as **unicast** messages. When a device is connected to the stratux Wi-Fi
network and a DHCP lease is issued to the device, the IP of the DHCP lease is added to the list of clients receiving GDL90 messages.


The GDL90 is "standard" with the exception of three non-standard GDL90-style messages: `0xCC` (stratux heartbeat), `0x5358` (another stratux heartbeat), and `0x4C` (AHRS report).

### How to recognize stratux

In order of preference:

1. Look for `0xCC` (or `0x5358`) GDL90 heartbeat message. This is sent at the same time as the GDL90 heartbeat (0x00) message.
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


### Additional data available to EFBs

Stratux makes available a webserver to retrieve statistics which may be useful to EFBs:

* `http://192.168.10.1/getTowers` - a list of ADS-B towers received with attached message receipt and signal level statistics. Example output:

```json
{
  "(28.845592,-96.920400)": {
    "Lat": 28.845591545105,
    "Lng": -96.920399665833,
    "Signal_strength_last_minute": 55,
    "Signal_strength_max": 69,
    "Messages_last_minute": 97,
    "Messages_total": 196
  },
  "(29.266505,-98.309097)": {
    "Lat": 29.266505241394,
    "Lng": -98.309097290039,
    "Signal_strength_last_minute": 78,
    "Signal_strength_max": 78,
    "Messages_last_minute": 1,
    "Messages_total": 3
  },
  "(29.702547,-96.900787)": {
    "Lat": 29.702546596527,
    "Lng": -96.900787353516,
    "Signal_strength_last_minute": 87,
    "Signal_strength_max": 119,
    "Messages_last_minute": 94,
    "Messages_total": 203
  }
}
```

* `http://192.168.10.1/getStatus` - device status and statistics. Example output (commented JSON):

```json
{
  "Version": "v0.5b1",            // Software version.
  "Devices": 0,                   // Number of radios connected.
  "Connected_Users": 1,           // Number of WiFi devices connected.
  "UAT_messages_last_minute": 0,  // UAT messages received in last minute.
  "UAT_messages_max": 17949,      // Max UAT messages received in a minute (since last reboot).
  "ES_messages_last_minute": 0,   // 1090ES messages received in last minute.
  "ES_messages_max": 0,           // Max 1090ES messages received in a minute (since last reboot).
  "GPS_satellites_locked": 0,     // Number of GPS satellites used in last GPS lock.
  "GPS_connected": true,          // GPS unit connected and functioning.
  "GPS_solution": "",             // "DGPS (WAAS)", "3D GPS", "N/A", or "" when GPS not connected/enabled.
  "Uptime": 227068,               // Device uptime (in milliseconds).
  "CPUTemp": 42.236               // CPU temperature (in ºC).
}
```

* `http://192.168.10.1/getSettings` - get device settings. Example output:

```json
{
  "UAT_Enabled": true,
  "ES_Enabled": false,
  "Ping_Enabled": false,
  "GPS_Enabled": true,
  "NetworkOutputs": [
    {
      "Conn": null,
      "Ip": "",
      "Port": 4000,
      "Capability": 5
    },
    {
      "Conn": null,
      "Ip": "",
      "Port": 49002,
      "Capability": 2
    }
  ],
  "DEBUG": false,
  "ReplayLog": true,
  "PPM": 0,
  "OwnshipModeS": "F00000",
  "WatchList": ""
}
```
* `http://192.168.10.1/setSettings` - set device settings. Use an HTTP POST of JSON content in the format given above - posting only the fields containing the settings to be modified.

* `http://192.168.10.1/getSituation` - get GPS/AHRS information. Example output:

```json
{
  "Lat": 39.108533,
  "Lng": -76.770862,
  "Satellites": 7,
  "Accuracy": 5.88,
  "NACp": 10,
  "Alt": 170.10767,
  "LastFixLocalTime": "2015-12-18T23:47:06.015563066Z",
  "TrueCourse": 0,
  "GroundSpeed": 0,
  "LastGroundTrackTime": "0001-01-01T00:00:00Z",
  "Temp": 6553,
  "Pressure_alt": 231.27980834234,
  "Pitch": -0.006116937627108,
  "Roll": -0.026442866350631,
  "Gyro_heading": 45.844213419776,
  "LastAttitudeTime": "2015-12-18T23:47:06.774039623Z"
}
```


* `ws://192.168.10.1/traffic` - traffic stream. On initial connect, all currently tracked traffic targets are dumped. Updates are streamed as they are received. Example output:

Initial connect:

```json
{"Icao_addr":2837120,"OnGround":false,"Lat":42.19293,"Lng":-83.92148,"Position_valid":true,"Alt":3400,"Track":9,"Speed":92,"Speed_valid":true,"Vvel":0,"Tail":"","Last_seen":"2015-12-22T21:29:22.241048727Z","Last_source":2}
{"Icao_addr":2836155,"OnGround":false,"Lat":42.122932,"Lng":-84.17615,"Position_valid":true,"Alt":2800,"Track":158,"Speed":105,"Speed_valid":true,"Vvel":0,"Tail":"","Last_seen":"2015-12-22T21:29:22.241543881Z","Last_source":2}
```

Subsequent update (2837120 = 2B4A80 reports a newer position, altitude increased from 2,800' to 3,400'):

```json
{"Icao_addr":2837120,"OnGround":false,"Lat":42.193336,"Lng":-83.92136,"Position_valid":true,"Alt":3400,"Track":9,"Speed":92,"Speed_valid":true,"Vvel":0,"Tail":"","Last_seen":"2015-12-22T21:29:22.252914555Z","Last_source":2}
```
