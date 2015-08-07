//
// Copyright 2015, Oliver Jowett <oliver@mutability.co.uk>
//

// This file is free software: you may copy, redistribute and/or modify it  
// under the terms of the GNU General Public License as published by the
// Free Software Foundation, either version 2 of the License, or (at your  
// option) any later version.  
//
// This file is distributed in the hope that it will be useful, but  
// WITHOUT ANY WARRANTY; without even the implied warranty of  
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU  
// General Public License for more details.
//
// You should have received a copy of the GNU General Public License  
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

#include <stdio.h>
#include <stdlib.h>
#include <math.h>
#include <string.h>

#include "uat.h"
#include "uat_decode.h"
#include "reader.h"

static void checksum_and_send(uint8_t *frame, int len, uint32_t parity);

// If you call this with constants for firstbit/lastbit
// gcc will do a pretty good job of crunching it down
// to just a couple of operations. Even more so if value
// is also constant.
static inline void setbits(uint8_t *frame, unsigned firstbit, unsigned lastbit, uint32_t value)
{
    // convert to 0-based:
    unsigned lb = lastbit-1;
    
    // align value with byte layout:
    unsigned offset = 7 - (lb&7);
    unsigned nb = (lastbit - firstbit + 1 + offset);
    uint32_t mask = (1 << (lastbit-firstbit+1))-1;
    uint32_t imask = ~(mask << offset);
    uint32_t aligned = (value & mask) << offset;

    frame[lb >> 3] = (frame[lb >> 3] & imask) | aligned;
    if (nb > 8)
        frame[(lb >> 3) - 1] = (frame[(lb >> 3) - 1] & (imask >> 8)) | (aligned >> 8);
    if (nb > 16)
        frame[(lb >> 3) - 2] = (frame[(lb >> 3) - 2] & (imask >> 16)) | (aligned >> 16);
    if (nb > 24)
        frame[(lb >> 3) - 3] = (frame[(lb >> 3) - 3] & (imask >> 24)) | (aligned >> 24);
}

static int encode_altitude(int ft)
{
    int i;

    i = (ft + 1000) / 25;
    if (i < 0) i = 0;
    if (i > 0x7FF) i = 0x7FF;

    return (i & 0x000F) | 0x0010 | ((i & 0x07F0) << 1);
}

static int encode_ground_speed(int kt)
{
    if (kt > 175)
        return 124;
    if (kt > 100)
        return (kt - 100) / 5 + 108;
    if (kt > 70)
        return (kt - 70) / 2 + 93;
    if (kt > 15)
        return (kt - 15) + 38;
    if (kt > 2)
        return (kt - 2) * 2 + 11;
    if (kt == 2)
        return 12;
    if (kt == 1)
        return 8;
    return 1;
}

static int encode_air_speed(int kt, int supersonic)
{
    int sign;

    if (kt < 0) {
        sign = 0x0400;
        kt = -kt;
    } else {
        sign = 0;
    }
    
    if (supersonic)
        kt = kt / 4;
    
    ++kt;
    if (kt > 1023)
        kt = 1023;

    return kt | sign;
}

static int encode_vert_rate(int rate)
{
    int sign;

    if (rate < 0) {
        sign = 0x200;
        rate = -rate;
    } else {
        sign = 0;
    }

    rate = (rate / 64) + 1;
    if (rate > 511)
        rate = 511;

    return rate | sign;
}

static double cprMod(double a, double b) {
    double res = fmod(a, b);
    if (res < 0) res += b;
    return res;
}

static int cprNL(double lat)
{
    if (lat < 0) lat = -lat;
    if (lat < 10.47047130) return 59;
    if (lat < 14.82817437) return 58;
    if (lat < 18.18626357) return 57;
    if (lat < 21.02939493) return 56;
    if (lat < 23.54504487) return 55;
    if (lat < 25.82924707) return 54;
    if (lat < 27.93898710) return 53;
    if (lat < 29.91135686) return 52;
    if (lat < 31.77209708) return 51;
    if (lat < 33.53993436) return 50;
    if (lat < 35.22899598) return 49;
    if (lat < 36.85025108) return 48;
    if (lat < 38.41241892) return 47;
    if (lat < 39.92256684) return 46;
    if (lat < 41.38651832) return 45;
    if (lat < 42.80914012) return 44;
    if (lat < 44.19454951) return 43;
    if (lat < 45.54626723) return 42;
    if (lat < 46.86733252) return 41;
    if (lat < 48.16039128) return 40;
    if (lat < 49.42776439) return 39;
    if (lat < 50.67150166) return 38;
    if (lat < 51.89342469) return 37;
    if (lat < 53.09516153) return 36;
    if (lat < 54.27817472) return 35;
    if (lat < 55.44378444) return 34;
    if (lat < 56.59318756) return 33;
    if (lat < 57.72747354) return 32;
    if (lat < 58.84763776) return 31;
    if (lat < 59.95459277) return 30;
    if (lat < 61.04917774) return 29;
    if (lat < 62.13216659) return 28;
    if (lat < 63.20427479) return 27;
    if (lat < 64.26616523) return 26;
    if (lat < 65.31845310) return 25;
    if (lat < 66.36171008) return 24;
    if (lat < 67.39646774) return 23;
    if (lat < 68.42322022) return 22;
    if (lat < 69.44242631) return 21;
    if (lat < 70.45451075) return 20;
    if (lat < 71.45986473) return 19;
    if (lat < 72.45884545) return 18;
    if (lat < 73.45177442) return 17;
    if (lat < 74.43893416) return 16;
    if (lat < 75.42056257) return 15;
    if (lat < 76.39684391) return 14;
    if (lat < 77.36789461) return 13;
    if (lat < 78.33374083) return 12;
    if (lat < 79.29428225) return 11;
    if (lat < 80.24923213) return 10;
    if (lat < 81.19801349) return 9;
    if (lat < 82.13956981) return 8;
    if (lat < 83.07199445) return 7;
    if (lat < 83.99173563) return 6;
    if (lat < 84.89166191) return 5;
    if (lat < 85.75541621) return 4;
    if (lat < 86.53536998) return 3;
    if (lat < 87.00000000) return 2;
    else return 1;
}

static int cprN(double lat, int odd)
{
    int nl = cprNL(lat) - (odd ? 1 : 0);
    if (nl < 1)
        nl = 1;
    return nl;
}

static int encode_cpr_lat(double lat, double lon, int odd, int surface)
{
    int NbPow = (surface ? 1<<19 : 1<<17);

    double Dlat = 360.0 / (odd ? 59 : 60);
    int YZ = floor(NbPow * cprMod(lat, Dlat) / Dlat + 0.5);

    return YZ & 0x1FFFF; // always a 17-bit field
}

static int encode_cpr_lon(double lat, double lon, int odd, int surface)
{
    int NbPow = (surface ? 1<<19 : 1<<17);

    double Dlat = 360.0 / (odd ? 59 : 60);
    int YZ = floor(NbPow * cprMod(lat, Dlat) / Dlat + 0.5);

    double Rlat = Dlat * (1.0 * YZ / NbPow + floor(lat / Dlat));
    double Dlon = (360.0 / cprN(Rlat, odd));
    int XZ = floor(NbPow * cprMod(lon, Dlon) / Dlon + 0.5);

    return XZ & 0x1FFFF; // always a 17-bit field
}

static int encode_imf(struct uat_adsb_mdb *mdb)
{
    // Encode the IMF bit for DF 18; this is 0 if the address
    // is a regular 24-bit ICAO address, or 1 if it uses a
    // different format.
    switch (mdb->address_qualifier) {
    case AQ_ADSB_ICAO:
    case AQ_TISB_ICAO:
        return 0;

    default:
        return 1;
    }
}

static void send_altitude_only(struct uat_adsb_mdb *mdb)
{
    uint8_t esnt_frame[14];
    int raw_alt;

    // Need barometric altitude, see if we have it
    if (mdb->altitude_type == ALT_BARO) {
        raw_alt = encode_altitude(mdb->altitude);
    } else if (mdb->sec_altitude_type == ALT_BARO) {
        raw_alt = encode_altitude(mdb->sec_altitude);
    } else {
        raw_alt = 0;
    }

    setbits(esnt_frame, 1, 5, 18);                 // DF=18, ES/NT
    setbits(esnt_frame, 6, 8, 6);                  // CF=6,  ADS-R
    setbits(esnt_frame, 9, 32, mdb->address);      // AA

    // ES:
    setbits(esnt_frame+4, 1, 5, 0);                // FORMAT TYPE CODE = 0, barometric altitude with no position
    setbits(esnt_frame+4, 6, 7, 0);                // SURVEILLANCE STATUS normal
    setbits(esnt_frame+4, 8, 8, encode_imf(mdb));  // IMF
    setbits(esnt_frame+4, 9, 20, raw_alt);         // ALTITUDE
    setbits(esnt_frame+4, 21, 21, 0);              // TIME (T)
    setbits(esnt_frame+4, 22, 22, 0);              // CPR FORMAT (F)
    setbits(esnt_frame+4, 23, 39, 0);              // ENCODED LATITUDE
    setbits(esnt_frame+4, 40, 56, 0);              // ENCODED LONGITUDE

    checksum_and_send(esnt_frame, 14, 0);
}

static void maybe_send_surface_position(struct uat_adsb_mdb *mdb)
{
    uint8_t esnt_frame[14];

    if (mdb->airground_state != AG_GROUND)
        return; // nope!

    setbits(esnt_frame, 1, 5, 18);                 // DF=18, ES/NT
    setbits(esnt_frame, 6, 8, 6);                  // CF=6,  ADS-R
    setbits(esnt_frame, 9, 32, mdb->address);      // AA

    setbits(esnt_frame+4, 1, 5, 8);                                            // FORMAT TYPE CODE = 8, surface position (NUCp=6)

    if (!mdb->speed_valid) {
        setbits(esnt_frame+4, 6, 12, 0);                                       // MOVEMENT: invalid
    } else {
        setbits(esnt_frame+4, 6, 12, encode_ground_speed(mdb->speed));         // MOVEMENT
    }

    if (mdb->track_type != TT_TRACK) {
        setbits(esnt_frame+4, 13, 13, 0);                                      // STATUS for ground track: invalid
        setbits(esnt_frame+4, 14, 20, 0);                                      // GROUND TRACK (TRUE)
    } else {
        setbits(esnt_frame+4, 13, 13, 1);                                      // STATUS for ground track: valid
        setbits(esnt_frame+4, 14, 20, mdb->track * 128 / 360);                 // GROUND TRACK (TRUE)
    }

    setbits(esnt_frame+4, 21, 21, encode_imf(mdb));                            // IMF

    // even frame:
    setbits(esnt_frame+4, 22, 22, 0);                                          // CPR FORMAT (F) = even
    setbits(esnt_frame+4, 23, 39, encode_cpr_lat(mdb->lat, mdb->lon, 0, 1));   // ENCODED LATITUDE
    setbits(esnt_frame+4, 40, 56, encode_cpr_lon(mdb->lat, mdb->lon, 0, 1));   // ENCODED LONGITUDE
    checksum_and_send(esnt_frame, 14, 0);

    // odd frame:
    setbits(esnt_frame+4, 22, 22, 1);                                          // CPR FORMAT (F) = odd
    setbits(esnt_frame+4, 23, 39, encode_cpr_lat(mdb->lat, mdb->lon, 1, 1));   // ENCODED LATITUDE
    setbits(esnt_frame+4, 40, 56, encode_cpr_lon(mdb->lat, mdb->lon, 1, 1));   // ENCODED LONGITUDE
    checksum_and_send(esnt_frame, 14, 0);
}

static void maybe_send_air_position(struct uat_adsb_mdb *mdb)
{
    uint8_t esnt_frame[14];
    int raw_alt;

    if (mdb->airground_state != AG_SUPERSONIC && mdb->airground_state != AG_SUBSONIC)
        return; // nope!

    if (!mdb->position_valid) {
        send_altitude_only(mdb);
        return;
    }        

    setbits(esnt_frame, 1, 5, 18);            // DF=18, ES/NT
    setbits(esnt_frame, 6, 8, 6);             // CF=6,  ADS-R
    setbits(esnt_frame, 9, 32, mdb->address); // AA

    // decide on a metype
    switch (mdb->altitude_type) {
    case ALT_BARO:
        setbits(esnt_frame+4, 1, 5, 18);           // FORMAT TYPE CODE = 18, airborne position (baro alt)
        raw_alt = encode_altitude(mdb->altitude);
        break;
        
    case ALT_GEO:
        setbits(esnt_frame+4, 1, 5, 22);           // FORMAT TYPE CODE = 22, airborne position (GNSS alt)
        raw_alt = encode_altitude(mdb->altitude);
        break;

    default:
        setbits(esnt_frame+4, 1, 5, 18);           // FORMAT TYPE CODE = 18, airborne position (baro alt)
        raw_alt = 0; // unavailable
        break;
    }

    setbits(esnt_frame+4, 6, 7, 0);                // SURVEILLANCE STATUS normal
    setbits(esnt_frame+4, 8, 8, encode_imf(mdb));  // IMF
    setbits(esnt_frame+4, 9, 20, raw_alt);         // ALTITUDE
    setbits(esnt_frame+4, 21, 21, 0);              // TIME (T)

    // even frame:
    setbits(esnt_frame+4, 22, 22, 0);                                               // CPR FORMAT (F) - even
    setbits(esnt_frame+4, 23, 39, encode_cpr_lat(mdb->lat, mdb->lon, 0, 0));        // ENCODED LATITUDE
    setbits(esnt_frame+4, 40, 56, encode_cpr_lon(mdb->lat, mdb->lon, 0, 0));        // ENCODED LONGITUDE
    checksum_and_send(esnt_frame, 14, 0);

    // odd frame:
    setbits(esnt_frame+4, 22, 22, 1);                                               // CPR FORMAT (F) - odd
    setbits(esnt_frame+4, 23, 39, encode_cpr_lat(mdb->lat, mdb->lon, 1, 0));        // ENCODED LATITUDE
    setbits(esnt_frame+4, 40, 56, encode_cpr_lon(mdb->lat, mdb->lon, 1, 0));        // ENCODED LONGITUDE
    checksum_and_send(esnt_frame, 14, 0);
}

static void maybe_send_air_velocity(struct uat_adsb_mdb *mdb)
{
    uint8_t esnt_frame[14];
    int supersonic;

    if (mdb->airground_state != AG_SUPERSONIC && mdb->airground_state != AG_SUBSONIC)
        return; // nope!

    if (!mdb->ew_vel_valid && !mdb->ns_vel_valid && mdb->vert_rate_source == ALT_INVALID) {
        // not really any point sending this
        return;
    }

    setbits(esnt_frame, 1, 5, 18);            // DF=18, ES/NT
    setbits(esnt_frame, 6, 8, 6);             // CF=6,  ADS-R
    setbits(esnt_frame, 9, 32, mdb->address); // AA

    supersonic = (mdb->airground_state == AG_SUPERSONIC);
    setbits(esnt_frame+4, 1, 5, 19);               // FORMAT TYPE CODE = 19, airborne velocity
    if (supersonic)
        setbits(esnt_frame+4, 6, 8, 2);            // SUBTYPE = 2, supersonic, speed over ground
    else
        setbits(esnt_frame+4, 6, 8, 1);            // SUBTYPE = 1, subsonic, speed over ground

    setbits(esnt_frame+4, 9, 9, encode_imf(mdb));  // IMF
    setbits(esnt_frame+4, 10, 10, 0);              // IFR
    setbits(esnt_frame+4, 11, 13, 0);              // NAVIGATIONAL UNCERTAINTY CATEGORY FOR VELOCITY

    // EAST/WEST DIRECTION BIT + EAST/WEST VELOCITY
    if (!mdb->ew_vel_valid)
        setbits(esnt_frame+4, 14, 24, 0);                             
    else
        setbits(esnt_frame+4, 14, 24, encode_air_speed(mdb->ew_vel, supersonic));

    // NORTH/SOUTH DIRECTION BIT + NORTH/SOUTH VELOCITY
    if (!mdb->ns_vel_valid)
        setbits(esnt_frame+4, 25, 35, 0);
    else
        setbits(esnt_frame+4, 25, 35, encode_air_speed(mdb->ns_vel, supersonic));

    switch (mdb->vert_rate_source) {
    case ALT_BARO:
        setbits(esnt_frame+4, 36, 36, 0);                                 // SOURCE = BARO
        setbits(esnt_frame+4, 37, 46, encode_vert_rate(mdb->vert_rate));  // SIGN BIT FOR VERTICAL RATE + VERTICAL RATE
        break;

    case ALT_GEO:
        setbits(esnt_frame+4, 36, 36, 1);                                 // SOURCE = GNSS
        setbits(esnt_frame+4, 37, 46, encode_vert_rate(mdb->vert_rate));  // SIGN BIT FOR VERTICAL RATE + VERTICAL RATE
        break;

    default:
        setbits(esnt_frame+4, 36, 36, 0);                                 // SOURCE = BARO
        setbits(esnt_frame+4, 37, 46, 0);                                 // SIGN BIT FOR VERTICAL RATE + VERTICAL RATE = 0, no information
        break;
    }

    setbits(esnt_frame+4, 47, 48, 0);              // RESERVED FOR TURN INDICATOR

    if (mdb->altitude_type != ALT_INVALID && mdb->sec_altitude_type != ALT_INVALID) {
        int delta, sign;

        if (mdb->altitude < mdb->sec_altitude) { // secondary above primary
            delta = mdb->sec_altitude - mdb->altitude;
            sign = mdb->altitude_type == ALT_BARO ? 0 : 1;
        } else { // primary above secondary
            delta = mdb->altitude - mdb->sec_altitude;
            sign = mdb->altitude_type == ALT_BARO ? 1 : 0;
        }

        delta = delta / 25 + 1;
        if (delta >= 127) delta = 127;
        setbits(esnt_frame+4, 49, 49, sign);              // DIFFERENCE SIGN BIT
        setbits(esnt_frame+4, 50, 56, delta);             // GNSS ALT DIFFERENCE FROM BARO ALT
    } else {
        setbits(esnt_frame+4, 49, 49, 0);                 // DIFFERENCE SIGN BIT
        setbits(esnt_frame+4, 50, 56, 0);                 // GNSS ALT DIFFERENCE FROM BARO ALT = 0, no information
    }

    checksum_and_send(esnt_frame, 14, 0);
}

// yeah, this could be done with a lookup table, meh.
static char *ais_charset = "@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_ !\"#$%&'()*+,-./0123456789:;<=>?";
static uint8_t char_to_ais(int ch)
{
    char *match;
    if (!ch)
        return 32;

    match = strchr(ais_charset, ch);
    if (match)
        return (uint8_t)(match - ais_charset);
    else
        return 32;
}

static unsigned encodeSquawk(char *squawkStr)
{
    unsigned squawk = strtoul(squawkStr, NULL, 16);
    unsigned encoded = 0;

    if (squawk & 0x1000) encoded |= 0x0800; // A1
    if (squawk & 0x2000) encoded |= 0x0200; // A2
    if (squawk & 0x4000) encoded |= 0x0080; // A4

    if (squawk & 0x0100) encoded |= 0x0020; // B1
    if (squawk & 0x0200) encoded |= 0x0008; // B2
    if (squawk & 0x0400) encoded |= 0x0002; // B4

    if (squawk & 0x0010) encoded |= 0x1000; // C1
    if (squawk & 0x0020) encoded |= 0x0400; // C2
    if (squawk & 0x0040) encoded |= 0x0100; // C4

    if (squawk & 0x0001) encoded |= 0x0010; // D1
    if (squawk & 0x0002) encoded |= 0x0004; // D2
    if (squawk & 0x0004) encoded |= 0x0001; // D4

    return encoded;
}

static void maybe_send_callsign(struct uat_adsb_mdb *mdb)
{
    uint8_t esnt_frame[14];
    int imf = encode_imf(mdb);

    // NB: we choose a CF value based on the address type (IMF value);
    // we shouldn't send CF=6 with no IMF bit for non-ICAO addresses
    // (see doc 9871 B.3.4.3)
    switch (mdb->callsign_type) {
    case CS_CALLSIGN:
        setbits(esnt_frame, 1, 5, 18);            // DF=18, ES/NT
        setbits(esnt_frame, 6, 8, imf ? 5 : 6);   // CF=6 for ICAO, CF=5 for non-ICAO
        setbits(esnt_frame, 9, 32, mdb->address); // AA

        if (mdb->emitter_category <= 7) {
            setbits(esnt_frame+4, 1, 5, 4);                         // FORMAT TYPE CODE = 4, aircraft category A
            setbits(esnt_frame+4, 6, 8, mdb->emitter_category & 7); // AIRCRAFT CATEGORY (A0 - A7)
        } else if (mdb->emitter_category <= 15) {
            setbits(esnt_frame+4, 1, 5, 3);                         // FORMAT TYPE CODE = 3, aircraft category B
            setbits(esnt_frame+4, 6, 8, mdb->emitter_category & 7); // AIRCRAFT CATEGORY (B0 - B7)
        } else if (mdb->emitter_category <= 23) {
            setbits(esnt_frame+4, 1, 5, 2);                         // FORMAT TYPE CODE = 2, aircraft category C
            setbits(esnt_frame+4, 6, 8, mdb->emitter_category & 7); // AIRCRAFT CATEGORY (C0 - C7)
        } else if (mdb->emitter_category <= 31) {
            setbits(esnt_frame+4, 1, 5, 1);                         // FORMAT TYPE CODE = 1, aircraft category D
            setbits(esnt_frame+4, 6, 8, mdb->emitter_category & 7); // AIRCRAFT CATEGORY (D0 - D7)
        } else {
            // reserved, map to A0
            setbits(esnt_frame+4, 1, 5, 4);                         // FORMAT TYPE CODE = 4, aircraft category A
            setbits(esnt_frame+4, 6, 8, 0);                         // AIRCRAFT CATEGORY A0
        }

        // Map callsign
        setbits(esnt_frame+4, 9, 14, char_to_ais(mdb->callsign[0]));
        setbits(esnt_frame+4, 15, 20, char_to_ais(mdb->callsign[1]));
        setbits(esnt_frame+4, 21, 26, char_to_ais(mdb->callsign[2]));
        setbits(esnt_frame+4, 27, 32, char_to_ais(mdb->callsign[3]));
        setbits(esnt_frame+4, 33, 38, char_to_ais(mdb->callsign[4]));
        setbits(esnt_frame+4, 39, 44, char_to_ais(mdb->callsign[5]));
        setbits(esnt_frame+4, 45, 50, char_to_ais(mdb->callsign[6]));
        setbits(esnt_frame+4, 51, 56, char_to_ais(mdb->callsign[7]));
        checksum_and_send(esnt_frame, 14, 0);
        break;

    case CS_SQUAWK:
        if (imf) {
            // Non-ICAO address, send as DF18 "test message"
            setbits(esnt_frame, 1, 5, 18);            // DF=18, ES/NT
            setbits(esnt_frame, 6, 8, 5);             // CF=5, TIS-B retransmission with non-ICAO address
            setbits(esnt_frame, 9, 32, mdb->address); // AA

            setbits(esnt_frame+4, 1, 5, 23);                           // FORMAT TYPE CODE = 23, test message
            setbits(esnt_frame+4, 6, 8, 7);                            // subtype = 7, squawk
            setbits(esnt_frame+4, 9, 21, encodeSquawk(mdb->callsign));

            checksum_and_send(esnt_frame, 14, 0);
        } else {
            // ICAO address, send as DF5
            setbits(esnt_frame, 1, 5, 5);            // DF=5, Surveillance Identity Reply
            setbits(esnt_frame, 6, 8, 0);            // Flight Status
            setbits(esnt_frame, 9, 13, 0);           // Downlink Request
            setbits(esnt_frame, 14, 19, 0);          // Utility Message
            setbits(esnt_frame, 20, 32, encodeSquawk(mdb->callsign)); // Identity

            checksum_and_send(esnt_frame, 7, mdb->address); // put address in checksum (Address/Parity)
        }

        break;

    default:
        break;
    }
}

// Generator polynomial for the Mode S CRC:
#define MODES_GENERATOR_POLY 0xfff409U

// CRC values for all single-byte messages;
// used to speed up CRC calculation.
static uint32_t crc_table[256];

static void initCrcTables()
{
    int i;
    for (i = 0; i < 256; ++i) {
        uint32_t c = i << 16;
        int j;
        for (j = 0; j < 8; ++j) {
            if (c & 0x800000)
                c = (c<<1) ^ MODES_GENERATOR_POLY;
            else
                c = (c<<1);
        }

        crc_table[i] = c & 0x00ffffff;
    }
}

static uint32_t checksum(uint8_t *message, int n)
{
    uint32_t rem = 0;
    int i;

    for (i = 0; i < n; ++i) {
        rem = (rem << 8) ^ crc_table[message[i] ^ ((rem & 0xff0000) >> 16)];
        rem = rem & 0xffffff;
    }

    return rem;
}

static void checksum_and_send(uint8_t *frame, int len, uint32_t parity)
{
    int j;
    uint32_t rem = checksum(frame, len-3) ^ parity;

    frame[len-3] = (rem & 0xFF0000) >> 16;
    frame[len-2] = (rem & 0x00FF00) >> 8;
    frame[len-1] = (rem & 0x0000FF);

    fprintf(stdout, "*");
    for (j = 0; j < len; j++)
        fprintf(stdout, "%02X", frame[j]);
    fprintf(stdout, ";\n");
    fflush(stdout);
}

static void generate_esnt(struct uat_adsb_mdb *mdb)
{
    maybe_send_surface_position(mdb);
    maybe_send_air_position(mdb);
    maybe_send_air_velocity(mdb);
    maybe_send_callsign(mdb);

}

static void handle_frame(frame_type_t type, uint8_t *frame, int len, void *extra)
{
    if (type == UAT_DOWNLINK) {
        struct uat_adsb_mdb mdb;
        uat_decode_adsb_mdb(frame, &mdb);
        generate_esnt(&mdb);
    }
}        

int main(int argc, char **argv)
{
    struct dump978_reader *reader;
    int framecount;

    initCrcTables();

    reader = dump978_reader_new(0,0);
    if (!reader) {
        perror("dump978_reader_new");
        return 1;
    }
    
    while ((framecount = dump978_read_frames(reader, handle_frame, NULL)) > 0)
        ;

    if (framecount < 0) {
        perror("dump978_read_frames");
        return 1;
    }

    return 0;
}

