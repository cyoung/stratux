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

#ifndef DUMP978_FEC_H
#define DUMP978_FEC_H

/* Initialize. Must be called once before correct_* */
void init_fec(void);

/* Correct a downlink frame.
 *
 * 'to' should contain LONG_FRAME_BYTES of data.
 * Errors are corrected in-place within 'to'.
 * Returns -1 on uncorrectable errors, 1 for a valid basic frame, 2 for a valid long frame.
 * Sets *rs_errors to the number of corrected errors, or 9999 if uncorrectable.
 */
int correct_adsb_frame(uint8_t *to, int *rs_errors);

/* Deinterleave and correct an uplink frame.
 *
 * 'from' should point to UPLINK_FRAME_BYTES of interleaved input data
 * 'to' should point to UPLINK_FRAME_BYTES of space for output data
 *   (only the first UPLINK_FRAME_DATA_BYTES will contain useful data)
 * Blocks are deinterleaved and corrected, and written to 'to'.
 * Returns -1 on uncorrectable errors, 1 for a valid uplink frame.
 * Sets *rs_errors to the number of corrected errors, or 9999 if uncorrectable.
 */
int correct_uplink_frame(uint8_t *from, uint8_t *to, int *rs_errors);

#endif
