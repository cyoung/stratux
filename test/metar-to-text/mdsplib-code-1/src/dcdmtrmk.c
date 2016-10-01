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

#include "metar_structs.h"
 
#define SKY1_len 50
float fracPart( char * );
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isTS_LOC                                         */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          06 May 1996                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:   Identify the input character string as a thunder-   */
/*              storm location.  If the input string is a thunder-  */
/*              storm location, then return TRUE.  Otherwise,       */
/*              return FALSE.                                       */
/*                                                                  */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                                                                  */
/*  Input:         string    - a pointer to a pointer to a charac-  */
/*                             ter string from a METAR report.      */
/*                                                                  */
/*                 Mptr - a pointer to a structure that has the     */
/*                        data type Decoded_METAR.                  */
/*                                                                  */
/*                 NDEX - a pointer to an integer that is the index */
/*                        into an array that contains the indi-     */
/*                        vidual groups of the METAR report being   */
/*                        decoded.  Upon entry, NDEX is the index   */
/*                        of the current group of the METAR report  */
/*                        that is to be indentified.                */
/*                                                                  */
/*  Output:        TRUE - if the input string is a thunderstorm     */
/*                        location.                                 */
/*                 FALSE - the input string is not a thunderstorm   */
/*                         location.                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
static MDSP_BOOL isTS_LOC( char **string, Decoded_METAR *Mptr,
                           int *NDEX )
{
 
   /***************************/
   /* DECLARE LOCAL VARIABLES */
   /***************************/
 
   int i;
 
   /*************************/
   /* START BODY OF ROUTINE */
   /*************************/
   /*******************************************/
   /* COMPARE THE INPUT CHARACTER STRING WITH */
   /* VALID AUTOMATED STATION CODE TYPE.  IF  */
   /* A MATCH IS FOUND, RETURN TRUE.  OTHER-  */
   /*           WISE, RETURN FALSE            */
   /*******************************************/
 
   if( *string == NULL )
      return FALSE;
 
   i = 0;
 
   if( strcmp( *string, "TS") != 0 )
      return FALSE;
   else {
      string++;
 
      if( *string == NULL )
         return FALSE;
 
      if(    strcmp(*string,"N")  == 0  ||
             strcmp(*string,"NE") == 0  ||
             strcmp(*string,"NW") == 0  ||
             strcmp(*string,"S")  == 0  ||
             strcmp(*string,"SE") == 0  ||
             strcmp(*string,"SW") == 0  ||
             strcmp(*string,"E")  == 0  ||
             strcmp(*string,"W")  == 0   ) {
         strcpy( Mptr->TS_LOC, *string );
         (*NDEX)++;
         (*NDEX)++;
         string++;
 
         if( *string == NULL )
            return TRUE;
 
         if( strcmp( *string, "MOV" ) == 0 ) {
            string++;
 
            if( *string == NULL ) {
               (*NDEX)++;
               return TRUE;
            }
 
            if(    strcmp(*string,"N")  == 0  ||
                   strcmp(*string,"NE") == 0  ||
                   strcmp(*string,"NW") == 0  ||
                   strcmp(*string,"S")  == 0  ||
                   strcmp(*string,"SE") == 0  ||
                   strcmp(*string,"SW") == 0  ||
                   strcmp(*string,"E")  == 0  ||
                   strcmp(*string,"W")  == 0   ) {
               strcpy( Mptr->TS_MOVMNT, *string );
               (*NDEX)++;
               (*NDEX)++;
               string++;
               return TRUE;
            }
         }
         else
            return TRUE;
 
      }
      else
         return FALSE;
 
   }
 
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isDVR                                            */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          15 Sep 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:                                                       */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                                                                  */
/*  Input:         x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
static MDSP_BOOL isDVR( char *token, Decoded_METAR *Mptr, int *NDEX )
{
   char *slashPtr, *FT_ptr;
   char *vPtr;
   int length;
 
   if( token == NULL )
      return FALSE;
 
   if( (length = strlen( token )) < 4 )
      return FALSE;
 
   if( strncmp( token, "DVR", 3 ) != 0 )
      return FALSE;
 
   if( *(slashPtr = token+3) != '/' ) {
      (*NDEX)++;
      return FALSE;
   }
 
   if( strcmp(token+(strlen(token)-2),"FT") != 0 )
      return FALSE;
   else
      FT_ptr = token + (strlen(token)-2);
 
   if( strchr(slashPtr+1, 'P' ) != NULL )
      Mptr->DVR.above_max_DVR = TRUE;
 
   if( strchr(slashPtr+1, 'M' ) != NULL )
      Mptr->DVR.below_min_DVR = TRUE;
 
 
   if( (vPtr = strchr(slashPtr, 'V' )) != NULL )
   {
      Mptr->DVR.vrbl_visRange = TRUE;
      Mptr->DVR.Min_visRange = antoi(slashPtr+1,
                              (vPtr-(slashPtr+1)) );
      Mptr->DVR.Max_visRange = antoi(vPtr+1,
                              (FT_ptr - (vPtr+1)) );
      (*NDEX)++;
      return TRUE;
   }
   else
   {
      if( Mptr->DVR.below_min_DVR ||
          Mptr->DVR.above_max_DVR    )
         Mptr->DVR.visRange = antoi(slashPtr+2,
                           (FT_ptr - (slashPtr+2)) );
      else
         Mptr->DVR.visRange = antoi(slashPtr+1,
                           (FT_ptr - (slashPtr+1)) );
 
      (*NDEX)++;
      return TRUE;
   }
 
}
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isRADAT                                          */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          07 Nov 1996                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:   Determines whether or not the input string is       */
/*              the 'RADAT' group elevation indicator.  If it is,   */
/*              then skip past the 'RADAT' indicator and also the   */
/*              next group which is the RADAT elevation informa-    */
/*              tion.                                               */
/*                                                                  */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                                                                  */
/*  Input:         string - the address of a pointer to a charac-   */
/*                          ter string that may or may not be the   */
/*                          RADAT group.                            */
/*                                                                  */
/*                 Mptr - a pointer to a structure that has the     */
/*                        data type Decoded_METAR.                  */
/*                                                                  */
/*                 NDEX - a pointer to an integer that is the index */
/*                        into an array that contains the indi-     */
/*                        vidual groups of the METAR report being   */
/*                        decoded.  Upon entry, NDEX is the index   */
/*                        of the current group of the METAR report  */
/*                        that is to be indentified.                */
/*                                                                  */
/*  Output:        TRUE - if a RADAT group is found.                */
/*                 FALSE - if no RADAT group is found.              */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
static MDSP_BOOL isRADAT( char **string, Decoded_METAR *Mptr,
                             int *NDEX )
{
   if( strcmp( *string, "RADAT" ) != 0 )
      return FALSE;
   else {
 
      (*NDEX)++;
      (++string);
 
      if( *string == NULL )
         return TRUE;
      else {
         (*NDEX)++;
         (++string);
 
         return TRUE;
      }
 
   }
 
}
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isTornadicActiv                                  */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          15 Sep 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:   Determines whether or not the input character       */
/*              string is signals the beginning of TORNADIC         */
/*              ACTIVITY data.  If it is, then interrogate subse-   */
/*              quent report groups for time, location, and movement*/
/*              of tornado.  Return TRUE, if TORNADIC ACTIVITY is   */
/*              found.  Otherwise, return FALSE.                    */
/*                                                                  */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                                                                  */
/*  Input:         string - the address of a pointer to a charac-   */
/*                          ter string that may or may not signal   */
/*                          TORNADIC ACTIVITY.                      */
/*                                                                  */
/*                 Mptr - a pointer to a structure that has the     */
/*                        data type Decoded_METAR.                  */
/*                                                                  */
/*                 NDEX - a pointer to an integer that is the index */
/*                        into an array that contains the indi-     */
/*                        vidual groups of the METAR report being   */
/*                        decoded.  Upon entry, NDEX is the index   */
/*                        of the current group of the METAR report  */
/*                        that is to be indentified.                */
/*                                                                  */
/*  Output:        TRUE - if TORNADIC ACTIVITY is found.            */
/*                 FALSE - if no TORNADIC ACTIVITY is found.        */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
static MDSP_BOOL isTornadicActiv( char **string, Decoded_METAR *Mptr,
                             int *NDEX )
{
   int saveNdex,
       TornadicTime;
   MDSP_BOOL Completion_flag;
   char *B_stringPtr,
        *E_stringPtr;
 
 
   /*************************/
   /* START BODY OF ROUTINE */
   /*************************/
 
 
   saveNdex = *NDEX;
 
   B_stringPtr = NULL;
   E_stringPtr = NULL;
 
   if( *string == NULL )
      return FALSE;
 
   if( !( strcmp(*string, "TORNADO")         == 0 ||
          strcmp(*string, "TORNADOS")        == 0 ||
          strcmp(*string, "TORNADOES")       == 0 ||
          strcmp(*string, "WATERSPOUT")      == 0 ||
          strcmp(*string, "WATERSPOUTS")     == 0 ||
          strcmp(*string, "FUNNEL")     == 0  ) )
         return FALSE;
   else {
      if( strcmp(*string, "FUNNEL") == 0 ) {
         (++string);
 
         if( *string == NULL )
            return FALSE;
 
         if( !(strcmp(*string,"CLOUD") == 0 ||
               strcmp(*string,"CLOUDS") == 0 ) ) {
            (*NDEX)++;
            return FALSE;
         }
         else
               strcpy(Mptr->TornadicType,"FUNNEL CLOUD");
      }
      else {
         strcpy(Mptr->TornadicType, *string);
         (*NDEX)++;
         (++string);
      }
 
      Completion_flag = FALSE;
 
      if( *string == NULL )
         return FALSE;
 
      while( !Completion_flag ) {
 
/*       printf("isTornadicActivity:  current *string = %s\n",
                        *string);    */
 
         if( *(*string) =='B' || *(*string) == 'E') {
            if( *(*string) == 'B' ) {
               B_stringPtr = *string;
               E_stringPtr = strchr((*string)+1,'E');
            }
            else {
               B_stringPtr = strchr((*string)+1,'B');
               E_stringPtr = *string;
            }
/*
         if( B_stringPtr != NULL )
            printf("isTornadicActivity:  B_stringPtr = %x\n",
                        B_stringPtr);
         else
            printf("isTornadicActivity:  B_stringPtr = NULL\n");
 
         if( E_stringPtr != NULL )
            printf("isTornadicActivity:  E_stringPtr = %x\n",
                        E_stringPtr);
         else
            printf("isTornadicActivity:  E_stringPtr = NULL\n");
*/
            if( B_stringPtr != NULL && E_stringPtr == NULL ) {
               if( nisdigit((*string)+1, strlen((*string)+1)) &&
                     strlen((*string)+1) <= 4 ) {
                  TornadicTime = antoi((*string)+1,
                                      strlen((*string)+1));
                  if( TornadicTime > 99 ) {
                     Mptr->BTornadicHour = TornadicTime / 100;
                     Mptr->BTornadicMinute = TornadicTime % 100;
                     (*NDEX)++;
                     (++string);
                  }
                  else {
                     Mptr->BTornadicHour = TornadicTime;
                     (*NDEX)++;
                     (++string);
                  }
               }
               else {
                  (*NDEX)++;
                  (++string);
               }
            }
            else if( B_stringPtr == NULL && E_stringPtr != NULL ) {
               if( nisdigit((*string)+1,strlen((*string)+1)) &&
                        strlen((*string)+1) <= 4 ) {
                  TornadicTime = antoi((*string)+1,
                                     strlen((*string)+1));
                  if( TornadicTime > 99 ) {
                     Mptr->ETornadicHour = TornadicTime / 100;
                     Mptr->ETornadicMinute = TornadicTime % 100;
                     (*NDEX)++;
                     (++string);
                  }
                  else {
                     Mptr->ETornadicHour = TornadicTime;
                     (*NDEX)++;
                     (++string);
                  }
               }
               else {
                  (*NDEX)++;
                  (++string);
               }
            }
            else {
/*          printf("isTornadicActivity:  B_stringPtr != NULL"
                   " and E_stringPtr != NULL\n");  */
               if( nisdigit((B_stringPtr+1),(E_stringPtr -
                                     (B_stringPtr+1)))) {
                  TornadicTime = antoi(( B_stringPtr+1),
                                     (E_stringPtr-(B_stringPtr+1)));
                  if( TornadicTime > 99 ) {
                     Mptr->BTornadicHour = TornadicTime / 100;
                     Mptr->BTornadicMinute = TornadicTime % 100;
                     (*NDEX)++;
                     (++string);
                  }
                  else {
                     Mptr->BTornadicHour = TornadicTime;
                     (*NDEX)++;
                     (++string);
                  }
 
                  TornadicTime = antoi(( E_stringPtr+1),
                                        strlen(E_stringPtr+1));
 
                  if( TornadicTime > 99 ) {
                     Mptr->ETornadicHour = TornadicTime / 100;
                     Mptr->ETornadicMinute = TornadicTime % 100;
                     (*NDEX)++;
                     (++string);
                  }
                  else {
                     Mptr->ETornadicHour = TornadicTime;
                     (*NDEX)++;
                     (++string);
                  }
               }
               else {
                  (*NDEX)++;
                  (++string);
               }
            }
         }
         else if( nisdigit(*string, strlen(*string))) {
            (++string);
 
            if( *string == NULL )
               return FALSE;
 
            if(  strcmp(*string,"N")  == 0  ||
                 strcmp(*string,"NE") == 0  ||
                 strcmp(*string,"NW") == 0  ||
                 strcmp(*string,"S")  == 0  ||
                 strcmp(*string,"SE") == 0  ||
                 strcmp(*string,"SW") == 0  ||
                 strcmp(*string,"E")  == 0  ||
                 strcmp(*string,"W")  == 0   ) {
                 (--string);
                 Mptr->TornadicDistance = antoi(*string,
                                  strlen(*string));
                 (*NDEX)++;
                 (++string);
            }
            else {
               (--string);
 
               if( saveNdex == *NDEX )
                  return FALSE;
               else
                  return TRUE;
            }
 
         }
         else if(strcmp(*string,"DSNT")  == 0 ||
                 strcmp(*string,"VC")    == 0 ||
                 strcmp(*string,"VCY")   == 0 ) {
            if( strcmp(*string,"VCY") == 0 ||
                  strcmp(*string,"VC") == 0  ) {
               (++string);
 
               if( *string == NULL )
                  return FALSE;
 
               if( strcmp(*string,"STN") == 0 ){
                  strcpy(Mptr->TornadicLOC,"VC STN");
                  (*NDEX)++;
                  (*NDEX)++;
                  (++string);
               }
               else {
                  strcpy(Mptr->TornadicLOC,"VC");
                  (*NDEX)++;
               }
            }
            else {
               strcpy(Mptr->TornadicLOC,"DSNT");
               (*NDEX)++;
               (++string);
            }
         }
         else if(strcmp(*string,"N")  == 0  ||
                 strcmp(*string,"NE") == 0  ||
                 strcmp(*string,"NW") == 0  ||
                 strcmp(*string,"S")  == 0  ||
                 strcmp(*string,"SE") == 0  ||
                 strcmp(*string,"SW") == 0  ||
                 strcmp(*string,"E")  == 0  ||
                 strcmp(*string,"W")  == 0   ) {
            strcpy(Mptr->TornadicDIR, *string);
            (*NDEX)++;
            (++string);
         }
         else if( strcmp(*string, "MOV" ) == 0 ) {
            (*NDEX)++;
            (++string);
 
            if( *string == NULL )
               return FALSE;
 
            if(   strcmp(*string, "N")  == 0  ||
                  strcmp(*string, "S")  == 0  ||
                  strcmp(*string, "E")  == 0  ||
                  strcmp(*string, "W")  == 0  ||
                  strcmp(*string, "NE")  == 0 ||
                  strcmp(*string, "NW")  == 0 ||
                  strcmp(*string, "SE")  == 0 ||
                  strcmp(*string, "SW")  == 0     ) {
               strcpy( Mptr->TornadicMovDir, *string );
               (*NDEX)++;
               (++string);
 
            }
         }
         else
            Completion_flag = TRUE;
      }
 
      if( saveNdex == *NDEX )
         return FALSE;
      else
         return TRUE;
 
   }
 
}
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isPartObscur                                     */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          15 Sep 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:   Determine whether or not the input character string */
/*              is a partial obscuration phenomenon.  If a partial  */
/*              obscuration is found, then take the preceding group */
/*              as the obscuring phenomenon.  If a partial obscura- */
/*              tion is found, then return TRUE.  Otherwise, return */
/*              false.                                              */
/*                                                                  */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                                                                  */
/*  Input:         string - the address of a pointer to a group     */
/*                          in a METAR report that may or may not   */
/*                          be a partial obscuration indicator.     */
/*                                                                  */
/*                 Mptr - a pointer to a structure that has the     */
/*                        data type Decoded_METAR.                  */
/*                                                                  */
/*                 NDEX - a pointer to an integer that is the index */
/*                        into an array that contains the indi-     */
/*                        vidual groups of the METAR report being   */
/*                        decoded.  Upon entry, NDEX is the index   */
/*                        of the current group of the METAR report  */
/*                        that is to be indentified.                */
/*                                                                  */
/*  Output:        TRUE - if the input string is a partial obscura- */
/*                        tion.                                     */
/*                 FALSE - if the input string is not a partial ob- */
/*                         scuration.                               */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
static MDSP_BOOL isPartObscur( char **string, Decoded_METAR *Mptr,
                          int ndex, int *NDEX )
{
 
   /***************************/
   /* DECLARE LOCAL VARIABLES */
   /***************************/
 
   int i;
 
   static char *phenom[] = {"-DZ", "DZ", "+DZ",
   "FZDZ", "-RA", "RA", "+RA",
   "SHRA", "TSRA", "FZRA", "-SN", "SN", "+SN", "DRSN",
   "SHSN", "TSSN", "-SG", "SG", "+SG", "IC", "-PE", "PE", "+PE",
   "SHPE", "TSPE", "GR", "SHGR", "TSGR", "GS", "SHGS", "TSGS", "-GS",
   "+GS", "TS", "VCTS", "-TSRA", "TSRA", "+TSRA", "-TSSN", "TSSN",
   "+TSSN", "-TSPE", "TSPE", "+TSPE", "-TSGS", "TSGS", "+TSGS",
   "VCSH", "-SHRA", "+SHRA", "-SHSN", "+SHSN", "-SHPE", "+SHPE",
   "-SHGS", "+SHGS", "-FZDZ", "+FZDZ", "-FZRA", "+FZRA", "FZFG",
   "+FZFG", "BR", "FG", "VCFG", "MIFG", "PRFG", "BCFG", "FU",
   "VA", "DU", "DRDU", "BLDU", "SA", "DRSA", "BLSA", "HZ",
   "BLPY", "BLSN", "+BLSN", "VCBLSN", "BLSA", "+BLSA",
   "VCBLSA", "+BLDU", "VCBLDU", "PO", "VCPO", "SQ", "FC", "+FC",
   "VCFC", "SS", "+SS", "VCSS", "DS", "+DS", "VCDS", NULL};
 
 
#ifdef DEBUGXX
   printf("isPartObscur:  Routine Entered...\n");
   printf("isPartObscur:  *string = %s\n",*string);
   if( Mptr->PartialObscurationAmt[ndex][0] != '\0' ) {
      printf("PartialObscurationAmt = %s\n",
                &(Mptr->PartialObscurationAmt[ndex][0]));
      if( strcmp( *string, "FEW///" ) == 0 ||
          strcmp( *string, "SCT///" ) == 0 ||
          strcmp( *string, "BKN///" ) == 0 ||
          strcmp( *string, "FEW000" ) == 0 ||
          strcmp( *string, "SCT000" ) == 0 ||
          strcmp( *string, "BKN000" ) == 0   ) {
 
          --string;
         printf("isPartObscur:  Preceding group = %s\n",
                  *string);
         ++string;
      }
   }
#endif
 
   if( *string == NULL )
      return FALSE;
 
   if( strcmp( *string, "FEW///" ) == 0 ||
       strcmp( *string, "SCT///" ) == 0 ||
       strcmp( *string, "BKN///" ) == 0 ||
       strcmp( *string, "FEW000" ) == 0 ||
       strcmp( *string, "SCT000" ) == 0 ||
       strcmp( *string, "BKN000" ) == 0   ) {
      if( Mptr->PartialObscurationAmt[ndex][0] == '\0' )
      {
         (*NDEX)++;
         return FALSE;
      }
      else {
         if( strcmp( *string,
                     &(Mptr->PartialObscurationAmt[ndex][0]) ) == 0 )
         {
            --string;
 
            if( *string == NULL )
               return FALSE;
 
            i = 0;
            while( phenom[i] != NULL ) {
               if( strcmp( *string, phenom[i] ) == 0 ) {
                  strcpy(&(Mptr->PartialObscurationPhenom[ndex][0]),
                         *string);
 
                  (*NDEX)++;
                  return TRUE;
               }
               else
                  i++;
            }
 
            (*NDEX)++;
            return FALSE;
 
         }
         else {
            (*NDEX)++;
            return FALSE;
         }
 
      }
 
   }
   else
      return FALSE;
 
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isA0indicator                                    */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          15 Sep 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:   Identify the input character string as an automated */
/*              station code type.  If the input character string   */
/*              is an automated station code type, then return      */
/*              TRUE.  Otherwise, return FALSE.                     */
/*                                                                  */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                                                                  */
/*  Input:         indicator - a pointer to a character string      */
/*                             that may or may not be an ASOS       */
/*                             automated station code type.         */
/*                                                                  */
/*                 Mptr - a pointer to a structure that has the     */
/*                        data type Decoded_METAR.                  */
/*                                                                  */
/*                 NDEX - a pointer to an integer that is the index */
/*                        into an array that contains the indi-     */
/*                        vidual groups of the METAR report being   */
/*                        decoded.  Upon entry, NDEX is the index   */
/*                        of the current group of the METAR report  */
/*                        that is to be indentified.                */
/*                                                                  */
/*  Output:        TRUE - if the input string matches one of the    */
/*                        valid ASOS automated station indicators.  */
/*                 FALSE - the input string did not match one of the*/
/*                        valid ASOS automated station indicators.  */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
static MDSP_BOOL isA0indicator( char *indicator, Decoded_METAR *Mptr,
                           int *NDEX )
{
 
   /***************************/
   /* DECLARE LOCAL VARIABLES */
   /***************************/
 
   char *autoFlag[] = {"A01", "A01A", "A02", "A02A", "AOA",
                       "A0A", "AO1", "AO1A", "AO2", "AO2A", NULL};
   int i;
 
   /*************************/
   /* START BODY OF ROUTINE */
   /*************************/
   /*******************************************/
   /* COMPARE THE INPUT CHARACTER STRING WITH */
   /* VALID AUTOMATED STATION CODE TYPE.  IF  */
   /* A MATCH IS FOUND, RETURN TRUE.  OTHER-  */
   /*           WISE, RETURN FALSE            */
   /*******************************************/
 
   if( indicator == NULL )
      return FALSE;
 
   i = 0;
 
   while( autoFlag[ i ] != NULL )
   {
      if( strcmp( indicator, autoFlag[ i ]) == 0 )
      {
         (*NDEX)++;
         strcpy(Mptr->autoIndicator, indicator);
         return TRUE;
      }
      i++;
   }
 
   return FALSE;
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isPeakWind                                       */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          15 Sep 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:  Determine whether or not the current and subsequent  */
/*             groups from the METAR report make up a valid report  */
/*             of peak wind.                                        */
/*                                                                  */
/*                                                                  */
/*  Input:         string - the addr of a ptr to a character string */
/*                             that may or may not be the indicator */
/*                             for a peak wind data group.          */
/*                                                                  */
/*                 Mptr - a pointer to a structure that has the     */
/*                        data type Decoded_METAR.                  */
/*                                                                  */
/*                 NDEX - a pointer to an integer that is the index */
/*                        into an array that contains the indi-     */
/*                        vidual groups of the METAR report being   */
/*                        decoded.  Upon entry, NDEX is the index   */
/*                        of the current group of the METAR report  */
/*                        that is to be indentified.                */
/*                                                                  */
/*  Output:        TRUE - if the input string (and subsequent grps) */
/*                        are decoded as peak wind.                 */
/*                 FALSE - if the input string (and subsequent grps)*/
/*                         are not decoded as peak wind.            */
/*  External Functions Called:                                      */
/*                 nisdigit                                         */
/*                                                                  */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
static MDSP_BOOL isPeakWind( char **string, Decoded_METAR *Mptr,
                        int *NDEX )
{
 
   /***************************/
   /* DECLARE LOCAL VARIABLES */
   /***************************/
 
   char buf[ 6 ];
   char *slash;
   int temp;
   MDSP_BOOL PK_WND_FLAG;
 
   /*************************/
   /* START BODY OF ROUTINE */
   /*************************/
 
 
 
   /******************************************************/
   /* IF THE CURRENT AND NEXT GROUPS ARE "PK WND", THEN  */
   /* DETERMINE WHETHER OR NOT THE GROUP THAT FOLLOWS IS */
   /* A VALID PK WND GROUP.  IF IT IS, THEN DECODE THE   */
   /* GROUP AND RETURN TRUE.  OTHERWISE, RETURN FALSE.   */
   /******************************************************/
 
   PK_WND_FLAG = TRUE;
 
   if( *string == NULL )
      return FALSE;
 
 
   if( !(strcmp(*string,"PK") == 0 ||
          strcmp(*string,"PKWND") == 0 ) )
      return FALSE;
   else
      (++string);
 
   if( *string == NULL )
      return FALSE;
 
   if( strcmp(*string,"WND") == 0 )
      (++string);
   else
      PK_WND_FLAG = FALSE;
 
   if( *string == NULL )
      return FALSE;
 
   if( (slash = strchr(*string,'/')) == NULL ) {
                              /********************************/
                              /* INVALID PEAK WIND. BUMP PAST */
                              /* PK AND WND GROUP AND RETURN  */
                              /*             FALSE.           */
                              /********************************/
      (*NDEX)++;
 
      if( PK_WND_FLAG )
         (*NDEX)++;
 
      return FALSE;
   }
   else if( strlen(*string) >= 8 && strlen(*string) <= 11 &&
             nisdigit(slash+1,strlen(slash+1)) &&
             nisdigit(*string, (slash - *string)) &&
             (slash - *string) <= 6 )
   {
      memset( buf, '\0', 4);
      strncpy( buf, *string, 3 );
      Mptr->PKWND_dir = atoi( buf );
 
      memset( buf, '\0', 4);
      strncpy( buf, *string+3, slash-(*string+3) );
      Mptr->PKWND_speed =  atoi( buf );
 
      memset( buf, '\0', 5);
      strcpy( buf, slash+1 );
      temp             =  atoi( buf );
 
      if( temp > 99 )
      {
         Mptr->PKWND_hour = atoi(buf)/100;
         Mptr->PKWND_minute = (atoi(buf)) % 100;
      }
      else
         Mptr->PKWND_minute =  atoi( buf );
                              /********************************/
                              /* VALID PEAK WIND FOUND.  BUMP */
                              /* PAST PK, WND, AND PEAK WIND  */
                              /* GROUPS AND RETURN TRUE.      */
                              /********************************/
      (*NDEX)++;
      (*NDEX)++;
 
      if( PK_WND_FLAG )
         (*NDEX)++;
 
      return TRUE;
   }
   else
      return FALSE;
}
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isWindShift                                      */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          15 Sep 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:  Determine whether or not the current and subsequent  */
/*             groups from the METAR report make up a valid report  */
/*             of wind shift and frontal passage, if included.      */
/*                                                                  */
/*                                                                  */
/*  Input:         string - the addr of a ptr to a character string */
/*                           that may or may not be the indicator   */
/*                           for a wind shift data group.           */
/*                                                                  */
/*                 Mptr - a pointer to a structure that has the     */
/*                        data type Decoded_METAR.                  */
/*                                                                  */
/*                 NDEX - a pointer to an integer that is the index */
/*                        into an array that contains the indi-     */
/*                        vidual groups of the METAR report being   */
/*                        decoded.  Upon entry, NDEX is the index   */
/*                        of the current group of the METAR report  */
/*                        that is to be indentified.                */
/*                                                                  */
/*  Output:        TRUE - if the input string (and subsequent grps) */
/*                        are decoded as wind shift.                */
/*                 FALSE - if the input string (and subsequent grps)*/
/*                         are not decoded as wind shift.           */
/*  External Functions Called:                                      */
/*                 nisdigit                                         */
/*                                                                  */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
static MDSP_BOOL isWindShift( char **string, Decoded_METAR *Mptr,
                        int *NDEX)
{
 
   /***************************/
   /* DECLARE LOCAL VARIABLES */
   /***************************/
 
   int temp;
 
   /*************************/
   /* START BODY OF ROUTINE */
   /*************************/
 
 
   /****************************************************/
   /* IF THE CURRENT GROUP IS "WSHFT", THEN DETERMINE  */
   /* WHETHER OR NOT THE GROUP THAT FOLLOWS IS A VALID */
   /* WSHFT GROUP.  IF IT IS, THEN DECODE THE GROUP    */
   /* GROUP AND RETURN TRUE.  OTHERWISE, RETURN FALSE. */
   /****************************************************/
 
   if( *string == NULL )
      return FALSE;
 
   if( strcmp( *string, "WSHFT" ) != 0 )
      return FALSE;
   else
      (++string);
 
   if( *string == NULL )
      return FALSE;
 
   if( nisdigit(*string,strlen(*string)) && strlen(*string) <= 4)
   {
      temp = atoi( *string );
 
      if( temp > 100 )
      {
         Mptr->WshfTime_hour = (atoi(*string))/100;
         Mptr->WshfTime_minute = (atoi(*string)) % 100;
      }
      else
         Mptr->WshfTime_minute = (atoi(*string)) % 100;
 
      (++string);
 
      if( *string == NULL )
         return FALSE;
 
 
      if( **string == '\0') {
         (*NDEX)++;
         (*NDEX)++;
         return TRUE;
      }
      else if( strcmp( *string, "FROPA") == 0 )
      {
         Mptr->Wshft_FROPA = TRUE;
                              /********************************/
                              /* VALID WIND SHIFT FOUND. BUMP */
                              /* PAST WSHFT, WSHFT GROUP, AND */
                              /* FROPA GROUPS AND RETURN TRUE.*/
                              /********************************/
         (*NDEX)++;
         (*NDEX)++;
         (*NDEX)++;
         return TRUE;
      }
      else {
                              /********************************/
                              /* VALID WIND SHIFT FOUND. BUMP */
                              /* PAST WSHFT AND WSHFT GROUP   */
                              /*       AND RETURN TRUE.       */
                              /********************************/
         (*NDEX)++;
         (*NDEX)++;
         return TRUE;
      }
   }
   else {
                              /**********************************/
                              /* INVALID WIND SHIFT FOUND. BUMP */
                              /* PAST WSHFT AND RETURN FALSE.   */
                              /********************************/
      (*NDEX)++;
      return FALSE;
   }
}
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isTowerVsby                                      */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          15 Sep 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:  Determine whether or not the current and subsequent  */
/*             groups from the METAR report make up a valid report  */
/*             of tower visibility.                                 */
/*                                                                  */
/*                                                                  */
/*  Input:         string - the addr of a ptr to a character string */
/*                          that may or may not be the indicator    */
/*                          for tower visibility.                   */
/*                                                                  */
/*                 Mptr - a pointer to a structure that has the     */
/*                        data type Decoded_METAR.                  */
/*                                                                  */
/*                 NDEX - a pointer to an integer that is the index */
/*                        into an array that contains the indi-     */
/*                        vidual groups of the METAR report being   */
/*                        decoded.  Upon entry, NDEX is the index   */
/*                        of the current group of the METAR report  */
/*                        that is to be indentified.                */
/*                                                                  */
/*  Output:        TRUE - if the input string (and subsequent grps) */
/*                        are decoded as tower visibility.          */
/*                 FALSE - if the input string (and subsequent grps)*/
/*                         are not decoded as tower visibility      */
/*  External Functions Called:                                      */
/*                 nisdigit                                         */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
static MDSP_BOOL isTowerVsby( char **token, Decoded_METAR *Mptr, int *NDEX)
{
 
   /***************************/
   /* DECLARE LOCAL VARIABLES */
   /***************************/
 
   char *slash;
   float T_vsby;
 
   /*************************/
   /* START BODY OF ROUTINE */
   /*************************/
 
 
   /****************************************************************/
   /* IF THE CURRENT AND NEXT GROUPS ARE "TWR VIS", THEN DETERMINE */
   /* WHETHER OR NOT THE GROUP(S) THAT FOLLOWS IS(ARE) A VALID     */
   /* TOWER VISIBILITY  GROUP.  IF IT IS, THEN DECODE THE GROUP    */
   /* GROUP AND RETURN TRUE.  OTHERWISE, RETURN FALSE.             */
   /****************************************************************/
 
   if( *token == NULL )
      return FALSE;
 
   if(strcmp(*token,"TWR") != 0)
      return FALSE;
   else
      (++token);
 
   if( *token == NULL )
      return FALSE;
 
   if( strcmp(*token,"VIS") != 0) {
      (*NDEX)++;
      return FALSE;
   }
   else
      (++token);
 
   if( *token == NULL )
      return FALSE;
 
   if( nisdigit(*token,
              strlen(*token)))
   {
      Mptr->TWR_VSBY = (float) atoi(*token);
      (++token);
      if( *token != NULL )
      {
         if( (slash = strchr(*token, '/'))
                             != NULL )
         {
            if( nisdigit(slash+1,strlen(slash+1)) &&
                         nisdigit(*token,
                             (slash-*token)))
            {
               T_vsby = fracPart(*token);
               Mptr->TWR_VSBY += T_vsby;
               (*NDEX)++;
               (*NDEX)++;
               (*NDEX)++;
               (*NDEX)++;
               return TRUE;
            }
            else {
               (*NDEX)++;
               (*NDEX)++;
               (*NDEX)++;
               return TRUE;
            }
 
         }
         else {
            (*NDEX)++;
            (*NDEX)++;
            (*NDEX)++;
            return TRUE;
         }
      }
      else {
         (*NDEX)++;
         (*NDEX)++;
         (*NDEX)++;
         return TRUE;
      }
 
   }
   else if( (slash = strchr(*token, '/'))
                             != NULL )
   {
      if( nisdigit(slash+1,strlen(slash+1)) &&
                         nisdigit(*token,
                             (slash-*token)))
      {
         Mptr->TWR_VSBY = fracPart(*token);
         (*NDEX)++;
         (*NDEX)++;
         (*NDEX)++;
         return TRUE;
      }
      else {
         (*NDEX)++;
         (*NDEX)++;
         return FALSE;
      }
   }
   else {
      (*NDEX)++;
      (*NDEX)++;
      return FALSE;
   }
 
}
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isSurfaceVsby                                    */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          15 Sep 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:  Determine whether or not the current and subsequent  */
/*             groups from the METAR report make up a valid report  */
/*             of surface visibility.                               */
/*                                                                  */
/*                                                                  */
/*  Input:         string - the addr of a ptr to a character string */
/*                          that may or may not be the indicator    */
/*                          for surface visibility.                 */
/*                                                                  */
/*                 Mptr - a pointer to a structure that has the     */
/*                        data type Decoded_METAR.                  */
/*                                                                  */
/*                 NDEX - a pointer to an integer that is the index */
/*                        into an array that contains the indi-     */
/*                        vidual groups of the METAR report being   */
/*                        decoded.  Upon entry, NDEX is the index   */
/*                        of the current group of the METAR report  */
/*                        that is to be indentified.                */
/*                                                                  */
/*  Output:        TRUE - if the input string (and subsequent grps) */
/*                        are decoded as surface visibility.        */
/*                 FALSE - if the input string (and subsequent grps)*/
/*                         are not decoded as surface visibility.   */
/*  External Functions Called:                                      */
/*                 nisdigit                                         */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
static MDSP_BOOL isSurfaceVsby( char **token, Decoded_METAR *Mptr,
                           int *NDEX)
{
 
   /***************************/
   /* DECLARE LOCAL VARIABLES */
   /***************************/
 
   char *slash;
   float S_vsby;
 
 
   /*************************/
   /* START BODY OF ROUTINE */
   /*************************/
 
 
   /****************************************************************/
   /* IF THE CURRENT AND NEXT GROUPS ARE "SFC VIS", THEN DETERMINE */
   /* WHETHER OR NOT THE GROUP(S) THAT FOLLOWS IS(ARE) A VALID     */
   /* SURFACE VISIBILITY  GROUP.  IF IT IS, THEN DECODE THE GROUP  */
   /* GROUP AND RETURN TRUE.  OTHERWISE, RETURN FALSE.             */
   /****************************************************************/
 
   if( *token == NULL )
      return FALSE;
 
   if(strcmp(*token,"SFC") != 0)
      return FALSE;
   else
      (++token);
 
   if( strcmp(*token,"VIS") != 0) {
      (*NDEX)++;
      return FALSE;
   }
   else
      (++token);
 
 
   if( *token == NULL )
      return FALSE;
 
 
   if( nisdigit(*token,
              strlen(*token)))
   {
      Mptr->SFC_VSBY = (float) atoi(*token);
      (++token);
      if( *token != NULL )
      {
         if( (slash = strchr(*token, '/'))
                             != NULL )
         {
            if( nisdigit(slash+1,strlen(slash+1)) &&
                         nisdigit(*token,
                             (slash-*token)))
            {
               S_vsby = fracPart(*token);
               Mptr->SFC_VSBY += S_vsby;
               (*NDEX)++;
               (*NDEX)++;
               (*NDEX)++;
               (*NDEX)++;
               return TRUE;
            }
            else {
               (*NDEX)++;
               (*NDEX)++;
               (*NDEX)++;
               return TRUE;
            }
 
         }
         else {
            (*NDEX)++;
            (*NDEX)++;
            (*NDEX)++;
            return TRUE;
         }
      }
      else {
         (*NDEX)++;
         (*NDEX)++;
         (*NDEX)++;
         return TRUE;
      }
 
   }
   else if( (slash = strchr(*token, '/'))
                             != NULL )
   {
      if( nisdigit(slash+1,strlen(slash+1)) &&
                         nisdigit(*token,
                             (slash-*token)))
      {
         Mptr->SFC_VSBY = fracPart(*token);
         (*NDEX)++;
         (*NDEX)++;
         (*NDEX)++;
         return TRUE;
      }
      else {
         (*NDEX)++;
         (*NDEX)++;
         return FALSE;
      }
   }
   else {
      (*NDEX)++;
      (*NDEX)++;
      return FALSE;
   }
 
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isVariableVsby                                   */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          21 Nov 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:  Determine whether or not the current and subsequent  */
/*             groups from the METAR report make up a valid report  */
/*             of variable prevailing visibility.                   */
/*                                                                  */
/*                                                                  */
/*  Input:         string - the addr of a ptr to a character string */
/*                          that may or may not be the indicator    */
/*                          for variable prevailing visibility.     */
/*                                                                  */
/*                 Mptr - a pointer to a structure that has the     */
/*                        data type Decoded_METAR.                  */
/*                                                                  */
/*                 NDEX - a pointer to an integer that is the index */
/*                        into an array that contains the indi-     */
/*                        vidual groups of the METAR report being   */
/*                        decoded.  Upon entry, NDEX is the index   */
/*                        of the current group of the METAR report  */
/*                        that is to be indentified.                */
/*                                                                  */
/*  Output:        TRUE - if the input string (and subsequent grps) */
/*                        are decoded as variable prevailing vsby.  */
/*                 FALSE - if the input string (and subsequent grps)*/
/*                         are not decoded as variable prevailing   */
/*                         vsby.                                    */
/*  External Functions Called:                                      */
/*                 nisdigit                                         */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
static MDSP_BOOL isVariableVsby( char **string, Decoded_METAR *Mptr,
                              int *NDEX )
{
 
   /***************************/
   /* DECLARE LOCAL VARIABLES */
   /***************************/
 
   char *slash,
        *slash1,
        *slash2,
        buf[ 5 ],
        *V_char;
   float minimumVsby,
         maximumVsby;
 
 
 
   /*************************/
   /* START BODY OF ROUTINE */
   /*************************/
 
 
   /***************************************************/
   /* IF THE CURRENT GROUP IS  "VIS", THEN DETERMINE  */
   /* WHETHER OR NOT THE GROUPS THAT FOLLOW ARE VALID */
   /* VARIABLE PREVAILING VSBY.  IF THEY ARE, THEN    */
   /* DECODE THE GROUPS AND RETURN TRUE.  OTHERWISE,  */
   /* RETURN FALSE.                                   */
   /***************************************************/
 
   if( *string == NULL )
      return FALSE;
 
   if( strcmp(*string, "VIS") != 0 )
      return FALSE;
   else
      (++string);
 
   if( *string == NULL )
      return FALSE;
 
   if( !((V_char = strchr(*string, 'V')) != NULL ||
         nisdigit(*string,strlen(*string))) )
      return FALSE;
   else if( nisdigit(*string,strlen(*string)) ) {
      minimumVsby = (float) atoi(*string);
      (++string);
 
      if( *string == NULL )
         return FALSE;
 
      if( (V_char = strchr(*string,'V')) == NULL )
         return FALSE;
      else {
         if( (slash = strchr(*string,'/')) == NULL )
            return FALSE;
         else {
            if( nisdigit(*string,(slash - *string)) &&
                  nisdigit(slash+1,(V_char-(slash+1))) &&
                  nisdigit(V_char+1,strlen(V_char+1)) ) {
               if( (V_char - *string) > 4 )
                  return FALSE;
               else {
                  memset(buf,'\0',5);
                  strncpy(buf,*string,(V_char - *string));
                  Mptr->minVsby = minimumVsby + fracPart(buf);
                  maximumVsby = (float) atoi(V_char+1);
               }
 
               (++string);
 
               if( *string == NULL )
                  return FALSE;
 
               if( (slash = strchr(*string,'/')) == NULL ) {
                  Mptr->maxVsby = maximumVsby;
                  (*NDEX)++;
                  (*NDEX)++;
                  (*NDEX)++;
                  return TRUE;
               }
               else if( nisdigit(*string,(slash - *string)) &&
                           nisdigit(slash+1, strlen(slash+1)) ) {
                  Mptr->maxVsby = maximumVsby + fracPart(*string);
                  (*NDEX)++;
                  (*NDEX)++;
                  (*NDEX)++;
                  (*NDEX)++;
                  return TRUE;
               }
               else {
                  Mptr->maxVsby = maximumVsby;
                  (*NDEX)++;
                  (*NDEX)++;
                  (*NDEX)++;
                  return TRUE;
               }
            }
            else
               return FALSE;
         }
      }
   }
   else {
      if( (V_char = strchr(*string,'V')) == NULL )
         return FALSE;
      if(nisdigit(*string,(V_char - *string)) &&
            nisdigit(V_char+1,strlen(V_char+1)) ) {
         Mptr->minVsby = (float) antoi(*string,(V_char - *string));
         maximumVsby = (float) atoi(V_char+1);
 
         (++string);
 
         if( *string == NULL )
            return FALSE;
 
         if( (slash = strchr(*string,'/')) == NULL ) {
            Mptr->maxVsby = maximumVsby;
            (*NDEX)++;
            (*NDEX)++;
            return TRUE;
         }
         else if( nisdigit(*string, (slash - *string)) &&
                     nisdigit(slash+1,strlen(slash+1)) ) {
            Mptr->maxVsby = maximumVsby + fracPart( *string );
            (*NDEX)++;
            (*NDEX)++;
            (*NDEX)++;
            return TRUE;
         }
         else {
            Mptr->maxVsby = maximumVsby;
            (*NDEX)++;
            (*NDEX)++;
            return TRUE;
         }
      }
      else {
         if( (slash2 = strchr(V_char+1,'/')) == NULL &&
               (slash1 = strchr(*string,'/')) == NULL )
            return FALSE;
         else if( slash1 == NULL )
            return FALSE;
         else if( slash == slash2 )
            return FALSE;
         else if( nisdigit(*string,(slash1 - *string)) &&
                     nisdigit((slash1+1),(V_char-(slash1+1))) ) {
            if( (V_char - *string) > 4 )
               return FALSE;
            else {
               memset(buf,'\0',5);
               strncpy(buf,*string,(V_char - *string));
               minimumVsby = fracPart(buf);
            }
            if( slash2 == NULL) {
               if( nisdigit(V_char+1, strlen(V_char+1)) ) {
                  maximumVsby = (float) atoi(V_char+1);
 
                  (++string);
 
                  if( *string == NULL )
                     return FALSE;
 
                  if( (slash = strchr(*string,'/')) == NULL ) {
                     Mptr->minVsby = minimumVsby;
                     Mptr->maxVsby = maximumVsby;
                     (*NDEX)++;
                     (*NDEX)++;
                     return TRUE;
                  }
                  else if( nisdigit(*string,(slash-*string)) &&
                         nisdigit((slash+1),strlen(slash+1)) ) {
                     Mptr->minVsby = minimumVsby;
                     Mptr->maxVsby = maximumVsby +
                                        fracPart(*string);
                     (*NDEX)++;
                     (*NDEX)++;
                     (*NDEX)++;
                     return TRUE;
                  }
                  else{
                     Mptr->minVsby = minimumVsby;
                     Mptr->maxVsby = maximumVsby;
                     (*NDEX)++;
                     (*NDEX)++;
                     return TRUE;
                  }
               }
               else
                  return FALSE;
            }
            else {
               if( nisdigit(V_char+1,(slash2-V_char+1)) &&
                     nisdigit((slash2+1),strlen(slash2+1)) ) {
                  Mptr->minVsby = minimumVsby;
                  Mptr->maxVsby = fracPart(V_char+1);
                  (*NDEX)++;
                  (*NDEX)++;
                  return TRUE;
               }
               else
                  return FALSE;
            }
         }
      }
   }
   return FALSE;
}
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isVsby2ndSite                                    */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          15 Sep 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:  Determine whether or not the current and subsequent  */
/*             groups from the METAR report make up a valid report  */
/*             of visibility at a secondary site.                   */
/*                                                                  */
/*                                                                  */
/*  Input:         token  - the addr of a ptr to a character string */
/*                          that may or may not be the indicator    */
/*                          for visibility at a secondary site.     */
/*                                                                  */
/*                 Mptr - a pointer to a structure that has the     */
/*                        data type Decoded_METAR.                  */
/*                                                                  */
/*                 NDEX - a pointer to an integer that is the index */
/*                        into an array that contains the indi-     */
/*                        vidual groups of the METAR report being   */
/*                        decoded.  Upon entry, NDEX is the index   */
/*                        of the current group of the METAR report  */
/*                        that is to be indentified.                */
/*                                                                  */
/*  Output:        TRUE - if the input string (and subsequent grps) */
/*                        are decoded as visibility at a 2ndry site.*/
/*                 FALSE - if the input string (and subsequent grps)*/
/*                         are not decoded as visibility at a 2ndry */
/*                         site.                                    */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 nisalnum                                         */
/*                 fracPart                                         */
/*                 nisdigit                                         */
/*                                                                  */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
static MDSP_BOOL isVsby2ndSite( char **token, Decoded_METAR *Mptr,
                           int *NDEX)
{
 
   /***************************/
   /* DECLARE LOCAL VARIABLES */
   /***************************/
 
   char *slash;
   float S_vsby,
         VSBY_2ndSite;
 
 
   /*************************/
   /* START BODY OF ROUTINE */
   /*************************/
 	

   /***************************************************/
   /* IF THE CURRENT GROUP IS  "VIS", THEN DETERMINE  */
   /* WHETHER OR NOT THE GROUPS THAT FOLLOW ARE VALID */
   /* VSBILITY AT A 2NDRY SITE.  IF THEY ARE, THEN    */
   /* DECODE THE GROUPS AND RETURN TRUE.  OTHERWISE,  */
   /* RETURN FALSE.                                   */
   /***************************************************/
 
   if( *token == NULL )
      return FALSE;
 
   if(strcmp(*token,"VIS") != 0)
      return FALSE;
   else
      (++token);

   if( *token == NULL )
      return FALSE;
 
   if( nisdigit(*token,
              strlen(*token)))
   {
      VSBY_2ndSite = (float) atoi(*token);
      (++token);
      if( *token != NULL )
      {
         if( (slash = strchr(*token, '/'))
                             != NULL )
         {
            if( nisdigit(slash+1,strlen(slash+1)) &&
                         nisdigit(*token,
                             (slash-*token)))
            {
               S_vsby = fracPart(*token);
 
               (++token);
 
               if( *token == NULL )
                  return FALSE;
 
               if( strncmp( *token, "RMY", 3 ) == 0) {
                  if( nisalnum( *token, strlen(*token) ) ) {
                     strcpy(Mptr->VSBY_2ndSite_LOC, *token);
                     Mptr->VSBY_2ndSite = VSBY_2ndSite + S_vsby;
                     (*NDEX)++;
                     (*NDEX)++;
                     (*NDEX)++;
                     (*NDEX)++;
                     return TRUE;
                  }
                  else
                     return FALSE;
               }
               else
                  return FALSE;
            }
            else {
               if( strncmp( *token, "RMY", 3 ) == 0) {
                  if( nisalnum( *token, strlen(*token) ) ) {
                     strcpy(Mptr->VSBY_2ndSite_LOC, *token);
                     Mptr->VSBY_2ndSite = VSBY_2ndSite;
                     (*NDEX)++;
                     (*NDEX)++;
                     (*NDEX)++;
                     return TRUE;
                  }
                  else
                     return FALSE;
               }
               else
                  return FALSE;
            }
 
         }
         else {
            if( strncmp( *token, "RMY", 3 ) == 0) {
               if( nisalnum( *token, strlen(*token) ) ) {
                  strcpy(Mptr->VSBY_2ndSite_LOC, *token);
                  Mptr->VSBY_2ndSite = VSBY_2ndSite;
                  (*NDEX)++;
                  (*NDEX)++;
                  (*NDEX)++;
                  return TRUE;
               }
               else
                  return TRUE;
            }
            else
               return FALSE;
         }
      }
      else {
         if(*token != NULL && strncmp( *token, "RMY", 3 ) == 0) {
            if( nisalnum( *token, strlen(*token) ) ) {
               strcpy(Mptr->VSBY_2ndSite_LOC, *token);
               Mptr->VSBY_2ndSite = VSBY_2ndSite;
               (*NDEX)++;
               (*NDEX)++;
               (*NDEX)++;
               return TRUE;
            }
            else
               return FALSE;
         }
         else
            return FALSE;
      }
 
   }
   else if( (slash = strchr(*token, '/'))
                             != NULL )
   {
      if( nisdigit(slash+1,strlen(slash+1)) &&
                         nisdigit(*token,
                             (slash-*token)))
      {
         VSBY_2ndSite = fracPart(*token);
         if( strncmp( *(++token), "RMY", 3 ) == 0) {
            if( nisalnum( *token, strlen(*token) ) ) {
               Mptr->VSBY_2ndSite = VSBY_2ndSite;
               strcpy(Mptr->VSBY_2ndSite_LOC, *token);
               (*NDEX)++;
               (*NDEX)++;
               (*NDEX)++;
               return TRUE;
            }
            else
               return FALSE;
         }
         else
            return FALSE;
      }
      else
         return FALSE;
   }
   else
      return FALSE;
 
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isLTGfreq                                        */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          15 Sep 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:  Determine whether or not the current and subsequent  */
/*             groups from the METAR report make up a valid report  */
/*             of lightning.                                        */
/*                                                                  */
/*                                                                  */
/*  Input:        string  - the addr of a ptr to a character string */
/*                          that may or may not be the indicator    */
/*                          for lightning.                          */
/*                                                                  */
/*                 Mptr - a pointer to a structure that has the     */
/*                        data type Decoded_METAR.                  */
/*                                                                  */
/*                 NDEX - a pointer to an integer that is the index */
/*                        into an array that contains the indi-     */
/*                        vidual groups of the METAR report being   */
/*                        decoded.  Upon entry, NDEX is the index   */
/*                        of the current group of the METAR report  */
/*                        that is to be indentified.                */
/*                                                                  */
/*  Output:        TRUE - if the input string (and subsequent grps) */
/*                        are decoded as lightning.                 */
/*                 FALSE - if the input string (and subsequent grps)*/
/*                         are not decoded as lightning.            */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 NONE.                                            */
/*                                                                  */
/*                                                                  */
/*  Modification History:                                           */
/*                 09 May 1996:  Software modified to properly      */
/*                               decode lightning types.            */
/*                                                                  */
/*                                                                  */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
MDSP_BOOL static isLTGfreq( char **string, Decoded_METAR *Mptr, int *NDEX )
{
 
   /***************************/
   /* DECLARE LOCAL VARIABLES */
   /***************************/
 
   MDSP_BOOL LTG_FREQ_FLAG,
        LTG_TYPE_FLAG,
        LTG_LOC_FLAG,
        LTG_DIR_FLAG,
        TYPE_NOT_FOUND;
   char *temp;
 
   /*************************/
   /* START BODY OF ROUTINE */
   /*************************/
 
 
 
   /***************************************************/
   /* IF THE CURRENT GROUP IS  "LTG", THEN DETERMINE  */
   /* WHETHER OR NOT THE PREVIOUS GROUP AS WELL AS    */
   /* GROUPS THAT FOLLOW ARE VALID LIGHTNING REPORT   */
   /* PARAMETERS.  IF THEY ARE, THEN DECODE THE       */
   /* GROUPS AND RETURN TRUE.  OTHERWISE, RETURN      */
   /*                   FALSE.                        */
   /***************************************************/
 
   if( *string == NULL )
      return FALSE;
 
   if( strcmp(*string,"VCTS") == 0 ) {
      Mptr->LightningVCTS = TRUE;
      (++string);
      (*NDEX)++;
      return TRUE;
   }
 
   if( *string == NULL )
      return FALSE;
 
   if( strncmp( *string, "LTG", 3 ) != 0 ) {
      return FALSE;
   }
   else {
 
      if( *string == NULL )
         return FALSE;
 
      (--string);
 
 
      LTG_FREQ_FLAG = FALSE;
                        /*-- CHECK FOR LIGHTNING FREQUENCY -----------*/
      if( strcmp( *string, "OCNL" ) == 0 ) {
         Mptr->OCNL_LTG = TRUE;
         LTG_FREQ_FLAG = TRUE;
      }
      else if( strcmp( *string, "FRQ" ) == 0 ) {
         Mptr->FRQ_LTG = TRUE;
         LTG_FREQ_FLAG = TRUE;
      }
      else if( strcmp( *string, "CONS" ) == 0 ) {
         Mptr->CNS_LTG = TRUE;
         LTG_FREQ_FLAG = TRUE;
      }
 
 
      (++string);
 
      if( *string == NULL )
         return FALSE;
 
 
      if( strcmp( *string, "LTG") == 0 ) {
         (++string);
 
         if( *string == NULL )
            return FALSE;
 
         (*NDEX)++;
 
         LTG_LOC_FLAG = FALSE;
                        /*-- CHECK FOR LIGHTNING LOCATION ------------*/
         if( strcmp( *string, "DSNT" ) == 0 ) {
            Mptr->DSNT_LTG = TRUE;
            LTG_LOC_FLAG = TRUE;
         }
         else if( strcmp( *string, "AP" ) == 0 ) {
            Mptr->AP_LTG = TRUE;
            LTG_LOC_FLAG = TRUE;
         }
         else if( strcmp( *string, "VCY" ) == 0 ||
                  strcmp( *string, "VC"  ) == 0 ) {
            Mptr->VcyStn_LTG = TRUE;
            LTG_LOC_FLAG = TRUE;
         }
         else if( strcmp( *string, "OVHD" ) == 0 ||
                  strcmp( *string, "OHD"  ) == 0 ) {
            Mptr->OVHD_LTG = TRUE;
            LTG_LOC_FLAG = TRUE;
         }
 
         if( LTG_LOC_FLAG )
            (++string);
 
         if( *string == NULL ) {
            if( LTG_LOC_FLAG )
               (*NDEX)++;
            return TRUE;
         }
 
         LTG_DIR_FLAG = FALSE;
                        /*-- CHECK FOR LIGHTNING DIRECTION -----------*/
         if( strcmp( *string, "N" ) == 0 ||
             strcmp( *string, "NE" ) == 0 ||
             strcmp( *string, "NW" ) == 0 ||
             strcmp( *string, "S" ) == 0 ||
             strcmp( *string, "SE" ) == 0 ||
             strcmp( *string, "SW" ) == 0 ||
             strcmp( *string, "E" ) == 0 ||
             strcmp( *string, "W" ) == 0    ) {
            strcpy( Mptr->LTG_DIR, *string);
            LTG_DIR_FLAG = TRUE;
         }
 
 
         if( LTG_LOC_FLAG )
            (*NDEX)++;
         if( LTG_DIR_FLAG )
            (*NDEX)++;
 
         return TRUE;
      }
      else {
 
         LTG_TYPE_FLAG = FALSE;
                        /*-- CHECK FOR LIGHTNING TYPE ----------------*/
         TYPE_NOT_FOUND = FALSE;
         temp = (*string) + 3;
         while( *temp != '\0' && !TYPE_NOT_FOUND ) {
            if( strncmp( temp, "CG", 2 ) == 0 ) {
               Mptr->CG_LTG = TRUE;
               LTG_TYPE_FLAG = TRUE;
               temp++;
               temp++;
            }
            else if( strncmp( temp, "IC", 2 ) == 0 ) {
               Mptr->IC_LTG = TRUE;
               LTG_TYPE_FLAG = TRUE;
               temp++;
               temp++;
            }
            else if( strncmp( temp, "CC", 2 ) == 0 ) {
               Mptr->CC_LTG = TRUE;
               LTG_TYPE_FLAG = TRUE;
               temp++;
               temp++;
            }
            else if( strncmp( temp, "CA", 2 ) == 0 ) {
               Mptr->CA_LTG = TRUE;
               LTG_TYPE_FLAG = TRUE;
               temp++;
               temp++;
            }
            else
               TYPE_NOT_FOUND = TRUE;
         }
 
         (++string);
 
         if( *string == NULL ) {
            (*NDEX)++;
            return TRUE;
         }
/*       else
            (*NDEX)++;   TURNED OFF 07-24-97  */
 
         LTG_LOC_FLAG = FALSE;
                        /*-- CHECK FOR LIGHTNING LOCATION ------------*/
         if( strcmp( *string, "DSNT" ) == 0 ) {
            Mptr->DSNT_LTG = TRUE;
            LTG_LOC_FLAG = TRUE;
         }
         else if( strcmp( *string, "AP" ) == 0 ) {
            Mptr->AP_LTG = TRUE;
            LTG_LOC_FLAG = TRUE;
         }
         else if( strcmp( *string, "VCY" ) == 0 ||
                  strcmp( *string, "VC"  ) == 0 ) {
            Mptr->VcyStn_LTG = TRUE;
            LTG_LOC_FLAG = TRUE;
         }
         else if( strcmp( *string, "OVHD" ) == 0 ) {
            Mptr->OVHD_LTG = TRUE;
            LTG_LOC_FLAG = TRUE;
         }
 
         if( LTG_LOC_FLAG )
            (++string);
 
         if( *string == NULL ) {
            if( LTG_LOC_FLAG )
               (*NDEX)++;
            if( LTG_TYPE_FLAG )
               (*NDEX)++;
            return TRUE;
         }
 
         LTG_DIR_FLAG = FALSE;
                        /*-- CHECK FOR LIGHTNING DIRECTION -----------*/
         if( strcmp( *string, "N" ) == 0 ||
             strcmp( *string, "NE" ) == 0 ||
             strcmp( *string, "NW" ) == 0 ||
             strcmp( *string, "S" ) == 0 ||
             strcmp( *string, "SE" ) == 0 ||
             strcmp( *string, "SW" ) == 0 ||
             strcmp( *string, "E" ) == 0 ||
             strcmp( *string, "W" ) == 0    ) {
            strcpy( Mptr->LTG_DIR, *string);
            LTG_DIR_FLAG = TRUE;
         }
 
 
         if( LTG_TYPE_FLAG )
            (*NDEX)++;
         if( LTG_LOC_FLAG )
            (*NDEX)++;
         if( LTG_DIR_FLAG )
            (*NDEX)++;
 
         if( !(LTG_TYPE_FLAG) &&     /*  Added on 02/23/98 to prevent */
             !(LTG_LOC_FLAG)  &&     /*  infinite looping when 'LTG'  */
             !(LTG_DIR_FLAG)    )    /*  is present in the input, but */
            (*NDEX)++;               /*  all other related parameters */
                                     /*  are missing or invalid       */
         return TRUE;
      }
   }
}
 
 
#pragma comment (compiler)
#pragma comment (date)
#pragma comment (timestamp)
#pragma pagesize(80)
 
#include "metar_structs.h"     /* standard header file */
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isRecentWx                                       */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          15 Sep 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:  Determine whether or not the current and subsequent  */
/*             groups from the METAR report make up a valid report  */
/*             recent weather.                                      */
/*                                                                  */
/*  Input:         token  - the addr of a ptr to a character token */
/*                          that may or may not be a recent weather */
/*                          group.                                  */
/*                                                                  */
/*                 Mptr - a pointer to a structure that has the     */
/*                        data type Decoded_METAR.                  */
/*                                                                  */
/*                 NDEX - a pointer to an integer that is the i*NDEX */
/*                        into an array that contains the indi-     */
/*                        vidual groups of the METAR report being   */
/*                        decoded.  Upon entry, NDEX is the i*NDEX   */
/*                        of the current group of the METAR report  */
/*                        that is to be indentified.                */
/*                                                                  */
/*  Output:        TRUE - if the input token (and possibly subse-  */
/*                        quent groups) are decoded as recent wx.   */
/*                 FALSE - if the input token (and possibly subse- */
/*                         quent groups) are not decoded as recent  */
/*                         wx.                                      */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 nisdigit                                         */
/*                                                                  */
/*  Input:         x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
static MDSP_BOOL isRecentWX( char **token, Decoded_METAR *Mptr,
                        int *NDEX )
{
   static char *phenom[] = {"FCB", "-DZB", "DZB", "+DZB",
   "FZDZB", "-RAB", "RAB", "+RAB",
   "SHRAB", "TSRAB", "FZRAB", "-SNB",
   "SNB", "+SNB", "DRSNB", "BLSNB",
   "SHSNB", "TSSNB", "-SGB", "SGB",
   "+SGB", "ICB", "-PEB", "PEB", "+PEB",
   "SHPEB", "TSPEB", "GRB", "SHGRB",
   "TSGRB", "GSB", "SHGSB", "TSGSB", "-GSB",
   "+GSB", "TSB", "VCTSB", "-TSRAB",
   "TSRAB", "+TSRAB", "-TSSNB", "TSSNB",
   "+TSSNB", "-TSPEB", "TSPEB", "+TSPEB",
   "-TSGSB", "TSGSB", "+TSGSB",
   "VCSHB", "-SHRAB", "+SHRAB", "-SHSNB",
   "+SHSNB", "-SHPEB", "+SHPEB",
   "-SHGSB", "+SHGSB", "-FZDZB", "+FZDZB",
   "-FZRAB", "+FZRAB", "FZFGB",
   "+FZFGB", "BRB", "FGB", "VCFGB", "MIFGB",
   "PRFGB", "BCFGB", "FUB",
   "VAB", "DUB", "DRDUB", "BLDUB", "SAB",
   "DRSAB", "BLSAB", "HZB",
   "BLPYB", "BLSNB", "+BLSNB", "VCBLSNB",
   "BLSAB", "+BLSAB",
   "VCBLSAB", "+BLDUB", "VCBLDUB", "POB",
   "VCPOB", "SQB", "FCB", "+FCB",
   "VCFCB", "SSB", "+SSB", "VCSSB", "DSB",
   "+DSB", "VCDSB",
 
 
   "FCE", "-DZE", "DZE", "+DZE",
   "FZDZE", "-RAE", "RAE", "+RAE",
   "SHRAE", "TSRAE", "FZRAE", "-SNE",
   "SNE", "+SNE", "DRSNE", "BLSNE",
   "SHSNE", "TSSNE", "-SGE", "SGE",
   "+SGE", "ICE", "-PEE", "PEE", "+PEE",
   "SHPEE", "TSPEE", "GRE", "SHGRE",
   "TSGRE", "GSE", "SHGSE", "TSGSE", "-GSE",
   "+GSE", "TSE", "VCTSE", "-TSRAE",
   "TSRAE", "+TSRAE", "-TSSNE", "TSSNE",
   "+TSSNE", "-TSPEE", "TSPEE", "+TSPEE",
   "-TSGSE", "TSGSE", "+TSGSE",
   "VCSHE", "-SHRAE", "+SHRAE", "-SHSNE",
   "+SHSNE", "-SHPEE", "+SHPEE",
   "-SHGSE", "+SHGSE", "-FZDZE", "+FZDZE",
   "-FZRAE", "+FZRAE", "FZFGE",
   "+FZFGE", "BRE", "FGE", "VCFGE", "MIFGE",
   "PRFGE", "BCFGE", "FUE",
   "VAE", "DUE", "DRDUE", "BLDUE", "SAE",
   "DRSAE", "BLSAE", "HZE",
   "BLPYE", "BLSNE", "+BLSNE", "VCBLSNE",
   "BLSAE", "+BLSAE",
   "VCBLSAE", "+BLDUE", "VCBLDUE", "POE",
   "VCPOE", "SQE", "FCE", "+FCE",
   "VCFCE", "SSE", "+SSE", "VCSSE", "DSE",
   "+DSE", "VCDSE", "4-Zs"};
 
   int i,
       beg_hour,
       beg_min,
       end_hour,
       end_min;
 
   char *temp,
        *free_temp,
        *numb_char,
        *C_char;
 
 
   if( *token == NULL )
      return FALSE;
 
 
   if( (free_temp = temp = (char *) malloc(sizeof(char) *
             (strlen(*token) + 1))) == NULL ) {
      return FALSE;
   }
   else
      strcpy(temp,*token);
 
 
 
   while ( *temp != '\0' ) {
/*
printf("isRecentWX:  JUST inside while-loop, *NDEX = %d\n",*NDEX);
printf("isRecentWX:  JUST inside while-loop, temp = %s\n",temp);
*/
      i = 0;
 
      beg_hour = beg_min = end_hour = end_min = MAXINT;
 
      while( strncmp(temp, phenom[i],strlen(phenom[i])) != 0 &&
                    strcmp(phenom[i],"4-Zs") != 0 )
         i++;
 
      if( strcmp(phenom[i],"4-Zs") != 0 ) {
 
         C_char = (strlen(phenom[i]) - 1) + temp;
         numb_char = C_char + 1;
 
         if( *numb_char == '\0')
            return FALSE;
 
         if( nisdigit(numb_char,4) && strlen(numb_char) >= 4) {
            if( *C_char == 'B' ) {
               beg_hour = antoi( numb_char, 2 );
               beg_min = antoi( numb_char+2,2 );
               temp = numb_char+4;
 
               if( *NDEX < 3 ) {
                  Mptr->ReWx[*NDEX].Bmm = beg_min;
                  Mptr->ReWx[*NDEX].Bhh = beg_hour;
               }
 
               temp = numb_char + 4;
 
               if( *(numb_char+4) == 'E' ) {
                  numb_char += 5;
                  if( nisdigit(numb_char,4) &&
                              strlen(numb_char) >= 4 ) {
                     end_hour = antoi( numb_char, 2 );
                     end_min = antoi( numb_char+2,2 );
                     temp = numb_char+4;
 
                     if( *NDEX < 3 ) {
                        Mptr->ReWx[*NDEX].Emm = end_min;
                        Mptr->ReWx[*NDEX].Ehh = end_hour;
                     }
 
                     temp = numb_char + 4;
 
                  }
                  else if( nisdigit(numb_char,2) &&
                            strlen(numb_char) >= 2 ) {
                     end_min = antoi( numb_char,2 );
 
                     if( *NDEX < 3 ) {
                        Mptr->ReWx[*NDEX].Emm = end_min;
                     }
                     temp = numb_char+2;
                  }
 
               }
 
               if( *NDEX < 3 ) {
                  strncpy(Mptr->ReWx[*NDEX].Recent_weather,
                             phenom[i], (strlen(phenom[i])-1) );
                  (*NDEX)++;
               }
               if( *temp == '\0' ) {
                  free( free_temp );
                  return TRUE;
               }
 
            }
            else {
               end_hour = antoi( numb_char, 2 );
               end_min = antoi( numb_char+2,2 );
 
               temp = numb_char + 4;
 
               if( *NDEX < 3 ) {
                  Mptr->ReWx[*NDEX].Emm = end_min;
                  Mptr->ReWx[*NDEX].Ehh = end_hour;
 
               }
 
               temp = numb_char+4;
 
               if( *(numb_char+4) == 'B' ) {
                  numb_char += 5;
                  if( nisdigit(numb_char,4) &&
                             strlen(numb_char) >= 4 ) {
 
                     beg_hour = antoi(numb_char,2);
                     beg_min  = antoi(numb_char+2,2);
                     temp = numb_char + 4;
 
                     if( *NDEX < 3 ) {
                        Mptr->ReWx[*NDEX].Bmm = beg_min;
                        Mptr->ReWx[*NDEX].Bhh = beg_hour;
 
                     }
 
                     temp = numb_char+4;
                  }
                  else if( nisdigit(numb_char,2) &&
                           strlen(numb_char) >= 2 ) {
                     beg_min = antoi( numb_char,2 );
 
                     if( *NDEX < 3 ) {
                        Mptr->ReWx[*NDEX].Bmm = beg_min;
                     }
 
                     temp = numb_char+2;
                  }
 
               }
 
               if( *NDEX < 3 ) {
                  strncpy(Mptr->ReWx[*NDEX].Recent_weather,
                          phenom[i], (strlen(phenom[i])-1) );
                  (*NDEX)++;
               }
 
               if( *temp == '\0' ) {
                  free( free_temp );
                  return TRUE;
               }
 
            }
 
         }
         else if(nisdigit(numb_char,2) && strlen(numb_char) >= 2 ) {
            if( *C_char == 'B' ) {
               beg_min = antoi( numb_char,2 );
 
               if( *NDEX < 3 ) {
                  Mptr->ReWx[*NDEX].Bmm = beg_min;
 
 
               }
 
               temp = numb_char+2;
 
               if( *(numb_char+2) == 'E' ) {
                  numb_char += 3;
                  if( nisdigit(numb_char,4) &&
                           strlen(numb_char) >= 4 ) {
                     end_hour = antoi( numb_char,2 );
                     end_min = antoi( numb_char+2,2 );
 
                     if( *NDEX < 3 ) {
                        Mptr->ReWx[*NDEX].Emm = end_min;
                        Mptr->ReWx[*NDEX].Ehh = end_hour;
 
 
                     }
 
                     temp = numb_char+4;
                  }
                  else if( nisdigit(numb_char,2) &&
                             strlen(numb_char) >= 2 ) {
                     end_min = antoi( numb_char,2 );
 
                     if( *NDEX < 3 )
                        Mptr->ReWx[*NDEX].Emm = end_min;
 
 
                     temp = numb_char+2;
                  }
               }
               if( *NDEX < 3 ) {
                  strncpy( Mptr->ReWx[*NDEX].Recent_weather,
                           phenom[i], (strlen(phenom[i])-1) );
 
                  (*NDEX)++;
               }
               if( *temp == '\0' ) {
                  free( free_temp );
                  return TRUE;
               }
            }
            else {
               end_min = antoi( numb_char, 2 );
 
               if( *NDEX < 3 )
                  Mptr->ReWx[*NDEX].Emm = end_min;
 
               temp = numb_char+2;
 
               if( *(numb_char+2) == 'B' ) {
                  numb_char += 3;
                  if( nisdigit(numb_char,4) &&
                               strlen(numb_char) >= 4 ) {
                     beg_hour = antoi( numb_char,2 );
                     beg_min = antoi( numb_char+2,2 );
 
                     if( *NDEX < 3 ) {
                        Mptr->ReWx[*NDEX].Bmm = beg_min;
                        Mptr->ReWx[*NDEX].Bhh = beg_hour;
 
                     }
 
                     temp = numb_char+4;
                  }
                  else if( nisdigit(numb_char,2) &&
                             strlen(numb_char) >= 2 ) {
                     beg_min = antoi( numb_char,2 );
 
                     if( *NDEX < 3 )
                        Mptr->ReWx[*NDEX].Bmm = beg_min;
 
 
                     temp = numb_char+2;
                  }
 
               }
               if( *NDEX < 3 ) {
                  strncpy( Mptr->ReWx[*NDEX].Recent_weather,
                           phenom[i], (strlen(phenom[i])-1) );
                  (*NDEX)++;
               }
               if( *temp == '\0' ) {
                  free( free_temp );
                  return TRUE;
               }
 
            }
 
         }
         else {
            free( free_temp );
 
            if( *NDEX > 0 && *NDEX < 3 )
               return TRUE;
            else
               return FALSE;
         }
 
      }
      else {
         free( free_temp );
         return FALSE;
      }
 
   }
 
}
 
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isVariableCIG                                    */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          21 Nov 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:      isVariableCIG determines whether or not the      */
/*                 current group in combination with the next       */
/*                 one or more groups is a report of variable       */
/*                 ceiling.                                         */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 nisdigit                                         */
/*                                                                  */
/*  Input:         token - a pointer to an array of METAR report    */
/*                           groups.                                */
/*                 Mptr - a pointer to a decoded_METAR structure    */
/*                 NDEX - the index value of the current METAR      */
/*                        report group array element.               */
/*                                                                  */
/*  Output:        TRUE, if the token is currently pointing to      */
/*                 METAR report group(s) that a report of vari-     */
/*                 ble ceiling.                                     */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
static MDSP_BOOL isVariableCIG( char **token, Decoded_METAR *Mptr,
                           int *NDEX )
{
   char *V_char;
 
   if( *token == NULL )
      return FALSE;
 
   if( strcmp(*token, "CIG") != 0 )
      return FALSE;
   else
      (++token);
 
   if( *token == NULL )
      return FALSE;
 
   if( (V_char = strchr(*token,'V')) != NULL ) {
      if( nisdigit(*token, (V_char - *token)) &&
            nisdigit( V_char+1, strlen(V_char+1)) ) {
         Mptr->minCeiling = antoi(*token, (V_char - *token));
         Mptr->maxCeiling = atoi(V_char+1);
 
         (*NDEX)++;
         (*NDEX)++;
         return TRUE;
      }
      else
         return FALSE;
   }
   else
      return FALSE;
}
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isCeil2ndSite                                    */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          15 Sep 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:      isCeil2ndSite determines whether or not the      */
/*                 current group in combination with the next       */
/*                 one or more groups is a report of a ceiling      */
/*                 at a secondary site.                             */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 nisdigit                                         */
/*                                                                  */
/*  Input:         token - a pointer to an array of METAR report    */
/*                           groups.                                */
/*                 Mptr - a pointer to a decoded_METAR structure    */
/*                 NDEX - the index value of the current METAR      */
/*                        report group array element.               */
/*                                                                  */
/*  Output:        TRUE, if the token is currently pointing to      */
/*                 METAR report group(s) that are reporting         */
/*                 ceiling at a secondary site.                     */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 nisdigit                                         */
/*                                                                  */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
static MDSP_BOOL isCIG2ndSite( char **token, Decoded_METAR *Mptr,
                           int *NDEX)
{
   int CIG2ndSite;
 
   if( (*token) == NULL )
      return FALSE;
 
   if(strcmp(*token,"CIG") != 0)
      return FALSE;
   else
      (++token);
 
   if( (*token) == NULL )
      return FALSE;
 
   if( strlen(*token) != 3 )
      return FALSE;
 
   if( nisdigit(*token,3) )
   {
      CIG2ndSite = atoi(*token ) * 10;
 
      if( strncmp(*(++token),"RY",2) != 0)
         return FALSE;
      else {
         strcpy(Mptr->CIG_2ndSite_LOC, *token );
         Mptr->CIG_2ndSite_Meters = CIG2ndSite;
         (*NDEX)++;
         (*NDEX)++;
         (*NDEX)++;
         return TRUE;
      }
   }
   else
      return FALSE;
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isPRESFR                                         */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          20 Nov 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:      x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Input:         x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
static MDSP_BOOL isPRESFR( char *string, Decoded_METAR *Mptr, int *NDEX)
{
 
   if( string == NULL )
      return FALSE;
 
   if( strcmp(string, "PRESFR") != 0 )
      return FALSE;
   else {
      Mptr->PRESFR = TRUE;
      (*NDEX)++;
      return TRUE;
   }
 
}
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isPRESRR                                         */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          20 Nov 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:      x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Input:         x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
static MDSP_BOOL isPRESRR( char *string, Decoded_METAR *Mptr, int *NDEX)
{
 
   if( string == NULL )
      return FALSE;
 
   if( strcmp(string, "PRESRR") != 0 )
      return FALSE;
   else {
      Mptr->PRESRR = TRUE;
      (*NDEX)++;
      return TRUE;
   }
 
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isSLP                                            */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          15 Sep 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:                                                       */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                                                                  */
/*  Input:         x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
static MDSP_BOOL isSLP( char **token, Decoded_METAR *Mptr, int *NDEX )
{
 
   /***************************/
   /* DECLARE LOCAL VARIABLES */
   /***************************/
 
   int pressure,
       ndex;
 
   /*************************/
   /* BEGIN BODY OF ROUTINE */
   /*************************/
 
   if( *token == NULL )
      return FALSE;
 
   if( strcmp(*token, "SLPNO") == 0 ) {
      Mptr->SLPNO = TRUE;
      (*NDEX)++;
      return TRUE;
   }
 
 
   if( strncmp(*token, "SLP", 3) != 0 )
      return FALSE;
   else
   {
      if( strncmp(*token, "SLP", 3) == 0 &&
                  strcmp(*token,"SLP") != 0 )
      {
         if( nisdigit( *token+3, 3) )
         {
            pressure = atoi(*token+3);
 
            if(pressure >= 550 )
               Mptr->SLP = ((float) pressure)/10. + 900.;
            else
               Mptr->SLP = ((float) pressure)/10. + 1000.;
            (*NDEX)++;
            return TRUE;
         }
         else
            return FALSE;
      }
      else
      {
         (++token);
 
         if( *token == NULL )
            return FALSE;
 
         if( nisdigit( *token, 3) )
         {
            pressure = atoi(*token);
 
            if(pressure >= 550 )
               Mptr->SLP = ((float) pressure)/10. + 900.;
            else
               Mptr->SLP = ((float) pressure)/10. + 1000.;
 
            (*NDEX)++;
            (*NDEX)++;
            return TRUE;
         }
         else
            return FALSE;
      }
 
   }
 
}
#pragma page(1)
static MDSP_BOOL isSectorVsby( char **string, Decoded_METAR *Mptr,
                          int  *NDEX )
{
   /***************************/
   /* DECLARE LOCAL VARIABLES */
   /***************************/
 
   int result,
       tempstrlen = 20;
 
   float vsby;
   char  dd[3],
         temp[20],
         *slash;
 
   /*************************/
   /* START BODY OF ROUTINE */
   /*************************/
 
   if( *string == NULL )
      return FALSE;
 
   memset( dd, '\0', 3 );
 
   if( strcmp(*string, "VIS") != 0 )
      return FALSE;
   else {
      ++string;
 
      if( *string == NULL )
         return FALSE;
 
      if( strncmp(*string,"NE", 2) == 0 )
         strncpy(dd,*string,2);
      else if( strncmp(*string,"SE",2) == 0 )
         strncpy(dd,*string,2);
      else if( strncmp(*string,"NW",2) == 0 )
         strncpy(dd,*string,2);
      else if( strncmp(*string,"SW",2) == 0 )
         strncpy(dd,*string,2);
      else if( strncmp(*string,"N",1) == 0 )
         strncpy(dd,*string,1);
      else if( strncmp(*string,"E",1) == 0 )
         strncpy(dd,*string,1);
      else if( strncmp(*string,"S",1) == 0 )
         strncpy(dd,*string,1);
      else if( strncmp(*string,"W",1) == 0 )
         strncpy(dd,*string,1);
      else
         return FALSE;
 
      (++string);
      if( *string == NULL )
         return FALSE;
/*
printf("DCDMTRMK result = %d\n",
                 strspn(*string,"0123456789/M"));
*/
      if( (result = strspn(*string,"0123456789/M")) == 0 )
         return FALSE;
      else if(nisdigit(*string,result) )
         vsby = antoi(*string,result);
      else if(result >= tempstrlen-1)
         return FALSE;
      else {
         memset( temp, '\0', tempstrlen );
         strncpy(temp, *string, result);
/*
printf("DCDMTRMK temp = %s\n",temp);
*/
         if( strcmp(temp, "M1/4") == 0) {
            strcpy(Mptr->SectorVsby_Dir,dd);
            Mptr->SectorVsby = 0.0;
            (*NDEX)++;
            (*NDEX)++;
            (*NDEX)++;
            return TRUE;
         }
         if( strchr(temp,'M') != NULL )
            return FALSE;
         if( (slash = strchr(temp,'/')) == NULL )
            return FALSE;
         else if(nisdigit(temp,(slash-temp)) &&
                  nisdigit(slash+1,strlen(slash+1)) ) {
            vsby = fracPart(temp);
            if(vsby > 0.875)
               return FALSE;
            else {
               Mptr->SectorVsby = vsby;
               strcpy(Mptr->SectorVsby_Dir,dd);
               (*NDEX)++;
               (*NDEX)++;
               (*NDEX)++;
               return TRUE;
            }
 
         }
 
      }
 
 
      (++string);
      if( *string == NULL ) {
         Mptr->SectorVsby = vsby;
         strcpy(Mptr->SectorVsby_Dir,dd);
         (*NDEX)++;
         (*NDEX)++;
         (*NDEX)++;
         return TRUE;
      }
      else if( (result = strspn(*string,"0123456789/")) == 0 ) {
         Mptr->SectorVsby = vsby;
         strcpy(Mptr->SectorVsby_Dir,dd);
         (*NDEX)++;
         (*NDEX)++;
         (*NDEX)++;
         return TRUE;
      }
      else if( (slash = strchr(*string,'/')) == NULL ) {
         Mptr->SectorVsby = vsby;
         strcpy(Mptr->SectorVsby_Dir,dd);
         (*NDEX)++;
         (*NDEX)++;
         (*NDEX)++;
         return TRUE;
      }
      else {
         if( fracPart(*string) > 0.875 ) {
            Mptr->SectorVsby = vsby;
            strcpy(Mptr->SectorVsby_Dir,dd);
            (*NDEX)++;
            (*NDEX)++;
            (*NDEX)++;
            return TRUE;
         }
         else {
            vsby += fracPart(*string);
            Mptr->SectorVsby = vsby;
            strcpy(Mptr->SectorVsby_Dir,dd);
            (*NDEX)++;
            (*NDEX)++;
            (*NDEX)++;
            (*NDEX)++;
            return TRUE;
         }
 
      }
 
   }
 
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isGR                                             */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          20 Nov 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:      x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Input:         x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
static MDSP_BOOL isGR( char **string, Decoded_METAR *Mptr, int *NDEX)
{
   char *slash;
 
   if( *string == NULL )
      return FALSE;
 
   if( strcmp(*string, "GS") == 0 ) {
      Mptr->GR = TRUE;
      (*NDEX)++;
      return TRUE;
   }
 
 
   if( strcmp(*string, "GR") != 0 )
      return FALSE;
   else {
      (++string);
 
      if( *string == NULL )
         return FALSE;
 
      if( (slash = strchr( *string, '/' )) != NULL ) {
         if( strcmp( *string, "M1/4" ) == 0 ) {
            Mptr->GR_Size = 1./8.;
            Mptr->GR = TRUE;
            (*NDEX)++;
            (*NDEX)++;
            return TRUE;
         }
         else if( nisdigit( *string, (slash - *string) ) &&
               nisdigit( slash+1, strlen(slash+1)) ) {
            Mptr->GR_Size = fracPart( *string );
            Mptr->GR = TRUE;
            (*NDEX)++;
            (*NDEX)++;
            return TRUE;
         }
         else {
            Mptr->GR = TRUE;
            (*NDEX)++;
            return TRUE;
         }
      }
      else if( nisdigit( *string, strlen(*string) ) ) {
         Mptr->GR_Size = antoi( *string, strlen(*string) );
         Mptr->GR = TRUE;
 
         (++string);
 
         if( *string == NULL )
            return FALSE;
 
         if( (slash = strchr( *string, '/' )) != NULL ) {
            if( nisdigit( *string, (slash - *string) ) &&
                     nisdigit( slash+1, strlen(slash+1)) ) {
               Mptr->GR_Size += fracPart( *string );
               (*NDEX)++;
               (*NDEX)++;
               (*NDEX)++;
               return TRUE;
            }
            else {
               (*NDEX)++;
               (*NDEX)++;
               return TRUE;
            }
         }
         else {
            (*NDEX)++;
            (*NDEX)++;
            return TRUE;
         }
      }
      else {
         Mptr->GR = TRUE;
         (*NDEX)++;
         return TRUE;
      }
   }
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isVIRGA                                          */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          20 Nov 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:      x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Input:         x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
static MDSP_BOOL isVIRGA( char **string, Decoded_METAR *Mptr, int *NDEX)
{
 
   if( *string == NULL )
      return FALSE;
 
   if( strcmp(*string, "VIRGA") != 0 )
      return FALSE;
   else {
      Mptr->VIRGA = TRUE;
      (*NDEX)++;
 
      (++string);
 
      if( *string == NULL )
         return FALSE;
 
 
      if( strcmp( *string, "N" ) == 0 ||
          strcmp( *string, "S" ) == 0 ||
          strcmp( *string, "E" ) == 0 ||
          strcmp( *string, "W" ) == 0 ||
          strcmp( *string, "NE" ) == 0 ||
          strcmp( *string, "NW" ) == 0 ||
          strcmp( *string, "SE" ) == 0 ||
          strcmp( *string, "SW" ) == 0    ) {
         strcpy(Mptr->VIRGA_DIR, *string);
         (*NDEX)++;
      }
      return TRUE;
   }
 
}
 
#pragma page(1)
static MDSP_BOOL isSfcObscuration( char *string, Decoded_METAR *Mptr,
                              int *NDEX )
{
   /***************************/
   /* DECLARE LOCAL VARIABLES */
   /***************************/
 
   static char *WxSymbols[] = {"BCFG", "BLDU", "BLSA", "BLPY",
          "DRDU", "DRSA", "DRSN", "DZ", "DS", "FZFG", "FZDZ", "FZRA",
          "FG", "FC", "FU", "GS", "GR", "HZ", "IC", "MIFG",
          "PE", "PO", "RA", "SHRA", "SHSN", "SHPE", "SHGS",
          "SHGR", "SN", "SG", "SQ", "SA", "SS", "TSRA",
          "TSSN", "TSPE", "TSGS", "TSGR", "TS",
          "VCSH", "VCPO", "VCBLDU", "VCBLSA", "VCBLSN",
          "VCFG", "VCFC","VA", NULL};
   int i,
       ndex;
   char *numLoc,
        ww[12],
        *temp;
 
   MDSP_BOOL IS_NOT_FOUND;
 
   /*************************/
   /* START BODY OF ROUTINE */
   /*************************/
 
   if( string == NULL )
      return FALSE;
 
   memset( ww, '\0', sizeof(ww) );
 
   if( strlen(string) < 4 )
      return FALSE;
 
   if( strncmp(string, "-X",2 ) != 0 )
      return FALSE;
 
   if( !(nisdigit(string+(strlen(string)-1), 1)) )
      return FALSE;
   else {
      temp = string + 2;
      strncpy( ww, temp, (strlen(string)-2) );
 
      ndex = 0;
      temp = ww;
      numLoc = temp + (strlen(temp) - 1 );
 
      while( temp < numLoc && ndex < 6 ) {
         i = 0;
 
         IS_NOT_FOUND = TRUE;
 
         while( WxSymbols[i] != NULL && IS_NOT_FOUND ) {
            if( strncmp( WxSymbols[i], temp, strlen(WxSymbols[i]))
                 != 0 )
               i++;
            else
               IS_NOT_FOUND = FALSE;
         }
 
         if( WxSymbols[i] == NULL ) {
            (*NDEX)++;
            return FALSE;
         }
         else {
            strcpy(&(Mptr->SfcObscuration[ndex][0]),WxSymbols[i]);
            temp += strlen(WxSymbols[i]);
            ndex++;
         }
 
      }
 
      if( ndex > 0 ) {
         Mptr->Num8thsSkyObscured = antoi( numLoc,1 );
         (*NDEX)++;
         return TRUE;
      }
      else {
         (*NDEX)++;
         return FALSE;
      }
 
   }
 
}
 
#pragma page(1)
static MDSP_BOOL isCeiling( char *string, Decoded_METAR *Mptr, int *NDEX )
{
 
   if( string == NULL )
      return FALSE;
 
   if( !(strncmp(string,"CIG",3) == 0 && strlen(string) >= 5) )
      return FALSE;
   else {
      if( strcmp(string, "CIGNO") == 0 ) {
         Mptr->CIGNO = TRUE;
         (*NDEX)++;
         return TRUE;
      }
      else if( strlen( string+3 ) == 3 ) {
         if( nisdigit(string+3, strlen(string+3)) &&
                    strlen(string+3) == 3 ) {
            Mptr->Ceiling = atoi(string+3) * 100;
            (*NDEX)++;
            return TRUE;
         }
         else
            return FALSE;
      }
      else if( strlen(string+3) == 4 ) {
         if( *(string+3) == 'E' && nisdigit(string+4,3) ) {
            Mptr->Estimated_Ceiling = antoi(string+4,3) * 100;
            (*NDEX)++;
            return TRUE;
         }
         else
            return FALSE;
      }
      else
         return FALSE;
 
   }
 
}
#pragma page(1)
static MDSP_BOOL isVrbSky( char **string, Decoded_METAR *Mptr, int *NDEX )
{
   static char *cldPtr[] = {"FEW", "SCT", "BKN", "OVC", NULL };
   MDSP_BOOL IS_NOT_FOUND;
   int i;
   char SKY1[ SKY1_len ];
 
 
   if( *string == NULL )
      return FALSE;
 
 
   memset( SKY1, '\0', SKY1_len );
   i = 0;
   IS_NOT_FOUND = TRUE;
 
   while( cldPtr[i] != NULL && IS_NOT_FOUND ) {
#ifdef DEBUGQQ
   printf("isVrbSky: *string = %s cldPtr[%d] = %s\n",
                            *string,i,cldPtr[i]);
#endif
      if( strncmp(*string, cldPtr[i], strlen(cldPtr[i])) != 0 )
         i++;
      else
         IS_NOT_FOUND = FALSE;
   }
 
   if( cldPtr[i] == NULL )
      return FALSE;
   else {
#ifdef DEBUGQQ
   printf("isVrbSky: *string = %s = cldPtr[%d] = %s\n",
                            *string,i,cldPtr[i]);
#endif
      strcpy( SKY1, cldPtr[i] );
 
      (++string);
 
      if( *string == NULL )
         return FALSE;
 
 
      if( strcmp(*string, "V") != 0 )
         return FALSE;
      else {
         (++string);
 
         if( *string == NULL )
            return FALSE;
 
         i = 0;
         IS_NOT_FOUND = TRUE;
         while( cldPtr[i] != NULL && IS_NOT_FOUND ) {
#ifdef DEBUGQQ
   printf("isVrbSky: *string = %s cldPtr[%d] = %s\n",
                            *string,i,cldPtr[i]);
#endif
            if( strncmp(*string, cldPtr[i], strlen(cldPtr[i])) != 0 )
               i++;
            else
               IS_NOT_FOUND = FALSE;
         }
 
         if( cldPtr[i] == NULL ) {
            (*NDEX)++;
            (*NDEX)++;
            return FALSE;
         }
         else {
            if(strlen(SKY1) == 6 ) {
               if( nisdigit(SKY1+3,3)) {
                  strncpy(Mptr->VrbSkyBelow,SKY1,3);
                  strcpy(Mptr->VrbSkyAbove,cldPtr[i]);
                  Mptr->VrbSkyLayerHgt = antoi(SKY1+3,3)*100;
                  (*NDEX)++;
                  (*NDEX)++;
                  (*NDEX)++;
                  return TRUE;
               }
               else {
                  (*NDEX)++;
                  (*NDEX)++;
                  (*NDEX)++;
                  return TRUE;
               }
            }
            else {
               strcpy(Mptr->VrbSkyBelow,SKY1);
               strcpy(Mptr->VrbSkyAbove,cldPtr[i]);
               (*NDEX)++;
               (*NDEX)++;
               (*NDEX)++;
               return TRUE;
            }
 
         }
 
      }
 
   }
 
}
 
#pragma page(1)
static MDSP_BOOL isObscurAloft( char **string, Decoded_METAR *Mptr,
                           int *NDEX )
{
 
   /***************************/
   /* DECLARE LOCAL VARIABLES */
   /***************************/
 
   static char *WxSymbols[] = {"BCFG", "BLDU", "BLSA", "BLPY",
          "DRDU", "DRSA", "DRSN", "DZ", "DS", "FZFG", "FZDZ", "FZRA",
          "FG", "FC", "FU", "GS", "GR", "HZ", "IC", "MIFG",
          "PE", "PO", "RA", "SHRA", "SHSN", "SHPE", "SHGS",
          "SHGR", "SN", "SG", "SQ", "SA", "SS", "TSRA",
          "TSSN", "TSPE", "TSGS", "TSGR", "TS",
          "VCSH", "VCPO", "VCBLDU", "VCBLSA", "VCBLSN",
          "VCFG", "VCFC","VA", NULL};
   int i;
   char *saveTemp,
        *temp;
 
   MDSP_BOOL IS_NOT_FOUND;
 
   /*************************/
   /* START BODY OF ROUTINE */
   /*************************/
 
   if( *string == NULL )
      return FALSE;
 
   saveTemp = temp = *string;
 
   if( *temp == '\0' )
      return FALSE;
 
   while( *temp != '\0' ) {
      i = 0;
 
      IS_NOT_FOUND = TRUE;
 
      while( WxSymbols[i] != NULL && IS_NOT_FOUND ) {
         if( strncmp(temp,WxSymbols[i],strlen(WxSymbols[i])) != 0 )
            i++;
         else
            IS_NOT_FOUND = FALSE;
      }
 
      if( WxSymbols[i] == NULL ) {
         return FALSE;
      }
      else
         temp += strlen(WxSymbols[i]);
   }
 
   (++string);
 
   if( *string == NULL )
      return FALSE;
 
   if( strlen(*string) != 6 )
      return FALSE;
   else {
      if((strncmp(*string,"FEW",3) == 0 ||
          strncmp(*string,"SCT",3) == 0 ||
          strncmp(*string,"BKN",3) == 0 ||
          strncmp(*string,"OVC",3) == 0  ) &&
                 (nisdigit(*string+3,3) &&
                  strcmp(*string+3,"000") != 0  )) {
         strcpy(Mptr->ObscurAloft,saveTemp);
         strncpy(Mptr->ObscurAloftSkyCond, *string,3);
         Mptr->ObscurAloftHgt = atoi(*string+3)*100;
         (*NDEX)++;
         (*NDEX)++;
         (*NDEX)++;
         return TRUE;
      }
      else {
         (*NDEX)++;
         return TRUE;
      }
 
   }
 
}
#pragma page(1)
static MDSP_BOOL isNOSPECI( char *string, Decoded_METAR *Mptr, int *NDEX )
{
 
   if( string == NULL )
      return FALSE;
 
   if( strcmp(string,"NOSPECI") != 0 )
      return FALSE;
   else {
      Mptr->NOSPECI = TRUE;
      (*NDEX)++;
      return TRUE;
   }
}
#pragma page(1)
static MDSP_BOOL isLAST( char *string, Decoded_METAR *Mptr, int *NDEX )
{
 
   if( string == NULL )
      return FALSE;
 
   if( strcmp(string,"LAST") != 0 )
      return FALSE;
   else {
      Mptr->LAST = TRUE;
      (*NDEX)++;
      return TRUE;
   }
}
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isSynopClouds                                    */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          15 Sep 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:                                                       */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                                                                  */
/*  Input:         x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
static MDSP_BOOL isSynopClouds( char *token, Decoded_METAR *Mptr,
                           int *NDEX )
{
 
 
   if( token == NULL )
      return FALSE;
 
   if(strlen(token) != 5)
      return FALSE;
 
   if( *token == '8' &&
       *(token+1) == '/'  &&
       ((*(token+2) <= '9' && *(token+2) >= '0') || *(token+2) == '/')
                          &&
       ((*(token+3) <= '9' && *(token+3) >= '0') || *(token+3) == '/')
                          &&
       ((*(token+4) <= '9' && *(token+4) >= '0') || *(token+4) == '/'))
   {
      strcpy(Mptr->synoptic_cloud_type,token);
 
      Mptr->CloudLow    = *(token+2);
      Mptr->CloudMedium = *(token+3);
      Mptr->CloudHigh   = *(token+4);
 
      (*NDEX)++;
      return TRUE;
   }
   else
      return FALSE;
}
 
#pragma page(1)
static MDSP_BOOL isSNINCR( char **string, Decoded_METAR *Mptr, int *NDEX )
{
 
   char *slash;
 
   if( *string == NULL )
      return FALSE;
 
   if( strcmp( *string, "SNINCR") != 0 )
      return FALSE;
   else {
      (++string);
 
      if( *string == NULL )
         return FALSE;
 
 
      if( (slash = strchr(*string,'/')) == NULL ) {
         (*NDEX)++;
         return FALSE;
      }
      else if( nisdigit (*string,(slash-*string)) &&
                 nisdigit(slash+1,strlen(slash+1)) ) {
         Mptr->SNINCR = antoi(*string,(slash-*string));
         Mptr->SNINCR_TotalDepth = antoi(slash+1,strlen(slash+1));
         (*NDEX)++;
         (*NDEX)++;
         return TRUE;
      }
      else {
         (*NDEX)++;
         return FALSE;
      }
 
   }
 
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isSnowDepth                                      */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          15 Sep 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:                                                       */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                                                                  */
/*  Input:         x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
static MDSP_BOOL isSnowDepth( char *token, Decoded_METAR *Mptr,
                         int *NDEX )
{
 
   if( token == NULL )
      return FALSE;
 
   if(strlen(token) != 5)
      return FALSE;
 
   if( *token == '4' &&
       *(token+1) == '/'  &&
       nisdigit( (token+2),3) )
   {
      strcpy(Mptr->snow_depth_group,token);
      Mptr->snow_depth = antoi(token+2,3);
      (*NDEX)++;
      return TRUE;
   }
   else
      return FALSE;
}
 
#pragma page(1)
static MDSP_BOOL isWaterEquivSnow( char *string,
                               Decoded_METAR *Mptr,
                               int *NDEX )
{
 
   if( string == NULL )
      return FALSE;
 
   if( strlen(string) != 6 )
      return FALSE;
   else if( !(nisdigit(string,6)) )
      return FALSE;
   else if( strncmp(string, "933", 3) != 0 )
      return FALSE;
   else {
      Mptr->WaterEquivSnow = ((float) atoi(string+3))/10.;
      (*NDEX)++;
      return TRUE;
   }
 
}
#pragma page(1)
static MDSP_BOOL isSunshineDur( char *string, Decoded_METAR *Mptr,
                           int *NDEX )
{
 
   if( string == NULL )
      return FALSE;
 
   if( strlen(string) != 5 )
      return FALSE;
   else if( strncmp(string, "98", 2) != 0 )
      return FALSE;
   else if(nisdigit(string+2,3)) {
      Mptr->SunshineDur = atoi(string+2);
      (*NDEX)++;
      return TRUE;
   }
   else if( strncmp(string+2, "///", 3) == 0 ) {
      Mptr->SunSensorOut = TRUE;
      (*NDEX)++;
      return TRUE;
   }
   else
      return FALSE;
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isHourlyPrecip                                   */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          20 Nov 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:      x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Input:         x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
static MDSP_BOOL isHourlyPrecip( char **string, Decoded_METAR *Mptr,
                            int *NDEX)
{
 
 
   if( *string == NULL )
      return FALSE;
 
   if( !(strcmp(*string, "P") == 0 || charcmp(*string, "'P'dddd") ||
                  charcmp(*string, "'P'ddd") ) )
      return FALSE;
   else if( strcmp(*string, "P") != 0 ) {
      if( nisdigit((*string+1), strlen(*string+1)) ) {
         Mptr->hourlyPrecip = ((float)
                                 atoi(*string+1)) * 0.01;
         (*NDEX)++;
         return TRUE;
      }
      else {
         (*NDEX)++;
         return FALSE;
      }
   }
   else {
 
      (++string);
 
      if( *string == NULL )
         return FALSE;
 
 
      if( nisdigit(*string,strlen(*string)) ) {
         Mptr->hourlyPrecip =  ((float)
                        atoi(*string)) * 0.01;
         (*NDEX)++;
         (*NDEX)++;
         return TRUE;
      }
      else {
         (*NDEX)++;
         return FALSE;
      }
   }
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isP6Precip                                       */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          20 Nov 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:      x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Input:         x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
static MDSP_BOOL isP6Precip( char *string, Decoded_METAR *Mptr,
                        int *NDEX )
{
 
   if( string == NULL )
      return FALSE;
 
   if( strlen(string) != 5 )
      return FALSE;
 
 
   if( charcmp(string,"'6'dddd") ||
         charcmp(string,"'6''/''/''/''/'") ) {
      if( strcmp(string+1, "////") == 0 ) {
         Mptr->precip_amt = (float) MAXINT;
         Mptr->Indeterminant3_6HrPrecip = TRUE;
         (*NDEX)++;
         return TRUE;
      }
      else {
         Mptr->precip_amt = ((float) atoi(string+1)) / 100;
         (*NDEX)++;
         return TRUE;
      }
   }
   else
      return FALSE;
 
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isP24Precip                                      */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          20 Nov 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:      x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Input:         x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
static MDSP_BOOL isP24Precip( char *string, Decoded_METAR *Mptr,
                        int *NDEX )
{
 
   if( string == NULL )
      return FALSE;
 
   if( strlen(string) != 5 )
      return FALSE;
 
   if( charcmp(string,"'7'dddd") ||
         charcmp(string,"'7''/''/''/''/'") ) {
      if( strcmp(string+1, "////") == 0 ) {
         Mptr->precip_24_amt = (float) MAXINT;
         Mptr->Indeterminant_24HrPrecip = TRUE;
         (*NDEX)++;
         return TRUE;
      }
      else {
         Mptr->precip_24_amt = ((float) atoi(string+1)) / 100.;
         (*NDEX)++;
         return TRUE;
      }
   }
   else
      return FALSE;
 
}
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isTTdTenths                                      */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          16 Nov 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:      x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Input:         x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
static MDSP_BOOL isTTdTenths( char *token, Decoded_METAR *Mptr, int *NDEX)
{
 
   /***************************/
   /* DECLARE LOCAL VARIABLES */
   /***************************/
 
   MDSP_BOOL returnFlag = FALSE;
   float sign;
 
   if( token == NULL )
      return FALSE;
 
   if( *token != 'T' )
      return FALSE;
   else if( !(strlen(token) == 5 || strlen(token) == 9) )
      return FALSE;
   else
   {
      if( (*(token+1) == '0' || *(token+1) == '1') &&
                 nisdigit(token+2,3) )
      {
         if( *(token+1) == '0' )
            sign = 0.1;
         else
            sign = -0.1;
 
         Mptr->Temp_2_tenths = sign * ((float) antoi(token+2,3));
         returnFlag = TRUE;
      }
      else
        return FALSE;
 
      if( (*(token+5) == '0' || *(token+5) == '1') &&
                 nisdigit(token+6,3) )
      {
         if( *(token+5) == '0' )
            sign = 0.1;
         else
            sign = -0.1;
 
         Mptr->DP_Temp_2_tenths = sign * ((float) atoi(token+6));
         (*NDEX)++;
         return TRUE;
 
      }
      else
      {
         if( returnFlag )
         {
            (*NDEX)++;
            return TRUE;
         }
         else
            return FALSE;
      }
   }
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isMaxTemp                                        */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          15 Sep 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:                                                       */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                                                                  */
/*  Input:         x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
static MDSP_BOOL isMaxTemp(char *string, Decoded_METAR *Mptr, int *NDEX)
{
   char buf[ 6 ];
 
   if( string == NULL )
      return FALSE;
 
   if(strlen(string) != 5 )
      return FALSE;
   else if(*string == '1' && (*(string+1) == '0' ||
                              *(string+1) == '1' ||
                              *(string+1) == '/'   ) &&
          (nisdigit((string+2),3) ||
            strncmp(string+2,"///",3) == 0) )
   {
      if(nisdigit(string+2,3))
      {
         memset(buf,'\0',6);
         strncpy(buf,string+2,3);
         Mptr->maxtemp = ( (float) atoi(buf))/10.;
 
         if( *(string+1) == '1' )
            Mptr->maxtemp *= (-1.0);
 
         (*NDEX)++;
         return TRUE;
      }
      else
      {
         Mptr->maxtemp = (float) MAXINT;
         (*NDEX)++;
         return TRUE;
      }
   }
   else
      return FALSE;
 
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isMinTemp                                        */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          15 Sep 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:                                                       */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                                                                  */
/*  Input:         x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
static MDSP_BOOL isMinTemp(char *string, Decoded_METAR *Mptr, int *NDEX)
{
   char buf[ 6 ];
 
   if( string == NULL )
      return FALSE;
 
   if(strlen(string) != 5 )
      return FALSE;
   else if(*string == '2' && (*(string+1) == '0' ||
                              *(string+1) == '1' ||
                              *(string+1) == '/'   ) &&
          (nisdigit((string+2),3) ||
              strncmp(string+2,"///",3) == 0) )
   {
      if(nisdigit(string+2,3))
      {
         memset(buf,'\0',6);
         strncpy(buf,string+2,3);
         Mptr->mintemp = ( (float) atoi(buf) )/10.;
 
         if( *(string+1) == '1' )
            Mptr->mintemp *= (-1.0);
         (*NDEX)++;
         return TRUE;
      }
      else
      {
         Mptr->mintemp = (float) MAXINT;
         (*NDEX)++;
         return TRUE;
      }
   }
   else
      return FALSE;
 
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isT24MaxMinTemp                                  */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          15 Sep 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:                                                       */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                                                                  */
/*  Input:         x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
static MDSP_BOOL isT24MaxMinTemp( char *string, Decoded_METAR *Mptr,
                             int *NDEX )
{
   char buf[ 6 ];
 
   if( string == NULL )
      return FALSE;
 
   if( strlen(string) != 9 )
      return FALSE;
   else if( (*string == '4' && (*(string+1) == '0' ||
                                *(string+1) == '1' ||
                                *(string+1) == '/')     &&
             (nisdigit((string+2),3) || strncmp(string+2,"///",3)))
                              &&
             ((*(string+5) == '0' || *(string+5) == '1' ||
              *(string+5) == '/') &&
              (nisdigit((string+6),3) ||
               strncmp(string+6,"///",3) == 0 )) )
   {
      if(nisdigit(string+1,4) && (*(string+1) == '0' ||
                                  *(string+1) == '1')   )
      {
         memset(buf, '\0', 6);
         strncpy(buf, string+2, 3);
         Mptr->max24temp = ( (float) atoi( buf ) )/10.;
 
         if( *(string+1) == '1' )
            Mptr->max24temp *= -1.;
      }
      else
         Mptr->max24temp = (float) MAXINT;
 
 
      if(nisdigit(string+5,4) && (*(string+5) == '0' ||
                                  *(string+5) == '1' )  )
      {
         memset(buf, '\0', 6);
         strncpy(buf, string+6, 3);
         Mptr->min24temp = ( (float) atoi(buf) )/10.;
 
         if( *(string+5) == '1' )
            Mptr->min24temp *= -1.;
      }
      else
         Mptr->min24temp = (float) MAXINT;
 
      (*NDEX)++;
      return TRUE;
 
   }
   else
      return FALSE;
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isPtendency                                      */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          15 Sep 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:                                                       */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                                                                  */
/*  Input:         x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
static MDSP_BOOL isPtendency(char *string, Decoded_METAR *Mptr, int *NDEX)
{
   char buf[ 6 ];
 
   if( string == NULL )
      return FALSE;
 
   if(strlen(string) != 5)
      return FALSE;
   else if(*string == '5' && ('0' <= *(string+1) <= '8') &&
             (nisdigit(string+2,3) || strncmp(string+2,"///",3)
                                             == 0) )
   {
      if( !(nisdigit(string+2,3)) )
      {
         memset(buf,'\0',6);
         strncpy(buf,(string+1),1);
         Mptr->char_prestndcy = atoi(buf);
         (*NDEX)++;
         return TRUE;
      }
      else
      {
         memset(buf,'\0',6);
         strncpy(buf,(string+1),1);
         Mptr->char_prestndcy = atoi(buf);
 
         Mptr->prestndcy = ((float) atoi(string+2)) * 0.1;
 
         (*NDEX)++;
         return TRUE;
      }
 
   }
   else
      return FALSE;
 
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isPWINO                                          */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          20 Nov 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:      x                                                */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                                                                  */
/*  Input:         x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
static MDSP_BOOL isPWINO( char *string, Decoded_METAR *Mptr, int *NDEX)
{
 
   if( string == NULL )
      return FALSE;
 
 
   if( strcmp(string, "PWINO") != 0 )
      return FALSE;
   else {
      Mptr->PWINO = TRUE;
      (*NDEX)++;
      return TRUE;
   }
 
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isPNO                                            */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          20 Nov 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:      x                                                */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                                                                  */
/*  Input:         x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
static MDSP_BOOL isPNO( char *string, Decoded_METAR *Mptr, int *NDEX)
{
 
 
   if( string == NULL )
      return FALSE;
 
   if( strcmp(string, "PNO") != 0 )
      return FALSE;
   else {
      Mptr->PNO = TRUE;
      (*NDEX)++;
      return TRUE;
   }
 
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isRVRNO                                          */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          20 Nov 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:      x                                                */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                                                                  */
/*  Input:         x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
static MDSP_BOOL isRVRNO( char *string, Decoded_METAR *Mptr, int *NDEX)
{
 
   if( string == NULL )
      return FALSE;
 
   if( strcmp(string, "RVRNO") != 0 )
      return FALSE;
   else {
      Mptr->RVRNO = TRUE;
      (*NDEX)++;
      return TRUE;
   }
 
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isCHINO                                          */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          20 Nov 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:      x                                                */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                                                                  */
/*  Input:         x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
static MDSP_BOOL isCHINO( char **string, Decoded_METAR *Mptr, int *NDEX)
{
 
 
   if( *string == NULL )
      return FALSE;
 
   if( strcmp(*string, "CHINO") != 0 )
      return FALSE;
   else
      string++;
 
   if( *string == NULL )
      return FALSE;
 
   if( strlen(*string) <= 2 ) {
      (*NDEX)++;
      return FALSE;
   }
   else {
      if( strncmp( *string, "RY", 2 ) == 0 &&
            nisdigit(*string+2,strlen(*string+2)) ) {
         Mptr->CHINO = TRUE;
         strcpy(Mptr->CHINO_LOC, *string);
         (*NDEX)++;
         (*NDEX)++;
         return TRUE;
      }
      else {
         (*NDEX)++;
         return FALSE;
      }
   }
 
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isVISNO                                          */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          20 Nov 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:      x                                                */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                                                                  */
/*  Input:         x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
static MDSP_BOOL isVISNO( char **string, Decoded_METAR *Mptr, int *NDEX)
{
 
   if( *string == NULL )
      return FALSE;
 
   if( strcmp(*string, "VISNO") != 0 )
      return FALSE;
   else
      string++;
 
   if( *string == NULL )
      return FALSE;
 
   if( strlen(*string) <= 2 ) {
      (*NDEX)++;
      return FALSE;
   }
   else {
      if( strncmp( *string, "RY", 2 ) == 0 &&
            nisdigit(*string+2,strlen(*string+2))) {
         Mptr->VISNO = TRUE;
         strcpy(Mptr->VISNO_LOC, *string);
         (*NDEX)++;
         (*NDEX)++;
         return TRUE;
      }
      else {
         (*NDEX)++;
         return FALSE;
      }
   }
 
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isFZRANO                                         */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          20 Nov 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:      x                                                */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                                                                  */
/*  Input:         x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
static MDSP_BOOL isFZRANO( char *string, Decoded_METAR *Mptr, int *NDEX)
{
 
 
   if( string == NULL )
      return FALSE;
 
   if( strcmp(string, "FZRANO") != 0 )
      return FALSE;
   else {
      Mptr->FZRANO = TRUE;
      (*NDEX)++;
      return TRUE;
   }
 
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isTSNO                                            */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          20 Nov 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:      x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Input:         x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                 x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
static MDSP_BOOL isTSNO( char *string, Decoded_METAR *Mptr, int *NDEX)
{
 
   if( string == NULL )
      return FALSE;
 
   if( strcmp(string, "TSNO") != 0 )
      return FALSE;
   else {
      Mptr->TSNO = TRUE;
      (*NDEX)++;
      return TRUE;
   }
 
}
 
#pragma subtitle(" ")
#pragma page(1)
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         isDollarSign                                 */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          15 Sep 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:                                                       */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                                                                  */
/*  Input:         x                                                */
/*                                                                  */
/*  Output:        x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
 
static MDSP_BOOL isDollarSign( char *indicator, Decoded_METAR *Mptr,
                              int *NDEX )
{
 
   if( indicator == NULL )
      return FALSE;
 
   if( strcmp(indicator,"$") != 0 )
      return FALSE;
   else
   {
      (*NDEX)++;
      Mptr->DollarSign = TRUE;
      return TRUE;
   }
}
 
#pragma page(1)
#pragma subtitle(" ")
#pragma subtitle("subtitle - description                       ")
/********************************************************************/
/*                                                                  */
/*  Title:         DcdMTRmk                                         */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          15 Sep 1994                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:      DcdMTRmk takes a pointer to a METAR              */
/*                 report and parses/decodes data elements from     */
/*                 the remarks section of the report.               */
/*                                                                  */
/*                                                                  */
/*                                                                  */
/*  External Functions Called:                                      */
/*                 None.                                            */
/*                                                                  */
/*  Input:         token - the address of a pointer to a METAR      */
/*                         report character string.                 */
/*                 Mptr  - a pointer to a structure of the vari-    */
/*                         able type Decoded_METAR.                 */
/*                                                                  */
/*                                                                  */
/*  Output:        x                                                */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
#pragma page(1)
void DcdMTRmk( char **token, Decoded_METAR *Mptr )
{
 
   /***************************/
   /* DECLARE LOCAL VARIABLES */
   /***************************/
 
   int TornadicActvty = 0, A0indicator = 0,
       peakwind = 0, windshift = 0, towerVsby = 0, surfaceVsby = 0,
       variableVsby = 0, LTGfreq = 0,
       TS_LOC = 0,
       recentWX = 0, variableCIG = 0, PRESFR = 0,
       Vsby2ndSite = 0, CIG2ndSite = 0,
       PRESRR = 0, SLP = 0, PartObscur = 0,
       SectorVsby = 0, GR = 0, Virga = 0,
       SfcObscur = 0, Ceiling = 0, VrbSkyCond = 0, ObscurAloft = 0,
       NoSPECI = 0, Last = 0, SynopClouds = 0, Snincr = 0,
       SnowDepth = 0, WaterEquivSnow = 0, SunshineDur = 0,
       hourlyPrecip = 0, P6Precip = 0, P24Precip = 0,
       TTdTenths = 0, MaxTemp = 0, MinTemp = 0, T24MaxMinTemp = 0,
       Ptendency = 0, PWINO = 0,
       FZRANO = 0, TSNO = 0, maintIndicator = 0, CHINO = 0, RVRNO = 0,
       VISNO = 0, PNO = 0, DVR = 0;
 
   int  NDEX,
        ndex,
        i;
   char *slash,
        *tokenX,
        *V_char,
        *temp_token;
 
   MDSP_BOOL extra_token,
        IS_NOT_RMKS;
 
   float T_vsby;
 
   /*************************/
   /* START BODY OF ROUTINE */
   /*************************/
 
   NDEX = 0;
 
   /*************************************************/
   /* LOCATE THE START OF THE METAR REMARKS SECTION */
   /*************************************************/
 
   IS_NOT_RMKS = TRUE;
 
   while( token[ NDEX ] != NULL && IS_NOT_RMKS) {
#ifdef DEBUGZZ
   printf("DcdMTRmk:  token[%d] = %s\n",NDEX,token[NDEX]);
#endif
      if( strcmp(token[ NDEX ], "RMK") != 0 )
         NDEX++;
      else
         IS_NOT_RMKS = FALSE;
   }
 
   /***********************************************/
   /* IF THE METAR REPORT CONTAINS NO REMARKS     */
   /* SECTION, THEN RETURN TO THE CALLING ROUTINE */
   /***********************************************/
 
   if( token[ NDEX ] != NULL ) {
#ifdef DEBUGZZ
   printf("DcdMTRmk:  RMK found, token[%d] = %s\n",
                   NDEX,token[NDEX]);
#endif
      NDEX++;
#ifdef DEBUGZZ
   printf("DcdMTRmk:  Bump NDEX, token[%d] = %s\n",
                   NDEX,token[NDEX]);
#endif
   }
   else {
#ifdef DEBUGZZ
   printf("DcdMTRmk:  No RMK found.  NULL ptr encountered\n");
#endif
      return;
   }
   /*****************************************/
   /* IDENTIFY AND VALIDATE REMARKS SECTION */
   /*   DATA GROUPS FOR PARSING/DECODING    */
   /*****************************************/
 
   while(token[NDEX] != NULL) {
 
#ifdef DEBUGZZ
   printf("DcdMTRmk:  DECODE RMKS: token[%d] = %s\n",NDEX,token[NDEX]);
#endif
 
      isRADAT( &(token[NDEX]), Mptr, &NDEX );
 
      if( isTornadicActiv( &(token[NDEX]), Mptr, &NDEX ) ) {
         TornadicActvty++;
         if( TornadicActvty > 1 ) {
            memset(Mptr->TornadicType,'\0',15);
            memset(Mptr->TornadicLOC,'\0',10);
            memset(Mptr->TornadicDIR,'\0',4);
            Mptr->BTornadicHour = MAXINT;
            Mptr->BTornadicMinute = MAXINT;
            Mptr->ETornadicHour = MAXINT;
            Mptr->ETornadicMinute = MAXINT;
         }
      }
      else if( isA0indicator( token[NDEX], Mptr, &NDEX ) ) {
         A0indicator++;
         if( A0indicator > 1 )
            memset(Mptr->autoIndicator,'\0',5);
      }
      else if( isPeakWind( &(token[NDEX]), Mptr, &NDEX ) ) {
         peakwind++;
         if( peakwind > 1 ) {
            Mptr->PKWND_dir = MAXINT;
            Mptr->PKWND_speed = MAXINT;
            Mptr->PKWND_hour = MAXINT;
            Mptr->PKWND_minute = MAXINT;
         }
      }
      else if( isWindShift( &(token[NDEX]), Mptr, &NDEX ) ) {
         windshift++;
         if( windshift > 1 ) {
            Mptr->WshfTime_hour = MAXINT;
            Mptr->WshfTime_minute = MAXINT;
         }
      }
      else if( isTowerVsby( &(token[NDEX]), Mptr, &NDEX ) ) {
         towerVsby++;
         if( towerVsby > 1 )
            Mptr->TWR_VSBY = (float) MAXINT;
      }
      else if( isSurfaceVsby( &(token[NDEX]), Mptr, &NDEX ) ) {
         surfaceVsby++;
         if( surfaceVsby > 1 )
            Mptr->SFC_VSBY = (float) MAXINT;
      }
      else if( isVariableVsby( &(token[NDEX]), Mptr, &NDEX ) ) {
         variableVsby++;
         if( variableVsby > 1 ) {
            Mptr->minVsby = (float) MAXINT;
            Mptr->maxVsby = (float) MAXINT;
         }
      }
      else if( isVsby2ndSite( &(token[NDEX]), Mptr, &NDEX ) ) {
         Vsby2ndSite++;
         if( Vsby2ndSite > 1 ) {
            Mptr->VSBY_2ndSite = (float) MAXINT;
            memset(Mptr->VSBY_2ndSite_LOC,'\0',10);
         }
      }
      else if( isLTGfreq( &(token[NDEX]), Mptr, &NDEX ) ) {
         LTGfreq++;
         if( LTGfreq > 1 ) {
            Mptr->OCNL_LTG = FALSE;
            Mptr->FRQ_LTG = FALSE;
            Mptr->CNS_LTG = FALSE;
            Mptr->CG_LTG = FALSE;
            Mptr->IC_LTG = FALSE;
            Mptr->CC_LTG = FALSE;
            Mptr->CA_LTG = FALSE;
            Mptr->DSNT_LTG = FALSE;
            Mptr->OVHD_LTG = FALSE;
            Mptr->VcyStn_LTG = FALSE;
            Mptr->LightningVCTS = FALSE;
            Mptr->LightningTS = FALSE;
            memset(Mptr->LTG_DIR,'\0',3 );
         }
      }
      else if( isTS_LOC( &(token[NDEX]), Mptr, &NDEX ) ) {
         TS_LOC++;
         if( TS_LOC > 1 ) {
            memset(Mptr->TS_LOC, '\0', 3);
            memset(Mptr->TS_MOVMNT, '\0', 3);
         }
      }
      else if( isRecentWX( &(token[NDEX]), Mptr, &recentWX ) ) {
         NDEX++;
      }
      else if( isVariableCIG( &(token[NDEX]), Mptr, &NDEX ) ) {
         variableCIG++;
         if( variableCIG > 1) {
            Mptr->minCeiling = MAXINT;
            Mptr->maxCeiling = MAXINT;
         }
      }
      else if( isCIG2ndSite( &(token[NDEX]), Mptr, &NDEX ) ) {
         CIG2ndSite++;
         if( CIG2ndSite > 1) {
            Mptr->CIG_2ndSite_Meters = MAXINT;
            memset( Mptr->CIG_2ndSite_LOC, '\0', 10);
         }
      }
      else if( isPRESFR( token[NDEX], Mptr, &NDEX ) ) {
         PRESFR++;
         if( PRESFR > 1 )
            Mptr->PRESFR = FALSE;
      }
      else if( isPRESRR( token[NDEX], Mptr, &NDEX ) ) {
         PRESRR++;
         if( PRESRR > 1 )
            Mptr->PRESRR = FALSE;
      }
      else if( isSLP( &(token[NDEX]), Mptr, &NDEX ) ) {
         SLP++;
         if( SLP > 1 )
            Mptr->SLP = (float) MAXINT;
      }
      else if( isPartObscur( &(token[NDEX]), Mptr, PartObscur,
               &NDEX ) ) {
         PartObscur++;
         if( PartObscur > 2 ) {
            memset(&(Mptr->PartialObscurationAmt[0][0]), '\0', 7 );
            memset(&(Mptr->PartialObscurationPhenom[0][0]),'\0',12 );
 
            memset(&(Mptr->PartialObscurationAmt[1][0]), '\0', 7 );
            memset(&(Mptr->PartialObscurationPhenom[1][0]),'\0',12 );
         }
      }
      else if( isSectorVsby( &(token[NDEX]), Mptr, &NDEX ) ) {
         SectorVsby++;
         if( SectorVsby > 1 ) {
            Mptr->SectorVsby = (float) MAXINT;
            memset(Mptr->SectorVsby_Dir, '\0', 3);
         }
      }
      else if( isGR( &(token[NDEX]), Mptr, &NDEX ) ) {
         GR++;
         if( GR > 1 ) {
            Mptr->GR_Size = (float) MAXINT;
            Mptr->GR = FALSE;
         }
      }
      else if( isVIRGA( &(token[NDEX]), Mptr, &NDEX ) ) {
         Virga++;
         if( Virga > 1 ) {
            Mptr->VIRGA = FALSE;
            memset(Mptr->VIRGA_DIR, '\0', 3);
         }
      }
      else if( isSfcObscuration( token[NDEX], Mptr, &NDEX ) ) {
         SfcObscur++;
         if( SfcObscur > 1 ) {
            for( i = 0; i < 6; i++ ) {
               memset(&(Mptr->SfcObscuration[i][0]), '\0', 10);
               Mptr->Num8thsSkyObscured = MAXINT;
            }
         }
      }
      else if( isCeiling( token[NDEX], Mptr, &NDEX ) ) {
         Ceiling++;
         if( Ceiling > 1 ) {
            Mptr->CIGNO = FALSE;
            Mptr->Ceiling = MAXINT;
            Mptr->Estimated_Ceiling = FALSE;
         }
      }
      else if( isVrbSky( &(token[NDEX]), Mptr, &NDEX ) ) {
         VrbSkyCond++;
         if( VrbSkyCond > 1 ) {
            memset(Mptr->VrbSkyBelow, '\0', 4);
            memset(Mptr->VrbSkyAbove, '\0', 4);
            Mptr->VrbSkyLayerHgt = MAXINT;
         }
      }
      else if( isObscurAloft( &(token[NDEX]), Mptr, &NDEX ) ) {
         ObscurAloft++;
         if( ObscurAloft > 1 ) {
            Mptr->ObscurAloftHgt = MAXINT;
            memset( Mptr->ObscurAloft, '\0', 12 );
            memset( Mptr->ObscurAloftSkyCond, '\0', 12 );
         }
      }
      else if( isNOSPECI( token[NDEX], Mptr, &NDEX ) ) {
         NoSPECI++;
         if( NoSPECI > 1 )
            Mptr->NOSPECI = FALSE;
      }
      else if( isLAST( token[NDEX], Mptr, &NDEX ) ) {
         Last++;
         if( Last > 1 )
            Mptr->LAST = FALSE;
      }
      else if( isSynopClouds( token[NDEX], Mptr, &NDEX ) ) {
         SynopClouds++;
         if( SynopClouds > 1 ) {
            memset( Mptr->synoptic_cloud_type, '\0', 6 );
            Mptr->CloudLow    = '\0';
            Mptr->CloudMedium = '\0';
            Mptr->CloudHigh   = '\0';
         }
      }
      else if( isSNINCR( &(token[NDEX]), Mptr, &NDEX ) ) {
         Snincr++;
         if( Snincr > 1 ) {
            Mptr->SNINCR = MAXINT;
            Mptr->SNINCR_TotalDepth = MAXINT;
         }
      }
      else if( isSnowDepth( token[NDEX], Mptr, &NDEX ) ) {
         SnowDepth++;
         if( SnowDepth > 1 ) {
            memset( Mptr->snow_depth_group, '\0', 6 );
            Mptr->snow_depth = MAXINT;
         }
      }
      else if( isWaterEquivSnow( token[NDEX], Mptr, &NDEX ) ) {
         WaterEquivSnow++;
         if( WaterEquivSnow > 1 )
            Mptr->WaterEquivSnow = (float) MAXINT;
      }
      else if( isSunshineDur( token[NDEX], Mptr, &NDEX ) ) {
         SunshineDur++;
         if( SunshineDur > 1 ) {
            Mptr->SunshineDur = MAXINT;
            Mptr->SunSensorOut = FALSE;
         }
      }
      else if( isHourlyPrecip( &(token[NDEX]), Mptr, &NDEX ) ) {
         hourlyPrecip++;
         if( hourlyPrecip > 1 )
            Mptr->hourlyPrecip = (float) MAXINT;
      }
      else if( isP6Precip( token[NDEX], Mptr, &NDEX ) ) {
         P6Precip++;
         if( P6Precip > 1 )
            Mptr->precip_amt = (float) MAXINT;
      }
      else if( isP24Precip( token[NDEX], Mptr, &NDEX ) ) {
         P24Precip++;
         if( P24Precip > 1 )
            Mptr->precip_24_amt = (float) MAXINT;
      }
      else  if( isTTdTenths( token[NDEX], Mptr, &NDEX ) ) {
         TTdTenths++;
         if( TTdTenths > 1 ) {
            Mptr->Temp_2_tenths = (float) MAXINT;
            Mptr->DP_Temp_2_tenths = (float) MAXINT;
         }
      }
      else if( isMaxTemp( token[NDEX], Mptr, &NDEX ) ) {
         MaxTemp++;
         if( MaxTemp > 1 )
            Mptr->maxtemp = (float) MAXINT;
      }
      else if( isMinTemp( token[NDEX], Mptr, &NDEX ) ) {
         MinTemp++;
         if( MinTemp > 1 )
            Mptr->mintemp = (float) MAXINT;
      }
      else if( isT24MaxMinTemp( token[NDEX],
                                          Mptr, &NDEX ) ) {
         T24MaxMinTemp++;
         if( T24MaxMinTemp > 1 ) {
            Mptr->max24temp = (float) MAXINT;
            Mptr->min24temp = (float) MAXINT;
         }
      }
      else if( isPtendency( token[NDEX], Mptr, &NDEX ) ) {
         Ptendency++;
         if( Ptendency > 1 ) {
            Mptr->char_prestndcy = MAXINT;
            Mptr->prestndcy = (float) MAXINT;
         }
      }
      else if( isPWINO( token[NDEX], Mptr, &NDEX ) ) {
         PWINO++;
         if( PWINO > 1 )
            Mptr->PWINO = FALSE;
      }
      else if( isFZRANO( token[NDEX], Mptr, &NDEX ) ) {
         FZRANO++;
         if( FZRANO > 1 )
            Mptr->FZRANO = FALSE;
      }
      else if( isTSNO( token[NDEX], Mptr, &NDEX ) ) {
         TSNO++;
         if( TSNO > 1 )
            Mptr->TSNO = FALSE;
      }
      else if( isDollarSign( token[NDEX], Mptr, &NDEX ) ) {
         maintIndicator++;
         if( maintIndicator > 1 )
            Mptr->DollarSign = FALSE;
      }
      else if( isRVRNO( token[NDEX], Mptr, &NDEX ) ) {
         RVRNO++;
         if( RVRNO > 1 )
            Mptr->RVRNO = FALSE;
      }
      else if( isPNO( token[NDEX], Mptr, &NDEX ) ) {
         PNO++;
         if( PNO > 1 )
            Mptr->PNO = FALSE;
      }
      else if( isVISNO( &(token[NDEX]), Mptr, &NDEX ) ) {
         VISNO++;
         if( VISNO > 1 ) {
            Mptr->VISNO = FALSE;
            memset(Mptr->VISNO_LOC, '\0', 6);
         }
      }
      else if( isCHINO( &(token[NDEX]), Mptr, &NDEX ) ) {
         CHINO++;
         if( CHINO > 1 ) {
            Mptr->CHINO = FALSE;
            memset(Mptr->CHINO_LOC, '\0', 6);
         }
      }
      else if( isDVR( token[NDEX], Mptr, &NDEX ) ) {
         DVR++;
         if( DVR > 1 ) {
            Mptr->DVR.Min_visRange = MAXINT;
            Mptr->DVR.Max_visRange = MAXINT;
            Mptr->DVR.visRange = MAXINT;
            Mptr->DVR.vrbl_visRange = FALSE;
            Mptr->DVR.below_min_DVR = FALSE;
            Mptr->DVR.above_max_DVR = FALSE;
         }
      }
      else
         NDEX++;
 
   }
 
   return;
}
