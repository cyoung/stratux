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
#include <stdint.h>
#include <string.h>
#include <math.h>
#include <unistd.h>

#include "uat.h"
#include "fec.h"

#ifdef BUILD_LIB
#include "dump978.h"
#endif

static void make_atan2_table(void);
#ifndef BUILD_LIB
static void read_from_stdin(void);
#endif
static int process_buffer(uint16_t *phi, uint16_t *raw, int len,
                          uint64_t offset);
static int demod_adsb_frame(uint16_t *phi, uint8_t *to, int *rs_errors);
static int demod_uplink_frame(uint16_t *phi, uint8_t *to, int *rs_errors);
static void demod_frame(uint16_t *phi, uint8_t *frame, int bytes,
                        int16_t center_dphi);
static void handle_adsb_frame(uint64_t timestamp, uint8_t *frame, int rs);
static void handle_uplink_frame(uint64_t timestamp, uint8_t *frame, int rs);

#define SYNC_BITS (36)
#define ADSB_SYNC_WORD 0xEACDDA4E2UL
#define UPLINK_SYNC_WORD 0x153225B1DUL

// relying on signed overflow is theoretically bad. Let's do it properly.

#ifdef USE_SIGNED_OVERFLOW
#define phi_difference(from, to) ((int16_t)((to) - (from)))
#else
inline int16_t phi_difference(uint16_t from, uint16_t to) {
  int32_t difference = to - from; // lies in the range -65535 .. +65535
  if (difference >= 32768)        //   +32768..+65535
    return difference - 65536;    //   -> -32768..-1: always in range
  else if (difference < -32768)   //   -65535..-32769
    return difference + 65536;    //   -> +1..32767: always in range
  else
    return difference;
}
#endif

#ifndef BUILD_LIB
int main(int argc, char **argv) {
  make_atan2_table();
  init_fec();
  read_from_stdin();
  return 0;
}
#else
static CallBack userCB = NULL;
void Dump978Init(CallBack cb) {
  make_atan2_table();
  init_fec();
  userCB = cb;
}
#endif

static int signal_strength = 0;

static void dump_raw_message(char updown, uint8_t *data, int len,
                             int rs_errors) {
#ifndef BUILD_LIB
  int i;

  fprintf(stdout, "%c", updown);
  for (i = 0; i < len; ++i) {
    fprintf(stdout, "%02x", data[i]);
  }

  if (rs_errors)
    fprintf(stdout, ";rs=%d", rs_errors);
  fprintf(stdout, ";ss=%d", signal_strength);
  fprintf(stdout, ";\n");
#else
  userCB(updown, data, len, rs_errors, signal_strength);
#endif
}

static void handle_adsb_frame(uint64_t timestamp, uint8_t *frame, int rs) {
  dump_raw_message('-', frame, (frame[0] >> 3) == 0 ? SHORT_FRAME_DATA_BYTES
                                                    : LONG_FRAME_DATA_BYTES,
                   rs);
  fflush(stdout);
}

static void handle_uplink_frame(uint64_t timestamp, uint8_t *frame, int rs) {
  dump_raw_message('+', frame, UPLINK_FRAME_DATA_BYTES, rs);
  fflush(stdout);
}

uint16_t iqphase[65536]; // contains value [0..65536) -> [0, 2*pi)

uint16_t iqamplitude[65536]; // contains value [0..65536) -> [0, 1000*sqrt(2))

void make_atan2_table(void) {
  unsigned i, q;
  union {
    uint8_t iq[2];
    uint16_t iq16;
  } u;

  for (i = 0; i < 256; ++i) {
    for (q = 0; q < 256; ++q) {
      double d_i = (i - 127.5);
      double d_q = (q - 127.5);
      double ang = atan2(d_q, d_i) +
                   M_PI; // atan2 returns [-pi..pi], normalize to [0..2*pi]
      double scaled_ang = round(32768 * ang / M_PI);

      double amp = sqrt(d_i * d_i + d_q * d_q);
      uint16_t scaled_amp = amp * 1000.0 / 127.5;

      u.iq[0] = i;
      u.iq[1] = q;
      iqphase[u.iq16] =
          (scaled_ang < 0 ? 0 : scaled_ang > 65535 ? 65535
                                                   : (uint16_t)scaled_ang);
      iqamplitude[u.iq16] = scaled_amp;
    }
  }
}

static void convert_to_phi(uint16_t *dest, uint16_t *src, int n) {
    int i;

    // unroll the loop. n is always > 2048, usually 36864
    for (i = 0; i+8 <= n; i += 8) {
        dest[i] = iqphase[src[i]];
        dest[i+1] = iqphase[src[i+1]];
        dest[i+2] = iqphase[src[i+2]];
        dest[i+3] = iqphase[src[i+3]];
        dest[i+4] = iqphase[src[i+4]];
        dest[i+5] = iqphase[src[i+5]];
        dest[i+6] = iqphase[src[i+6]];
        dest[i+7] = iqphase[src[i+7]];
    }
    for (; i < n; ++i)
        dest[i] = iqphase[src[i]];
}

static void calc_power(uint16_t *samples, int len) { // sets signal_strength to scaled amplitude. 0 = no signal, 1000 = saturated receiver on all samples in measurement.
  long avg = 0;
  int n = len;
  while (n--) {
    avg += iqamplitude[*samples++];
  }
  signal_strength = avg / len;
}

#ifndef BUILD_LIB
void read_from_stdin(void) {
  char buffer[65536 * 2];
  uint16_t phi[65536];

  int n;
  int used = 0;
  uint64_t offset = 0;

  while ((n = read(0, buffer + used, sizeof(buffer) - used)) > 0) {
    int processed;

    convert_to_phi(phi + used / 2, (uint16_t *)(buffer + (used & ~1)),
                   ((used & 1) + n) / 2);

    used += n;
    processed = process_buffer(phi, (uint16_t *)buffer, used / 2, offset);
    used -= processed * 2;
    offset += processed;
    if (used > 0) {
      memmove(buffer, buffer + processed * 2, used);
      memmove(phi, phi + processed, used);
    }
  }
}
#else
// #define DEFAULT_SAMPLE_RATE       2048000
// #define DEFAULT_BUF_LENGTH      (262144) 16*16384
static char buffer[65536 * 2]; // 131072, max received should be 113120
static uint16_t phi[65536];
int process_data(char *data, int dlen) {
  int n;
  int processed;
  int doffset = 0;
  static int used = 0;
  static uint64_t offset = 0;

  while (dlen > 0) {
    n = (sizeof(buffer) - used) >= dlen ? dlen : (sizeof(buffer) - used);
    memcpy(buffer + used, data + doffset, n);

    convert_to_phi(phi + used / 2, (uint16_t *)(buffer + (used & ~1)),
                   ((used & 1) + n) / 2);

    used += n;
    processed = process_buffer(phi, (uint16_t *)buffer, used / 2, offset);
    used -= processed * 2;
    offset += processed;
    if (used > 0) {
      memmove(buffer, buffer + processed * 2, used);
      memmove(phi, phi + processed, used);
    }

    doffset += n;
    dlen -= n;
  }
  return dlen;
}
#endif

// Return 1 if word is "equal enough" to expected
static inline int sync_word_fuzzy_compare(uint64_t word, uint64_t expected) {
  uint64_t diff;

  if (word == expected)
    return 1;

  diff = word ^ expected; // guaranteed nonzero

  // This is a bit-twiddling popcount
  // hack, tweaked as we only care about
  // "<N" or ">=N" set bits for fixed N -
  // so we can bail out early after seeing N
  // set bits.
  //
  // It relies on starting with a nonzero value
  // with zero or more trailing clear bits
  // after the last set bit:
  //
  //    010101010101010000
  //                 ^
  // Subtracting one, will flip the
  // bits starting at the last set bit:
  //
  //    010101010101001111
  //                 ^
  // then we can use that as a bitwise-and
  // mask to clear the lowest set bit:
  //
  //    010101010101000000
  //                 ^
  // And repeat until the value is zero
  // or we have seen too many set bits.

  // >= 1 bit
  diff &= (diff - 1); // clear lowest set bit
  if (!diff)
    return 1; // 1 bit error

  // >= 2 bits
  diff &= (diff - 1); // clear lowest set bit
  if (!diff)
    return 1; // 2 bits error

  // >= 3 bits
  diff &= (diff - 1); // clear lowest set bit
  if (!diff)
    return 1; // 3 bits error

  // >= 4 bits
  diff &= (diff - 1); // clear lowest set bit
  if (!diff)
    return 1; // 4 bits error

  // > 4 bits in error, give up
  return 0;
}

#define MAX_SYNC_ERRORS 4

// check that there is a valid sync word starting at 'phi'
// that matches the sync word 'pattern'. Place the dphi
// threshold to use for bit slicing in '*center'. Return 1
// if the sync word is OK, 0 on failure
int check_sync_word(uint16_t *phi, uint64_t pattern, int16_t *center) {
  int i;
  int32_t dphi_zero_total = 0;
  int zero_bits = 0;
  int32_t dphi_one_total = 0;
  int one_bits = 0;
  int error_bits;

  // find mean dphi for zero and one bits;
  // take the mean of the two as our central value

  for (i = 0; i < SYNC_BITS; ++i) {
    int16_t dphi = phi_difference(phi[i * 2], phi[i * 2 + 1]);

    if (pattern & (1UL << (35 - i))) {
      ++one_bits;
      dphi_one_total += dphi;
    } else {
      ++zero_bits;
      dphi_zero_total += dphi;
    }
  }

  dphi_zero_total /= zero_bits;
  dphi_one_total /= one_bits;

  *center = (dphi_one_total + dphi_zero_total) / 2;

  // recheck sync word using our center value
  error_bits = 0;
  for (i = 0; i < SYNC_BITS; ++i) {
    int16_t dphi = phi_difference(phi[i * 2], phi[i * 2 + 1]);

    if (pattern & (1UL << (35 - i))) {
      if (dphi < *center)
        ++error_bits;
    } else {
      if (dphi >= *center)
        ++error_bits;
    }
  }

  // fprintf(stdout, "check_sync_word: center=%.0fkHz, errors=%d\n", *center *
  // 2083334.0 / 65536 / 1000, error_bits);

  return (error_bits <= MAX_SYNC_ERRORS);
}

#define SYNC_MASK ((((uint64_t)1) << SYNC_BITS) - 1)

int process_buffer(uint16_t *phi, uint16_t *raw, int len, uint64_t offset) {
  uint64_t sync0 = 0, sync1 = 0;
  int lenbits;
  int bit;

  uint8_t demod_buf_a[UPLINK_FRAME_BYTES];
  uint8_t demod_buf_b[UPLINK_FRAME_BYTES];

  // We expect samples at twice the UAT bitrate.
  // We look at phase difference between pairs of adjacent samples, i.e.
  //  sample 1 - sample 0   -> sync0
  //  sample 2 - sample 1   -> sync1
  //  sample 3 - sample 2   -> sync0
  //  sample 4 - sample 3   -> sync1
  // ...
  //
  // We accumulate bits into two buffers, sync0 and sync1.
  // Then we compare those buffers to the expected 36-bit sync word that
  // should be at the start of each UAT frame. When (if) we find it,
  // that tells us which sample to start decoding from.

  // Stop when we run out of remaining samples for a max-sized frame.
  // Arrange for our caller to pass the trailing data back to us next time;
  // ensure we don't consume any partial sync word we might be part-way
  // through. This means we don't need to maintain state between calls.

  lenbits = len / 2 - (SYNC_BITS + UPLINK_FRAME_BITS);
  for (bit = 0; bit < lenbits; ++bit) {
    int16_t dphi0 = phi_difference(phi[bit * 2], phi[bit * 2 + 1]);
    int16_t dphi1 = phi_difference(phi[bit * 2 + 1], phi[bit * 2 + 2]);

    sync0 = ((sync0 << 1) | (dphi0 > 0 ? 1 : 0)) & SYNC_MASK;
    sync1 = ((sync1 << 1) | (dphi1 > 0 ? 1 : 0)) & SYNC_MASK;

    if (bit < SYNC_BITS)
      continue; // haven't fully populated sync0/1 yet

    // see if we have (the start of) a valid sync word
    // It would be nice to look at popcount(expected ^ sync)
    // so we can tolerate some errors, but that turns out
    // to be very expensive to do on every sample

    // when we find a match, try to demodulate both with that match
    // and with the next position, and pick the one with fewer
    // errors.

    // check for downlink frames:
    if (sync_word_fuzzy_compare(sync0, ADSB_SYNC_WORD) ||
        sync_word_fuzzy_compare(sync1, ADSB_SYNC_WORD)) {
      int startbit = (bit - SYNC_BITS + 1);
      int shift = (sync_word_fuzzy_compare(sync0, ADSB_SYNC_WORD) ? 0 : 1);
      int index = startbit * 2 + shift;

      int skip_0, skip_1;
      int rs_0 = -1, rs_1 = -1;

      skip_0 = demod_adsb_frame(phi + index, demod_buf_a, &rs_0);
      skip_1 = demod_adsb_frame(phi + index + 1, demod_buf_b, &rs_1);
      if (skip_0 && rs_0 <= rs_1) {
        calc_power(raw + index, skip_0 * 2);
        handle_adsb_frame(offset + index, demod_buf_a, rs_0);
        bit = startbit + skip_0;
        continue;
      } else if (skip_1 && rs_1 <= rs_0) {
        calc_power(raw + index + 1, skip_1 * 2);
        handle_adsb_frame(offset + index + 1, demod_buf_b, rs_1);
        bit = startbit + skip_1;
        continue;
      } else {
        // demod failed
      }
    }

    // check for uplink frames:
    else if (sync_word_fuzzy_compare(sync0, UPLINK_SYNC_WORD) ||
             sync_word_fuzzy_compare(sync1, UPLINK_SYNC_WORD)) {
      int startbit = (bit - SYNC_BITS + 1);
      int shift = (sync_word_fuzzy_compare(sync0, UPLINK_SYNC_WORD) ? 0 : 1);
      int index = startbit * 2 + shift;

      int skip_0, skip_1;
      int rs_0 = -1, rs_1 = -1;

      skip_0 = demod_uplink_frame(phi + index, demod_buf_a, &rs_0);
      skip_1 = demod_uplink_frame(phi + index + 1, demod_buf_b, &rs_1);
      if (skip_0 && rs_0 <= rs_1) {
        calc_power(raw + index, skip_0 * 2);
        handle_uplink_frame(offset + index, demod_buf_a, rs_0);
        bit = startbit + skip_0;
        continue;
      } else if (skip_1 && rs_1 <= rs_0) {
        calc_power(raw + index, skip_1 * 2);
        handle_uplink_frame(offset + index + 1, demod_buf_b, rs_1);
        bit = startbit + skip_1;
        continue;
      } else {
        // demod failed
      }
    }
  }

  return (bit - SYNC_BITS) * 2;
}

// demodulate 'bytes' bytes from samples at 'phi' into 'frame',
// using 'center_dphi' as the bit slicing threshold
static void demod_frame(uint16_t *phi, uint8_t *frame, int bytes,
                        int16_t center_dphi) {
  while (--bytes >= 0) {
    uint8_t b = 0;
    if (phi_difference(phi[0], phi[1]) > center_dphi)
      b |= 0x80;
    if (phi_difference(phi[2], phi[3]) > center_dphi)
      b |= 0x40;
    if (phi_difference(phi[4], phi[5]) > center_dphi)
      b |= 0x20;
    if (phi_difference(phi[6], phi[7]) > center_dphi)
      b |= 0x10;
    if (phi_difference(phi[8], phi[9]) > center_dphi)
      b |= 0x08;
    if (phi_difference(phi[10], phi[11]) > center_dphi)
      b |= 0x04;
    if (phi_difference(phi[12], phi[13]) > center_dphi)
      b |= 0x02;
    if (phi_difference(phi[14], phi[15]) > center_dphi)
      b |= 0x01;
    *frame++ = b;
    phi += 16;
  }
}

// Demodulate an ADSB (Long UAT or Basic UAT) downlink frame
// with the first sync bit in 'phi', storing the frame into 'to'
// of length up to LONG_FRAME_BYTES. Set '*rs_errors' to the
// number of corrected errors, or 9999 if demodulation failed.
// Return 0 if demodulation failed, or the number of bits (not
// samples) consumed if demodulation was OK.
static int demod_adsb_frame(uint16_t *phi, uint8_t *to, int *rs_errors) {
  int16_t center_dphi;
  int frametype;

  if (!check_sync_word(phi, ADSB_SYNC_WORD, &center_dphi)) {
    *rs_errors = 9999;
    return 0;
  }

  demod_frame(phi + SYNC_BITS * 2, to, LONG_FRAME_BYTES, center_dphi);
  frametype = correct_adsb_frame(to, rs_errors);
  if (frametype == 1)
    return (SYNC_BITS + SHORT_FRAME_BITS);
  else if (frametype == 2)
    return (SYNC_BITS + LONG_FRAME_BITS);
  else
    return 0;
}

// Demodulate an uplink frame
// with the first sync bit in 'phi', storing the frame into 'to'
// of length up to UPLINK_FRAME_BYTES. Set '*rs_errors' to the
// number of corrected errors, or 9999 if demodulation failed.
// Return 0 if demodulation failed, or the number of bits (not
// samples) consumed if demodulation was OK.
static int demod_uplink_frame(uint16_t *phi, uint8_t *to, int *rs_errors) {
  int16_t center_dphi;
  uint8_t interleaved[UPLINK_FRAME_BYTES];

  if (!check_sync_word(phi, UPLINK_SYNC_WORD, &center_dphi)) {
    *rs_errors = 9999;
    return 0;
  }

  demod_frame(phi + SYNC_BITS * 2, interleaved, UPLINK_FRAME_BYTES,
              center_dphi);

  // deinterleave and correct
  if (correct_uplink_frame(interleaved, to, rs_errors) == 1)
    return (UPLINK_FRAME_BITS + SYNC_BITS);
  else
    return 0;
}