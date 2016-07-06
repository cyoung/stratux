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
 
#include "local.h"    /* standard header file */
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("charcmp - characters compare with patterns  ")
/********************************************************************/
/*                                                                  */
/*  Title:         charcmp                                          */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          12 Dec 1995                                      */
/*  Programmer:    CINDY L. CHONG                                   */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:      This function will compare each character in the */
/*                 string match with each character in the pattern  */
/*                 which is made up of characters.   The str can    */
/*                 be longer than the pattern.                      */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                                                                  */
/*  Input:         str is a pointer to char                         */
/*                 pattern is a pointer to char                     */
/*                                                                  */
/*  Output:        Return true if str matches pattern,              */
/*                 otherwise, return false                          */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
MDSP_BOOL charcmp(char *str, char *pattern)
{
 
 
   /**********************************************************/
   /* Loop while str and pattern is not equal to null, then  */
   /* inscreases str and pattern by one                      */
   /**********************************************************/
 
   for (; *pattern != '\0'; pattern++)
   {
      if (*str == '\0')
         return FALSE;
 
      /************************************************************/
      /* If pattern match str, then increase str and jump out the */
      /* case and read next char of the str and pattern           */
      /************************************************************/
 
      if ( isspace(*pattern) )
         pattern = nxtalpha(pattern);
 
      switch( *pattern )
      {
         case 'c':
            if ( !isalnum(*str++) )
            {
               return FALSE;
            }
            break;
 
         case 'a':
            if ( !isalpha(*str) )
            {
               return FALSE;
            }
            str++;
            break;
 
         case 'n':
            if ( !iscntrl(*str++) )
            {
               return FALSE;
            }
            break;
 
         case 'd':
            if ( !isdigit(*str) )
            {
               return FALSE;
            }
            str++;
            break;
 
         case 'g':
            if ( !isgraph(*str++) )
            {
               return FALSE;
            }
            break;
 
         case 'i':
            if ( !islower(*str++) )
            {
               return FALSE;
            }
            break;
 
         case 'p':
            if ( !isprint(*str++) )
            {
               return FALSE;
            }
            break;
 
         case 't':
            if ( !ispunct(*str++) )
            {
               return FALSE;
            }
            break;
 
         case 'w':
            if ( !isspace(*str++) )
            {
               return FALSE;
            }
            break;
 
         case 'u':
            if ( !isupper(*str++) )
            {
               return FALSE;
            }
            break;
 
         case 's':
            if (*str++ != ' ')
            {
               return FALSE;
            }
            break;
 
         case 'm':
            if ( !isspace(*str) )
            {
               return FALSE;
            }
            else
            {
               while ( isspace(*str) )
                  str++;
            }
            break;
 
         case '\'':
            pattern++;
            if (*pattern != *str)
            {
               return FALSE;
            }
            pattern++;
            str++;
            break;
 
         default:
            return FALSE;
 
      } /* end switch */
 
   } /* end for */
 
   return (TRUE);
}
