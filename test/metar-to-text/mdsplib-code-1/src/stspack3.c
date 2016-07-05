/*
METAR Decoder Software Package Library: Parses Aviation Routine Weather Reports
Copyright (C) 2003  Eric McCarthy

This library is free software; you can redistribute it and/or
modify it under the terms of the GNU Lesser General Public
License as published by the Free Software Foundation; either
version 2.1 of the License, or (at your option) any later version.

This library is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public
License along with this library; if not, write to the Free Software
Foundation, Inc., 59 Temple Place, Suite 330, Boston, MA  02111-1307  USA
*/

#pragma comment (compiler)
#pragma comment (date)
#pragma comment (timestamp)
#pragma pagesize(80)
 
#include "local.h"     /* standard header file */
 
#pragma page(1)
#pragma subtitle(" ")
#pragma subtitle("stspack3 - Local string test functions       ")
/********************************************************************/
/*                                                                  */
/*  Title:         stspack3                                         */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          05 Oct 1992                                      */
/*  Programmer:    ALLAN DARLING                                    */
/*  Language:      C/2                                              */
/*                                                                  */
/*  Abstract:      The stspack3 package contains functions to       */
/*                 perform the isalnum through isxdigit functions   */
/*                 on strings.  The functions come in four forms:   */
/*                 those that test NULL delimited strings and are   */
/*                 named in the form sxxxxxxx, those that test at   */
/*                 most n characters and are named in the form      */
/*                 nxxxxxxx, those that search forward in a string  */
/*                 and are named in the form nxtyyyyy, and those    */
/*                 that search backward in a string and are named   */
/*                 in the form lstyyyyy.                            */
/*                                                                  */
/*                 The xxxxxxx is the name of the test applied to   */
/*                 each character in the string, such as isalpha,   */
/*                 thus a function to test a NULL delimited string  */
/*                 an return a nonzero value if all characters in   */
/*                 the string are digits is named sisdigit.         */
/*                                                                  */
/*                 The yyyyy is the name of the test applied to     */
/*                 characters in a string, minus the 'is' prefix.   */
/*                 Thus a function to find the next digit in a NULL */
/*                 delimited string and return a pointer to it is   */
/*                 named nxtdigit.                                  */
/*                                                                  */
/*                 The only exception to the naming rule is for the */
/*                 functions that test for hexadecimal digits.      */
/*                 These are named sisxdigi, nisxdigi, nxtxdigi,    */
/*                 and lstxdigi because of the eight character      */
/*                 function name limitation.                        */
/*                                                                  */
/*                 The nxxxxxxx class of functions will test up to  */
/*                 n characters or the first NULL character         */
/*                 encountered, whichever comes first.  For all     */
/*                 classes of functions, the string sentinal is     */
/*                 not included in the test.                        */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 isalnum, isalpha, iscntrl, isdigit, isgraph,     */
/*                 islower, isprint, ispunct, isspace, isupper,     */
/*                 isxdigit.                                        */
/*                                                                  */
/*  Input:         For sxxxxxxx class functions, a pointer to a     */
/*                 NULL delimited character string.                 */
/*                                                                  */
/*                 For nxtyyyyy class functions, a pointer to a     */
/*                 NULL delimited character string.                 */
/*                                                                  */
/*                 for nxxxxxxx class functions, a pointer to a     */
/*                 character array, and a positive, nonzero integer.*/
/*                                                                  */
/*                 for lstyyyyy class functions, a pointer to a     */
/*                 character array, and a positive, nonzero integer.*/
/*                                                                  */
/*  Output:        A nonzero value if the test is true for all      */
/*                 characters in the string, a zero value otherwise.*/
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
char *nxtalnum(char *s) {
 
   for (; !isalnum(*s) && *s; s++) ;
 
   if (*s)
      return (s);
   else
      return (NULL);
 
} /* end nxtalnum */
 
 
char *nxtalpha(char *s) {
 
   for (; !isalpha(*s) && *s; s++) ;
 
   if (*s)
      return (s);
   else
      return (NULL);
 
} /* end nxtalpha */
 
 
char *nxtcntrl(char *s) {
 
   for (; !iscntrl(*s) && *s; s++) ;
 
   if (*s)
      return (s);
   else
      return (NULL);
 
} /* end nxtcntrl */
 
 
char *nxtdigit(char *s) {
 
   for (; !isdigit(*s) && *s; s++) ;
 
   if (*s)
      return (s);
   else
      return (NULL);
 
} /* end nxtdigit */
 
 
char *nxtgraph(char *s) {
 
   for (; !isgraph(*s) && *s; s++) ;
 
   if (*s)
      return (s);
   else
      return (NULL);
 
} /* end nxtgraph */
 
 
char *nxtlower(char *s) {
 
   for (; !islower(*s) && *s; s++) ;
 
   if (*s)
      return (s);
   else
      return (NULL);
 
} /* end nxtlower */
 
 
char *nxtprint(char *s) {
 
   for (; !isprint(*s) && *s; s++) ;
 
   if (*s)
      return (s);
   else
      return (NULL);
 
} /* end nxtprint */
 
 
char *nxtpunct(char *s) {
 
   for (; !ispunct(*s) && *s; s++) ;
 
   if (*s)
      return (s);
   else
      return (NULL);
 
} /* end nxtpunct */
 
 
char *nxtspace(char *s) {
 
   for (; !isspace(*s) && *s; s++) ;
 
   if (*s)
      return (s);
   else
      return (NULL);
 
} /* end nxtspace */
 
 
char *nxtupper(char *s) {
 
   for (; !isupper(*s) && *s; s++) ;
 
   if (*s)
      return (s);
   else
      return (NULL);
 
} /* end nxtupper */
 
 
char *nxtxdigi(char *s) {
 
   for (; !isxdigit(*s) && *s; s++) ;
 
   if (*s)
      return (s);
   else
      return (NULL);
 
} /* end nxtxdigi */
 
 
#pragma page(1)
