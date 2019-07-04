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
4. Use the the second [stratux status](http://hiltonsoftware.com/stratux/StratuxStatusMessage-V104.pdf) message.

See main/gen_gdl90.go:makeStratuxHeartbeat() for heartbeat (#1) format.

### Sleep mode

Stratux makes use of of ICMP Echo/Echo Reply and ICMP Destination Unreachable packets to determine the state of the application receiving GDL90 messages.

*Queueable messages* includes all UAT uplink messages, and in general: any messages whose value will not degrade for having been delayed for
the typical "sleep mode" period, whose content remains accurate despite said delay, and whose content is sufficiently redundant such
that if the typical delay period is exceeded then it can be discarded.

When a client enters sleep mode, queueable messages are entered into a FIFO queue of fixed size which should be sufficient to hold 10-25 minutes of data per client. Non-queueable messages that are directed
towards a client while in sleep mode are discarded. When in sleep mode, therefore, no GDL90 messages are received by the client.

There are three cases that are used to determine the state of a client:

1. Responding to ICMP Echo AND no ICMP Destination Unreachable received for destination port => not sleeping.
2. Not responding to ICMP Echo => sleeping.
3. Responding to ICMP Echo AND ICMP Destination Unreachable received for destination port => sleeping.

The sleep mode detection routine has two parts:

1. ICMP Echo packets are sent every 5 seconds. If a response is not received in the last 10 seconds, the client is in sleep mode.
2. If an ICMP Destination Unreachable is received in the last 5 seconds, the client is in sleep mode.
3. If an ICMP Destination Unreachable has been received in the last 15 seconds, but not in the last 5 seconds, the client is in *throttle mode*. In throttle mode, messages are sent at 0.1% of the normal rate. This gives the client a chance to recover or to respond that the port is closed.


__Note__: NEXRAD frames, METARs, and Winds Aloft, and other weather data may be delayed in reception to
the receiving application. The timestamp in the GDL90 message should always be used to for the observation
time, and **not the time the message was received**.

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


### Additional data/control available to EFBs

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

```javascript
{
  "Version": "v1.4r2",
  "Build": "ebd6b9bf5049aa5bb31c345c1eaa39648bc219a2",
  "HardwareBuild": "",
  "Devices": 1,
  "Connected_Users": 0,
  "DiskBytesFree": 60625375232,
  "UAT_messages_last_minute": 0,
  "UAT_messages_max": 0,
  "ES_messages_last_minute": 0,
  "ES_messages_max": 0,
  "UAT_traffic_targets_tracking": 0,
  "ES_traffic_targets_tracking": 0,
  "Ping_connected": false,
  "GPS_satellites_locked": 5,
  "GPS_satellites_seen": 7,
  "GPS_satellites_tracked": 9,
  "GPS_position_accuracy": 10.2,
  "GPS_connected": true,
  "GPS_solution": "GPS + SBAS (WAAS)",
  "GPS_detected_type": 55,
  "Uptime": 323020,
  "UptimeClock": "0001-01-01T00:05:23.02Z",
  "CPUTemp": 47.774,
  "NetworkDataMessagesSent": 0,
  "NetworkDataMessagesSentNonqueueable": 0,
  "NetworkDataBytesSent": 0,
  "NetworkDataBytesSentNonqueueable": 0,
  "NetworkDataMessagesSentLastSec": 0,
  "NetworkDataMessagesSentNonqueueableLastSec": 0,
  "NetworkDataBytesSentLastSec": 0,
  "NetworkDataBytesSentNonqueueableLastSec": 0,
  "UAT_METAR_total": 0,
  "UAT_TAF_total": 0,
  "UAT_NEXRAD_total": 0,
  "UAT_SIGMET_total": 0,
  "UAT_PIREP_total": 0,
  "UAT_NOTAM_total": 0,
  "UAT_OTHER_total": 0,
  "Errors": [
    
  ],
  "Logfile_Size": 34487043,
  "AHRS_LogFiles_Size": 0,
  "BMPConnected": true,
  "IMUConnected": true
}
```

* `http://192.168.10.1/getSettings` - get device settings. Example output:

```json
{
  "UAT_Enabled": true,
  "ES_Enabled": false,
  "Ping_Enabled": false,
  "GPS_Enabled": true,
  "BMP_Sensor_Enabled": true,
  "IMU_Sensor_Enabled": true,
  "NetworkOutputs": [
    {
      "Conn": null,
      "Ip": "",
      "Port": 4000,
      "Capability": 5,
      "MessageQueueLen": 0,
      "LastUnreachable": "0001-01-01T00:00:00Z",
      "SleepFlag": false,
      "FFCrippled": false
    }
  ],
  "SerialOutputs": null,
  "DisplayTrafficSource": false,
  "DEBUG": false,
  "ReplayLog": false,
  "AHRSLog": false,
  "IMUMapping": [
    -1,
    0
  ],
  "SensorQuaternion": [
    0.0068582877312501,
    0.0067230280142738,
    0.7140806859355,
    -0.69999752767998
  ],
  "C": [
    -0.019065523239845,
    -0.99225684377575,
    -0.019766228217414
  ],
  "D": [
    -2.7707754753258,
    5.544145023957,
    -1.890621662038
  ],
  "PPM": 0,
  "OwnshipModeS": "F00000",
  "WatchList": "",
  "DeveloperMode": false,
  "GLimits": "",
  "StaticIps": [
    
  ]
}
```
* `http://192.168.10.1/setSettings` - set device settings. Use an HTTP POST of JSON content in the format given above - posting only the fields containing the settings to be modified.

* `http://192.168.10.1/getSituation` - get GPS/AHRS information. Example output:

```json
{
  "GPSLastFixSinceMidnightUTC": 67337.6,
  "GPSLatitude": 39.108533,
  "GPSLongitude": -76.770862,
  "GPSFixQuality": 2,
  "GPSHeightAboveEllipsoid": 115.51,
  "GPSGeoidSep": -17.523,
  "GPSSatellites": 5,
  "GPSSatellitesTracked": 11,
  "GPSSatellitesSeen": 8,
  "GPSHorizontalAccuracy": 10.2,
  "GPSNACp": 9,
  "GPSAltitudeMSL": 170.10767,
  "GPSVerticalAccuracy": 8,
  "GPSVerticalSpeed": -0.6135171,
  "GPSLastFixLocalTime": "0001-01-01T00:06:44.24Z",
  "GPSTrueCourse": 0,
  "GPSTurnRate": 0,
  "GPSGroundSpeed": 0.77598433056951,
  "GPSLastGroundTrackTime": "0001-01-01T00:06:44.24Z",
  "GPSTime": "2017-09-26T18:42:17Z",
  "GPSLastGPSTimeStratuxTime": "0001-01-01T00:06:43.65Z",
  "GPSLastValidNMEAMessageTime": "0001-01-01T00:06:44.24Z",
  "GPSLastValidNMEAMessage": "$PUBX,04,184426.00,260917,240266.00,1968,18,-177618,-952.368,21*1A",
  "GPSPositionSampleRate": 0,
  "BaroTemperature": 37.02,
  "BaroPressureAltitude": 153.32,
  "BaroVerticalSpeed": 1.3123479,
  "BaroLastMeasurementTime": "0001-01-01T00:06:44.23Z",
  "AHRSPitch": -0.97934145732801,                     // Degrees. 3276.7 = Invalid.
  "AHRSRoll": -2.2013729217108,                       // Degrees. 3276.7 = Invalid.
  "AHRSGyroHeading": 187741.08073052,                 // Degrees. Process mod 360. 3276.7 = Invalid.
  "AHRSMagHeading": 3276.7,                           // Degrees. Process mod 360. 3276.7 = Invalid.
  "AHRSSlipSkid": 0.52267604604907,                   // Degrees. 3276.7 = Invalid.
  "AHRSTurnRate": 3276.7,                             // Degrees per second. 3276.7 = Invalid.
  "AHRSGLoad": 0.99847599584255,                      // Current G load, in G's. Reads 1 G at rest.
  "AHRSGLoadMin": 0.99815989027411,                   // Minimum recorded G load, in G's.
  "AHRSGLoadMax": 1.0043409597397,                    // Maximum recorded G load, in G's.
  "AHRSLastAttitudeTime": "0001-01-01T00:06:44.28Z",  // Stratux clock ticks since last attitude update. Reference against /getStatus -> UptimeClock.
  "AHRSStatus": 7                                     // Status bitmask. See main/sensors.go -> updateAHRSStatus().
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

* `http://192.168.10.1/calibrateAHRS` - run AHRS sensor calibration routine. Submit a blank POST to this URL.

* `http://192.168.10.1/cageAHRS` - "level" attitude display. Submit a blank POST to this URL.

* `http://192.168.10.1/resetGMeter` - reset G-meter to zero. Submit a blank POST to this URL.

* `http://192.168.10.1/restart` - restart Stratux application.

* `http://192.168.10.1/reboot` - reboot the system.

* `http://192.168.10.1/shutdown` - shutdown the system.

