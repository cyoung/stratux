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
#include <math.h>

#include "uat.h"
#include "uat_decode.h"
#include "reader.h"

#define BLOCK_WIDTH (48.0/60.0)
#define WIDE_BLOCK_WIDTH (96.0/60.0)
#define BLOCK_HEIGHT (4.0/60.0)
#define BLOCK_THRESHOLD 405000
#define BLOCKS_PER_RING 450

//
// This reads demodulated uplink messages and extracts NEXRAD global block representation formats - type 63 and 64
//
// The output format is a series of lines, one line per decoded block.
// Each line is space-separated and is formatted as:
//
//  NEXRAD <type> <hour>:<minute> <scale> <north> <west> <height> <width> <data>
//
// where:
//   <type> is Regional (for type 63) or CONUS (for type 64)
//   <hour>:<minute> is the time from the PDU header - all blocks from one composite radar image will have the same time
//   <scale> is the scale value of this block - 0 (high res), 1 (med res), or 2 (low res)
//   <north> is the north edge of the block, in _integer arcminutes_. Divide by 60 to get degrees.
//   <west> is the west edge of the block, in _positive integer arcminutes_. Divide by 60 to get degrees; subtract 360 if you want the conventional -180..+180 range.
//   <height> is the height of the block, in integer arcminutes of latitude
//   <width> is the width of the block, in integer arcminutes of longitude
//
// Each block contains 128 evenly spaced bins, in a grid of 32 (longitude) x 4 (latitude), working west-to-east then north-to-south.
// i.e. each bin represents a pixel that covers <width>/32 arcminutes of longitude by <height>/4 arcminutes of latitude.
//
// <data> is a string of 128 digits (no spaces); each character represents the intensity of one bin, in the order above.
//

// Given:
//   bn - block number
//   ns - north/south flag
//   sf - scale factor
//
// compute the northwest corner of the referenced block and place it in (*latN, *lonW) 
// and place the total height and width of the block in (*latSize, *lonSize)
void block_location(int bn, int ns, int sf, double *latN, double *lonW, double *latSize, double *lonSize)
{
    // With SF=0:

    // blocks are (48 arcminutes longitude) x (4 arcminute latitude) between 0 and 60 degrees latitude
    //   (450 blocks for each ring of latitude)
    // blocks are (96 arcminutes longitude) x (4 arcminute latitude) between 60 and 90 degrees latitude
    //   (225 blocks for each ring of latitude) - but the block numbering continues to use
    //   a 48-arcminute spacing, so only even numbered blocks are meaningful.
    // block zero is immediately northeast of (0,0), then blocks are numbered east-to-west, south-to-north.
    //
    // Southern hemisphere numbering is mirrored around the equator, and indicated by the "ns" flag.

    //                             ^N
    // |   405446    |   405448    |   405000    |   405002    |
    // ---------------------------------------------------------  60 00' 00" N
    // |404996|404997|404998|404999|404550|404551|404552|404553|
    // ---------------------------------------------------------  59 56' 00" N
    //                            ...
    // | 896  | 897  | 898  | 899  | 450  | 451  | 452  | 453  |
    // ---------------------------------------------------------  00 04' 00" N
    // | 446  | 447  | 447  | 449  |  0   |  1   |  2   |  3   |
    //W<------------------------------------------------------->E equator
    // | 446* | 447* | 447* | 449* |  0*  |  1*  |  2*  |  3*  |
    // ---------------------------------------------------------  00 04' 00" S
    // | 896* | 897* | 898* | 899* | 450* | 451* | 452* | 453* |
    //      2d24'W 1d36'W 0d48'W   V    0d48'E 1d36'E 2d24'E
    // (* = ns_flag set)

    // Each block is subdivided into 32 (longitude) x 4 (latitude) bins.
    // The bins are numbered starting at the northwest corner of the block,
    // west-to-east then north-to-south.

    // block 0:
    //
    //    ------------------------------------  <- 0d04m00s N
    //    |  0  1  2  3  ...   28  29  30  31|  <- each bin is 1 arcminute tall
    //    | 32 33 34 35  ...   60  61  62  63|
    //    | 64 65 66 67  ...   92  93  94  95|
    //    | 96 97 98 99  ...  124 125 126 127|
    //    ------------------------------------  <- 0N - equator
    //    ^    ^ each bin is                 ^
    //    0E     1.5 arcminute wide       0d48m00s E

    // With SF=1, an identical block numbering is used to locate the northwest corner of the block,
    // but then each bin is 5x larger in both axes i.e. 240 x 20 or 480 x 20 arcminutes.
    // this means that the block data will actually overlap 24 other block positions.
    
    // With SF=2, it works like SF=1 but with a scale factor of 9x.

    double raw_lat, raw_lon;
    double scale;

    if (sf == 1)
        scale = 5.0;
    else if (sf == 2)
        scale = 9.0;
    else
        scale = 1.0;

    if (bn >= BLOCK_THRESHOLD) {
        // 60-90 degrees - even-numbered blocks only
        bn = bn & ~1;
    }

    raw_lat = BLOCK_HEIGHT * trunc(bn / BLOCKS_PER_RING);
    raw_lon = (bn % BLOCKS_PER_RING) * BLOCK_WIDTH;

    *lonSize = (bn >= BLOCK_THRESHOLD ? WIDE_BLOCK_WIDTH : BLOCK_WIDTH) * scale;
    *latSize = BLOCK_HEIGHT * scale;

    // raw_lat/raw_lon points to the southwest corner in the northern hemisphere version
    *lonW = raw_lon;
    if (ns) {
        // southern hemisphere, mirror along the equator
        *latN = 0 - raw_lat;
    } else {
        // adjust to the northwest corner
        *latN = raw_lat + BLOCK_HEIGHT;
    }
}

void decode_nexrad(struct fisb_apdu *fisb)
{
    // Header:
    //
    // byte/bit 7   6   5   4   3   2   1   0
    //   0    |RLE|NS | Scale |  MSB Block #  |
    //   1    |        Block #                |
    //   2    |        Block #            LSB |
    

    int rle_flag = (fisb->data[0] & 0x80) != 0;
    int ns_flag = (fisb->data[0] & 0x40) != 0;
    int block_num = ((fisb->data[0] & 0x0f) << 16) | (fisb->data[1] << 8) | (fisb->data[2]);
    int scale_factor = (fisb->data[0] & 0x30) >> 4;

    // now decode the bins
    if (rle_flag) {
        // One bin, 128 values, RLE-encoded
        int i;
        double latN = 0, lonW = 0, latSize = 0, lonSize = 0;
        block_location(block_num, ns_flag, scale_factor, &latN, &lonW, &latSize, &lonSize);

        fprintf(stdout, "NEXRAD %s %02d:%02d %d %.0f %.0f %.0f %.0f ",
                fisb->product_id == 63 ? "Regional" : "CONUS",
                fisb->hours,
                fisb->minutes,
                scale_factor,
                latN * 60,
                lonW * 60,
                latSize * 60,
                lonSize * 60);

        // each byte following the header is:
        //   7   6   5   4   3   2   1   0
        // |   runlength - 1   | intensity |

        for (i = 3; i < fisb->length; ++i) {
            int intensity = fisb->data[i] & 7;
            int runlength = (fisb->data[i] >> 3) + 1;

            while (runlength-- > 0)
                fprintf(stdout, "%d", intensity);
        }
        fprintf(stdout, "\n");
    } else {
        int L = fisb->data[3] & 15;
        int i;
        int row_start, row_offset, row_size;

        //
        // Empty block representation, representing one
        // or more blocks that are completely empty of
        // data.
        //
        //       7    6    5    4    3    2    1    0
        // 3   |b+4 |b+3 |b+2 |b+1 |    length (L)     |
        // 4   |b+12|b+11|b+10|b+9 |b+8 |b+7 |b+6 |b+5 |
        // ...
        // 3+L |b+8L-3            ...            b+8L+4|

        // The block number from the header is always
        // empty.
        //
        // If the bit for b+x is empty, then the block
        // X to the right of the block from the header is
        // empty. Note that the block is _always on the
        // same row_ even if the offset would make the
        // block cross the 0E meridian, so it is not simply
        // a case of adding to the block number.
        
        // find the lowest-numbered block of this row
        if (block_num >= 405000) {
            row_start = block_num - ((block_num - 405000) % 225);
            row_size = 225;
        } else {
            row_start = block_num - (block_num % 450);
            row_size = 450;
        }
        
        // find the offset of the first block in this row handled
        // by this message
        row_offset = block_num - row_start;

        for (i = 0; i < L; ++i) {
            int bb;
            int j;

            if (i == 0)
                bb = (fisb->data[3] & 0xF0) | 0x08; // synthesize a first byte in the same format as all the other bytes
            else
                bb = (fisb->data[i+3]);

            for (j = 0; j < 8; ++j) {
                if (bb & (1 << j)) {
                    // find the relevant block for this bit, limited
                    // to the same row as the original block.
                    int row_x = (row_offset + 8*i + j - 3) % row_size;
                    int bn = row_start + row_x;
                    double latN = 0, lonW = 0, latSize = 0, lonSize = 0;
                    int k;
                    block_location(bn, ns_flag, scale_factor, &latN, &lonW, &latSize, &lonSize);

                    fprintf(stdout, "NEXRAD %s %02d:%02d %d %.0f %.0f %.0f %.0f ",
                            fisb->product_id == 63 ? "Regional" : "CONUS",
                            fisb->hours,
                            fisb->minutes,
                            scale_factor,
                            latN * 60,
                            lonW * 60,
                            latSize * 60,
                            lonSize * 60);

                    // seems to work best if we assume that
                    // CONUS empty blocks = intensity 1 (valid data, but no precipitation)
                    // regional empty blocks = intensity 0 (valid data <5dBz)
                    for (k = 0; k < 128; ++k)
                        fprintf(stdout, "%d", (fisb->product_id == 63 ? 0 : 1));
                    fprintf(stdout, "\n");
                }
            }
        }
    }
}

void handle_frame(frame_type_t type, uint8_t *frame, int len, void *extra)
{
    if (type == UAT_UPLINK) {
        struct uat_uplink_mdb mdb;
        int i;

        uat_decode_uplink_mdb(frame, &mdb);
        if (!mdb.app_data_valid)
            return;

        for (i = 0; i < mdb.num_info_frames; ++i) {
            struct fisb_apdu *fisb;

            if (!mdb.info_frames[i].is_fisb)
                continue;

            fisb = &mdb.info_frames[i].fisb;
            if (fisb->product_id != 63 && fisb->product_id != 64)
                continue;

            decode_nexrad(fisb);
        }
    }

    fflush(stdout);
}        

int main(int argc, char **argv)
{
    struct dump978_reader *reader;
    int framecount;

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

