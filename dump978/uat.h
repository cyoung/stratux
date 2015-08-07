// Part of dump978, a UAT decoder.
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

#ifndef UAT_H
#define UAT_H

// Frame size constants

#define SHORT_FRAME_DATA_BITS (144)
#define SHORT_FRAME_BITS (SHORT_FRAME_DATA_BITS+96)
#define SHORT_FRAME_DATA_BYTES (SHORT_FRAME_DATA_BITS/8)
#define SHORT_FRAME_BYTES (SHORT_FRAME_BITS/8)

#define LONG_FRAME_DATA_BITS (272)
#define LONG_FRAME_BITS (LONG_FRAME_DATA_BITS+112)
#define LONG_FRAME_DATA_BYTES (LONG_FRAME_DATA_BITS/8)
#define LONG_FRAME_BYTES (LONG_FRAME_BITS/8)

#define UPLINK_BLOCK_DATA_BITS (576)
#define UPLINK_BLOCK_BITS (UPLINK_BLOCK_DATA_BITS+160)
#define UPLINK_BLOCK_DATA_BYTES (UPLINK_BLOCK_DATA_BITS/8)
#define UPLINK_BLOCK_BYTES (UPLINK_BLOCK_BITS/8)

#define UPLINK_FRAME_BLOCKS (6)
#define UPLINK_FRAME_DATA_BITS (UPLINK_FRAME_BLOCKS * UPLINK_BLOCK_DATA_BITS)
#define UPLINK_FRAME_BITS (UPLINK_FRAME_BLOCKS * UPLINK_BLOCK_BITS)
#define UPLINK_FRAME_DATA_BYTES (UPLINK_FRAME_DATA_BITS/8)
#define UPLINK_FRAME_BYTES (UPLINK_FRAME_BITS/8)

#endif
