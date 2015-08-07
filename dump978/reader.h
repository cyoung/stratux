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

#ifndef DUMP978_READER_H
#define DUMP978_READER_H

#include <stdint.h>

struct dump978_reader;

typedef enum { UAT_UPLINK, UAT_DOWNLINK } frame_type_t;

// Function pointer type for a handler called by dump978_read_frames().
// It is called with arguments:
//   t: frame type (UAT_UPLINK or UAT_DOWNLINK)
//   f: pointer to frame data buffer
//   l: length of frame data
//   d: value of handler_data passed to dump978_read_frames
// The frame data buffer is a shared buffer owned by the caller
// and may be reused after return; if the handler needs to
// preserve the data after returning, it should take a copy.
typedef void (*frame_handler_t)(frame_type_t t,uint8_t *f,int l,void *d);

// Allocate a new reader that reads from file descriptor 'fd'.
// If 'nonblock' is nonzero, the FD will be made nonblocking.
// Returns the reader, or NULL on error with errno set.
struct dump978_reader *dump978_reader_new(int fd, int nonblock);

// Free a reader previously created by dump978_reader_new.
// Does not close the underlying file descriptor.
void dump978_reader_free(struct dump978_reader *reader);

// Read frames from the given reader.
// Pass complete frames to 'handler', passing 'handler_data'
// as the 4th argument.
//
// Returns a positive number of frames read on success.
// Returns 0 on EOF
// Returns <0 on error with errno set.
// If the underlying FD is nonblocking and no frames are
// available, returns <0 with errno = EAGAIN/EINTR/EWOULDBLOCK.
int dump978_read_frames(struct dump978_reader *reader,
                        frame_handler_t handler,
                        void *handler_data);

#endif


                         
