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
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <limits.h>

#include <time.h>
#include <sys/select.h>
#include <errno.h>

#include "uat.h"
#include "uat_decode.h"
#include "reader.h"

#define NON_ICAO_ADDRESS 0x1000000U

struct aircraft {
    struct aircraft *next;
    uint32_t address;

    uint32_t messages;
    time_t last_seen;
    time_t last_seen_pos;

    int position_valid : 1;
    int altitude_valid : 1;
    int track_valid : 1;
    int speed_valid : 1;
    int vert_rate_valid : 1;

    airground_state_t airground_state;
    char callsign[9];
    char squawk[9];

    // if position_valid:
    double lat;
    double lon;

    // if altitude_valid:
    int32_t altitude; // in feet
    
    // if track_valid:
    uint16_t track;

    // if speed_valid:
    uint16_t speed; // in kts

    // if vert_rate_valid:
    int16_t vert_rate; // in ft/min
};        

static struct aircraft *aircraft_list;
static time_t NOW;
static const char *json_dir;

static struct aircraft *find_aircraft(uint32_t address)
{
    struct aircraft *a;
    for (a = aircraft_list; a; a = a->next)
        if (a->address == address)
            return a;
    return NULL;
}

static struct aircraft *find_or_create_aircraft(uint32_t address)
{
    struct aircraft *a = find_aircraft(address);
    if (a)
        return a;

    a = calloc(1, sizeof(*a));
    a->address = address;
    a->airground_state = AG_RESERVED;

    a->next = aircraft_list;
    aircraft_list = a;

    return a;
}

static void expire_old_aircraft()
{
    struct aircraft *a, **last;    
    for (last = &aircraft_list, a = *last; a; a = *last) {
        if ((NOW - a->last_seen) > 300) {
            *last = a->next;
            free(a);
        } else {
            last = &a->next;
        }
    }
}

static uint32_t message_count;

static void process_mdb(struct uat_adsb_mdb *mdb)
{
    struct aircraft *a;
    uint32_t addr;
    
    ++message_count;

    switch (mdb->address_qualifier) {
    case AQ_ADSB_ICAO:
    case AQ_TISB_ICAO:
        addr = mdb->address;
        break;

    default:
        addr = mdb->address | NON_ICAO_ADDRESS;
        break;
    }
   
    a = find_or_create_aircraft(addr);
    a->last_seen = NOW;
    ++a->messages;
    
    // copy state into aircraft
    if (mdb->airground_state != AG_RESERVED)
        a->airground_state = mdb->airground_state;

    if (mdb->position_valid) {
        a->position_valid = 1;
        a->lat = mdb->lat;
        a->lon = mdb->lon;
        a->last_seen_pos = NOW;
    }
        
    if (mdb->altitude_type != ALT_INVALID) {
        a->altitude_valid = 1;
        a->altitude = mdb->altitude;
    }

    if (mdb->track_type != TT_INVALID) {
        a->track_valid = 1;
        a->track = mdb->track;
    }

    if (mdb->speed_valid) {
        a->speed_valid = 1;
        a->speed = mdb->speed;
    }
    
    if (mdb->vert_rate_source != ALT_INVALID) {
        a->vert_rate_valid = 1;
        a->vert_rate = mdb->vert_rate;
    }
    
    if (mdb->callsign_type == CS_CALLSIGN)
        strcpy(a->callsign, mdb->callsign);
    else if (mdb->callsign_type == CS_SQUAWK)
        strcpy(a->squawk, mdb->callsign);

    if (mdb->sec_altitude_type != ALT_INVALID) {
        // only use secondary if no primary is available
        if (!a->altitude_valid || mdb->altitude_type == ALT_INVALID) {
            a->altitude_valid = 1;
            a->altitude = mdb->sec_altitude;
        }
    }
}

static int write_receiver_json(const char *dir)
{
    char path[PATH_MAX];
    FILE *f;

    if (snprintf(path, PATH_MAX, "%s/receiver.json.new", dir) >= PATH_MAX) {
        fprintf(stderr, "write_receiver_json: path too long\n");
        return 0;
    }

    if (!(f = fopen(path, "w"))) {
        fprintf(stderr, "fopen(%s): %m\n", path);
        return 0;
    }

    fprintf(f,
            "{\n"
            "  \"version\" : \"dump978-uat2json\",\n"
            "  \"refresh\" : 1000,\n"
            "  \"history\" : 0\n"
            "}\n");
    fclose(f);

    return 1;
}

static int write_aircraft_json(const char *dir)
{
    char path[PATH_MAX];
    char path_new[PATH_MAX];
    FILE *f;
    struct aircraft *a;

    if (snprintf(path, PATH_MAX, "%s/aircraft.json", dir) >= PATH_MAX || snprintf(path_new, PATH_MAX, "%s/aircraft.json.new", dir) >= PATH_MAX) {
        fprintf(stderr, "write_aircraft_json: path too long\n");
        return 0;
    }

    if (!(f = fopen(path_new, "w"))) {
        fprintf(stderr, "fopen(%s): %m\n", path_new);
        return 0;
    }

    fprintf(f,
            "{\n"
            "  \"now\" : %u,\n"
            "  \"messages\" : %u,\n"
            "  \"aircraft\" : [\n",
            (unsigned)NOW,
            message_count);
    

    for (a = aircraft_list; a; a = a->next) {
        if (a != aircraft_list)
            fprintf(f, ",\n");
        fprintf(f,
                "    {\"hex\":\"%s%06x\"",
                (a->address & NON_ICAO_ADDRESS) ? "~" : "",
                a->address & 0xFFFFFF);

        if (a->squawk[0])
            fprintf(f, ",\"squawk\":\"%s\"", a->squawk);
        if (a->callsign[0])
            fprintf(f, ",\"flight\":\"%s\"", a->callsign);
        if (a->position_valid)
            fprintf(f, ",\"lat\":%.6f,\"lon\":%.6f,\"seen_pos\":%u", a->lat, a->lon, (unsigned) (NOW - a->last_seen_pos));        
        if (a->altitude_valid)
            fprintf(f, ",\"altitude\":%d", a->altitude);
        if (a->vert_rate_valid)
            fprintf(f, ",\"vert_rate\":%d", a->vert_rate);
        if (a->track_valid)
            fprintf(f, ",\"track\":%u", a->track);
        if (a->speed_valid)
            fprintf(f, ",\"speed\":%u", a->speed);
        fprintf(f, ",\"messages\":%u,\"seen\":%u,\"rssi\":0}",
                a->messages, (unsigned) (NOW - a->last_seen));
    }

    fprintf(f,
            "\n  ]\n"
            "}\n");
    fclose(f);

    if (rename(path_new, path) < 0) {
        fprintf(stderr, "rename(%s,%s): %m\n", path_new, path);
        return 0;
    }

    return 1;
}
    
static void periodic_work()
{
    static time_t next_write;
    if (NOW >= next_write) {
        expire_old_aircraft();
        write_aircraft_json(json_dir);
        next_write = NOW + 1;
    }
}

static void handle_frame(frame_type_t type, uint8_t *frame, int len, void *extra)
{
    struct uat_adsb_mdb mdb;

    if (type != UAT_DOWNLINK)
        return;

    if (len == SHORT_FRAME_DATA_BYTES) {
        if ((frame[0] >> 3) != 0) {
            fprintf(stderr, "short frame with non-zero type\n");
            return;
        }
    } else if (len == LONG_FRAME_DATA_BYTES) {
        if ((frame[0] >> 3) == 0) {
            fprintf(stderr, "long frame with zero type\n");
            return;
        }
    } else {
        fprintf(stderr, "odd frame size: %d\n", len);
        return;
    }

    uat_decode_adsb_mdb(frame, &mdb);
    //uat_display_adsb_mdb(&mdb, stdout);    
    process_mdb(&mdb);
}                                                        

static void read_loop()
{
    struct dump978_reader *reader;

    reader = dump978_reader_new(0, 1);
    if (!reader) {
        perror("dump978_reader_new");
        return;
    }

    for (;;) {
        fd_set readset, writeset, excset;
        struct timeval timeout;
        int framecount;

        FD_ZERO(&readset);
        FD_ZERO(&writeset);
        FD_ZERO(&excset);
        FD_SET(0, &readset);
        FD_SET(0, &excset);
        timeout.tv_sec = 0;
        timeout.tv_usec = 500000;

        select(1, &readset, &writeset, &excset, &timeout);

        NOW = time(NULL);
        framecount = dump978_read_frames(reader, handle_frame, NULL);

        if (framecount == 0)
            break;

        if (framecount < 0 && errno != EAGAIN && errno != EINTR && errno != EWOULDBLOCK) {
            perror("dump978_read_frames");
            break;
        }

        periodic_work();
    }

    dump978_reader_free(reader);
}                    

int main(int argc, char **argv)
{
    if (argc < 2) {
        fprintf(stderr,
                "Syntax: %s <dir>\n"
                "\n"
                "Reads UAT messages on stdin.\n"
                "Periodically writes aircraft state to <dir>/aircraft.json\n"
                "Also writes <dir>/receiver.json once on startup\n",
                argv[0]);
        return 1;
    }

    json_dir = argv[1];

    if (!write_receiver_json(json_dir)) {
        fprintf(stderr, "Failed to write receiver.json - check permissions?\n");
        return 1;
    }
    read_loop();
    write_aircraft_json(json_dir);
    return 0;
}
