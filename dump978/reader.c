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

#include <stdlib.h>
#include <fcntl.h>
#include <errno.h>
#include <unistd.h>
#include <string.h>

#include "uat.h"
#include "reader.h"

struct dump978_reader {
    int fd;
    char buf[4096];
    uint8_t frame[UPLINK_FRAME_DATA_BYTES]; // max uplink frame size
    int used;
};

static int process_input(struct dump978_reader *reader, frame_handler_t handler, void *handler_data);
static int process_line(struct dump978_reader *reader, frame_handler_t handler, void *handler_data, char *p, char *end);
static int hexbyte(char *buf);

struct dump978_reader *dump978_reader_new(int fd, int nonblock)
{
    struct dump978_reader *reader = calloc(1, sizeof(*reader));
    if (!reader)
        return NULL;

    if (nonblock) {
        int flags = fcntl(fd, F_GETFL);
        if (flags < 0 || fcntl(fd, F_SETFL, flags | O_NONBLOCK) < 0) {
            int save_errno = errno;
            free(reader);
            errno = save_errno;
            return NULL;
        }
    }
        
    reader->fd = fd;
    reader->used = 0;
    return reader;
}
    
int dump978_read_frames(struct dump978_reader *reader,
                        frame_handler_t handler,
                        void *handler_data)
{
    int framecount = 0;
    ssize_t bytes_read;

    if (!reader) {
        errno = EINVAL;
        return -1;
    }

    for (;;) {
        if (reader->used == sizeof(reader->buf)) {
            // line too long, ditch input
            reader->used = 0;
        }

        bytes_read = read(reader->fd,
                          reader->buf + reader->used,
                          sizeof(reader->buf) - reader->used);
        if (bytes_read <= 0)
            break;
        
        reader->used += bytes_read;

        framecount += process_input(reader, handler, handler_data);
    }

    if (bytes_read == 0)
        return framecount; // EOF

    // only report EAGAIN et al if no frames were read
    if (errno == EAGAIN || errno == EWOULDBLOCK || errno == EINTR)
        return (framecount > 0 ? framecount : -1);

    return -1; // propagate unexpected error
}

void dump978_reader_free(struct dump978_reader *reader)
{
    if (!reader)
        return;

    free(reader);
}

static int process_input(struct dump978_reader *reader, frame_handler_t handler, void *handler_data)
{
    char *p = reader->buf;
    char *end = reader->buf + reader->used;
    int framecount = 0;

    while (p < end) {
        char *newline;

        newline = memchr(p, '\n', end - p);
        if (newline == NULL)
            break;
        
        if (*p == '-' || *p == '+')
            framecount += process_line(reader, handler, handler_data, p, newline);
        
        p = newline+1;
    }

    if (p >= end) {
        reader->used = 0;
    } else {
        reader->used = end - p;
        memmove(reader->buf, p, reader->used);
    }

    return framecount;
}

static int process_line(struct dump978_reader *reader, frame_handler_t handler, void *handler_data, char *p, char *end)
{
    uint8_t *out;
    int len = 0;
    frame_type_t frametype;
    
    if (*p == '-')
        frametype = UAT_DOWNLINK;
    else if (*p == '+')
        frametype = UAT_UPLINK;
    else
        return 0;
    
    out = reader->frame;
    ++p;
    while (p < end) {
        int byte;
                
        if (p[0] == ';') {
            // ignore rest of line
            handler(frametype, reader->frame, len, handler_data);
            return 1;
        }
        
        if (len >= sizeof(reader->frame))
            return 0; // oversized frame
                
        byte = hexbyte(p);
        if (byte < 0)
            return 0; // badly formatted byte
                
        ++len;
        *out++ = byte;
        p += 2;
    }

    return 0; // ran off the end without seeing semicolon
}    

static int hexbyte(char *buf)
{
    int i;
    char c;

    c = buf[0];
    if (c >= '0' && c <= '9')
        i = (c - '0');
    else if (c >= 'a' && c <= 'f')
        i = (c - 'a' + 10);
    else if (c >= 'A' && c <= 'F')
        i = (c - 'A' + 10);
    else
        return -1;

    i <<= 4;
    c = buf[1];
    if (c >= '0' && c <= '9')
        return i | (c - '0');
    else if (c >= 'a' && c <= 'f')
        return i | (c - 'a' + 10);
    else if (c >= 'A' && c <= 'F')
        return i | (c - 'A' + 10);
    else
        return -1;
}
