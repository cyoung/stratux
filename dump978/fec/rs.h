/* User include file for libfec
 * Copyright 2004, Phil Karn, KA9Q
 * May be used under the terms of the GNU Lesser General Public License (LGPL)
 */

#ifndef _FEC_RS_H_
#define _FEC_RS_H_

/* General purpose RS codec, 8-bit symbols */
int decode_rs_char(void *rs,unsigned char *data,int *eras_pos,
                   int no_eras);
void *init_rs_char(int symsize,int gfpoly,
                   int fcr,int prim,int nroots,
                   int pad);
void free_rs_char(void *rs);

#endif
