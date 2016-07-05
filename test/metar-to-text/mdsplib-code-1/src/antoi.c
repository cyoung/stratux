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

#pragma comment(compiler)
#pragma comment(date)
#pragma comment(timestamp)
 
#include <stdlib.h>
 
#pragma title("antoi - char array to integer")
#pragma pagesize (80)
 
#pragma page(1)
/********************************************************************/
/*                                                                  */
/*  Title:         antoi                                            */
/*  Date:          Jan 28, 1991                                     */
/*  Organization:  W/OSO242 - Graphics and Display Section          */
/*  Programmer:    Allan Darling                                    */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:      This function will convert a character array     */
/*                 (string) of length (len) into an integer.        */
/*                 The integer is created via a call to the         */
/*                 function atoi.  This function extends the        */
/*                 functionality of atoi by removing the            */
/*                 requirement for a sentinal delimited string      */
/*                 as input.                                        */
/*                                                                  */
/*  Input: - Pointer to an array of characters.                     */
/*         - Integer indicating the number of character to include  */
/*           in the conversion.                                     */
/*                                                                  */
/*  Output:- An integer corresponding to the value in the character */
/*           array or MAXNEG (-2147483648) if the function is       */
/*           unable to acquire system storage.                      */
/*                                                                  */
/*  Modification History:                                           */
/*                 None                                             */
/*                                                                  */
/********************************************************************/
 
int antoi(char * string, int len)
{
 
    /*******************/
    /* local variables */
    /*******************/
 
    char * tmpstr;
    int i,
        retval;
 
 
    /*****************/
    /* function body */
    /*****************/
 
    tmpstr = malloc((len+1) * sizeof(char));
 
    if (tmpstr == NULL) return (-2147483648);
 
    for (i = 0; i < len; i++)
       tmpstr[i] = string[i];
 
    tmpstr[len] = '\0';
 
    retval = atoi(tmpstr);
 
    free(tmpstr);
 
    return(retval);
 
} /* end antoi */
 
#pragma page(1)
