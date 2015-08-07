/* Common code for intializing a Reed-Solomon control block (char or int symbols)
 * Copyright 2004 Phil Karn, KA9Q
 * May be used under the terms of the GNU Lesser General Public License (LGPL)
 */
#undef NULL
#define NULL ((void *)0)

{
  int i, j, sr,root,iprim;

  rs = NULL;
  /* Check parameter ranges */
  if(symsize < 0 || symsize > 8*sizeof(data_t)){
    goto done;
  }

  if(fcr < 0 || fcr >= (1<<symsize))
    goto done;
  if(prim <= 0 || prim >= (1<<symsize))
    goto done;
  if(nroots < 0 || nroots >= (1<<symsize))
    goto done; /* Can't have more roots than symbol values! */
  if(pad < 0 || pad >= ((1<<symsize) -1 - nroots))
    goto done; /* Too much padding */

  rs = (struct rs *)calloc(1,sizeof(struct rs));
  if(rs == NULL)
    goto done;

  rs->mm = symsize;
  rs->nn = (1<<symsize)-1;
  rs->pad = pad;

  rs->alpha_to = (data_t *)malloc(sizeof(data_t)*(rs->nn+1));
  if(rs->alpha_to == NULL){
    free(rs);
    rs = NULL;
    goto done;
  }
  rs->index_of = (data_t *)malloc(sizeof(data_t)*(rs->nn+1));
  if(rs->index_of == NULL){
    free(rs->alpha_to);
    free(rs);
    rs = NULL;
    goto done;
  }

  /* Generate Galois field lookup tables */
  rs->index_of[0] = A0; /* log(zero) = -inf */
  rs->alpha_to[A0] = 0; /* alpha**-inf = 0 */
  sr = 1;
  for(i=0;i<rs->nn;i++){
    rs->index_of[sr] = i;
    rs->alpha_to[i] = sr;
    sr <<= 1;
    if(sr & (1<<symsize))
      sr ^= gfpoly;
    sr &= rs->nn;
  }
  if(sr != 1){
    /* field generator polynomial is not primitive! */
    free(rs->alpha_to);
    free(rs->index_of);
    free(rs);
    rs = NULL;
    goto done;
  }

  /* Form RS code generator polynomial from its roots */
  rs->genpoly = (data_t *)malloc(sizeof(data_t)*(nroots+1));
  if(rs->genpoly == NULL){
    free(rs->alpha_to);
    free(rs->index_of);
    free(rs);
    rs = NULL;
    goto done;
  }
  rs->fcr = fcr;
  rs->prim = prim;
  rs->nroots = nroots;

  /* Find prim-th root of 1, used in decoding */
  for(iprim=1;(iprim % prim) != 0;iprim += rs->nn)
    ;
  rs->iprim = iprim / prim;

  rs->genpoly[0] = 1;
  for (i = 0,root=fcr*prim; i < nroots; i++,root += prim) {
    rs->genpoly[i+1] = 1;

    /* Multiply rs->genpoly[] by  @**(root + x) */
    for (j = i; j > 0; j--){
      if (rs->genpoly[j] != 0)
	rs->genpoly[j] = rs->genpoly[j-1] ^ rs->alpha_to[modnn(rs,rs->index_of[rs->genpoly[j]] + root)];
      else
	rs->genpoly[j] = rs->genpoly[j-1];
    }
    /* rs->genpoly[0] can never be zero */
    rs->genpoly[0] = rs->alpha_to[modnn(rs,rs->index_of[rs->genpoly[0]] + root)];
  }
  /* convert rs->genpoly[] to index form for quicker encoding */
  for (i = 0; i <= nroots; i++)
    rs->genpoly[i] = rs->index_of[rs->genpoly[i]];
 done:;

}
