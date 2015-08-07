/* Initialize a RS codec
 *
 * Copyright 2002 Phil Karn, KA9Q
 * May be used under the terms of the GNU Lesser General Public License (LGPL)
 */
#include <stdlib.h>

#include "char.h"
#include "rs-common.h"

void free_rs_char(void *p){
  struct rs *rs = (struct rs *)p;

  free(rs->alpha_to);
  free(rs->index_of);
  free(rs->genpoly);
  free(rs);
}

/* Initialize a Reed-Solomon codec
 * symsize = symbol size, bits
 * gfpoly = Field generator polynomial coefficients
 * fcr = first root of RS code generator polynomial, index form
 * prim = primitive element to generate polynomial roots
 * nroots = RS code generator polynomial degree (number of roots)
 * pad = padding bytes at front of shortened block
 */
void *init_rs_char(int symsize,int gfpoly,int fcr,int prim,
	int nroots,int pad){
  struct rs *rs;

#include "init_rs.h"

  return rs;
}
