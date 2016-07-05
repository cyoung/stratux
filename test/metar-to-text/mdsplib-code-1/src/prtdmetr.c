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


void say_text(char *speech, char *text) {
 int i;
 char tmp[2];
 for (i = 0; i < strlen(text); i++) {
  switch(text[i]) {
   case '-':
    strcat(speech, "minus");
    break;
   case '0':
    strcat(speech, "zero");
    break;
   case '1':
    strcat(speech, "one");
    break;
   case '2':
    strcat(speech, "two");
    break;
   case '3':
    strcat(speech, "three");
    break;
   case '4':
    strcat(speech, "four");
    break;
   case '5':
    strcat(speech, "five");
    break;
   case '6':
    strcat(speech, "six");
    break;
   case '7':
    strcat(speech, "seven");
    break;
   case '8':
    strcat(speech, "eight");
    break;
   case '9':
    strcat(speech, "niner");
    break;
   case '@':
    strcat(speech, "at");
    break;
   default:
    sprintf(tmp, "%c", text[i]);
    strcat(speech, tmp);
    break;
  }
  strcat(speech, " ");
 }
}



void sprint_metar (char * string, Decoded_METAR *Mptr)
{
 
   /***************************/
   /* DECLARE LOCAL VARIABLES */
   /***************************/
 
   int i;
   char temp[100];
 
   /*************************/
   /* START BODY OF ROUTINE */
   /*************************/
 

   strcat(string, "ME TAR. ");

   if ( Mptr->stnid[ 0 ] == '\0' ) {
    strcat(string, "Error");
    return;
   }

   for (i = 0; i < strlen(Mptr->stnid); i++) {
     sprintf(temp, "%c ", Mptr->stnid[i]);
     strcat(string, temp);
   }
   strcat(string, ". ");

   if (Mptr->ob_hour != MAXINT && Mptr->ob_minute != MAXINT) {
      if (Mptr->AUTO) {
       strcat(string, "Automated ");
      }
      if (Mptr->COR) {
       strcat(string, "Corrected ");
      }
      strcat(string, "Observation ");
      sprintf(temp, "%d%d", Mptr->ob_hour, Mptr->ob_minute);
      say_text(string, temp);
      strcat(string, "zulu. ");
   }
 
   strcat(string, "Wind ");
   if (Mptr->winData.windDir != MAXINT) {
    sprintf(temp, "%03d", Mptr->winData.windDir);
    say_text(string, temp);
   }

   if ( Mptr->winData.windVRB ) {
      strcat(string, "variable ");
   }
 
  //FIXME.
   if ( Mptr->minWnDir != MAXINT ) {
      sprintf(temp, "%03d", Mptr->minWnDir);
      say_text(string, temp);
   }

   //FIXME.
   if ( Mptr->maxWnDir != MAXINT ) {
      sprintf(temp, "%03d", Mptr->maxWnDir);
      say_text(string, temp);
   }

 
   if ( Mptr->winData.windSpeed != MAXINT ) {
      sprintf(temp, "@%d", Mptr->winData.windSpeed);
      say_text(string, temp);
   } else {
    strcat(string, "calm ");
   }
 
   if ( Mptr->winData.windGust != MAXINT ) {
      strcat(string, "gusting ");
      sprintf(temp, "%d", Mptr->winData.windGust);
      say_text(string, temp);
   }

   strcat(string, ". ");
  
   if ( Mptr->prevail_vsbyM != (float) MAXINT ) {
      strcat(string, "Visibility ");
      sprintf(temp, "%d", (int)Mptr->prevail_vsbyM);
      say_text(string, temp);
      strcat(string, "miles. ");
   }
  
   if ( Mptr->prevail_vsbySM != (float) MAXINT ) {
      strcat(string, "Visibility ");
      sprintf(temp, "%d", (int)Mptr->prevail_vsbySM);
      say_text(string, temp);
      strcat(string, "miles. ");
   }


/*   for ( i = 0; i < 12; i++ )
   {
      if( Mptr->RRVR[i].runway_designator[0] != '\0' ) {
         sprintf(temp, "RUNWAY DESIGNATOR   : %s\n",
                 Mptr->RRVR[i].runway_designator);
         strcat(string, temp);
      }
 
      if( Mptr->RRVR[i].visRange != MAXINT ) {
         sprintf(temp, "R_WAY VIS RANGE (FT): %d\n",
                 Mptr->RRVR[i].visRange);
         strcat(string, temp);
      }
 
      if ( Mptr->RRVR[i].vrbl_visRange ) {
         sprintf(temp, "VRBL VISUAL RANGE   : TRUE\n");
         strcat(string, temp);
      }
 
      if ( Mptr->RRVR[i].below_min_RVR ) {
         sprintf(temp, "BELOW MIN RVR       : TRUE\n");
         strcat(string, temp);
      }
 
      if ( Mptr->RRVR[i].above_max_RVR ) {
         sprintf(temp, "ABOVE MAX RVR       : TRUE\n");
         strcat(string, temp);
      }
 
      if( Mptr->RRVR[i].Max_visRange != MAXINT ) {
         sprintf(temp, "MX R_WAY VISRNG (FT): %d\n",
                 Mptr->RRVR[i].Max_visRange);
         strcat(string, temp);
      }
 
      if( Mptr->RRVR[i].Min_visRange != MAXINT ) {
         sprintf(temp, "MN R_WAY VISRNG (FT): %d\n",
                 Mptr->RRVR[i].Min_visRange);
         strcat(string, temp);
      }
 
   }
 
 
   if( Mptr->DVR.visRange != MAXINT ) {
      sprintf(temp, "DISPATCH VIS RANGE  : %d\n",
              Mptr->DVR.visRange);
      strcat(string, temp);
   }
 
   if ( Mptr->DVR.vrbl_visRange ) {
      sprintf(temp, "VRBL DISPATCH VISRNG: TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->DVR.below_min_DVR ) {
      sprintf(temp, "BELOW MIN DVR       : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->DVR.above_max_DVR ) {
      sprintf(temp, "ABOVE MAX DVR       : TRUE\n");
      strcat(string, temp);
   }
 
   if( Mptr->DVR.Max_visRange != MAXINT ) {
      sprintf(temp, "MX DSPAT VISRNG (FT): %d\n",
              Mptr->DVR.Max_visRange);
      strcat(string, temp);
   }
 
   if( Mptr->DVR.Min_visRange != MAXINT ) {
      sprintf(temp, "MN DSPAT VISRNG (FT): %d\n",
              Mptr->DVR.Min_visRange);
      strcat(string, temp);
   }
 

   i = 0;
   while ( Mptr->WxObstruct[i][0] != '\0' && i < MAXWXSYMBOLS )
   {
      sprintf(temp, "WX/OBSTRUCT VISION  : %s\n",
         Mptr->WxObstruct[i] );
      strcat(string, temp);
      i++;
   }
 
   if ( Mptr->PartialObscurationAmt[0][0] != '\0' ) {
      sprintf(temp, "OBSCURATION AMOUNT  : %s\n",
            &(Mptr->PartialObscurationAmt[0][0]));
      strcat(string, temp);
   }
 
   if ( Mptr->PartialObscurationPhenom[0][0] != '\0' ) {
      sprintf(temp, "OBSCURATION PHENOM  : %s\n",
            &(Mptr->PartialObscurationPhenom[0][0]));
      strcat(string, temp);
   }
 
 
   if ( Mptr->PartialObscurationAmt[1][0] != '\0' ) {
      sprintf(temp, "OBSCURATION AMOUNT  : %s\n",
            &(Mptr->PartialObscurationAmt[1][0]));
      strcat(string, temp);
   }
 
   if ( Mptr->PartialObscurationPhenom[1][0] != '\0' ) {
      sprintf(temp, "OBSCURATION PHENOM  : %s\n",
            &(Mptr->PartialObscurationPhenom[1][0]));
      strcat(string, temp);
   }
 */

   strcat(string, "Sky condition ");

   i = 0;
   while ( Mptr->cldTypHgt[ i ].cloud_type[0] != '\0' &&
                     i < 6 )
   {
      if ( Mptr->cldTypHgt[ i ].cloud_type[0] != '\0' ) {
		  if (strcmp(Mptr->cldTypHgt[ i ].cloud_type, "CLR") == 0) {
			  strcat(string, "clear. ");
		  } else {
			  if (strcmp(Mptr->cldTypHgt[ i ].cloud_type, "FEW") == 0) {
				  strcat(string, "scattered ");
			  }
			  if (strcmp(Mptr->cldTypHgt[ i ].cloud_type, "SCT") == 0) {
				  strcat(string, "scattered ");
			  }
			  if (strcmp(Mptr->cldTypHgt[ i ].cloud_type, "BKN") == 0) {
				  strcat(string, "broken ");
			  }
			  if (strcmp(Mptr->cldTypHgt[ i ].cloud_type, "OVC") == 0) {
				  strcat(string, "overcast ");
			  }
			  //TODO: Others?
		  }
      }
 
      if ( Mptr->cldTypHgt[ i ].cloud_hgt_char[0] != '\0' ) {
		  char thousands[4];
		  char hundreds[4];
		  int height = atoi(Mptr->cldTypHgt[i].cloud_hgt_char);
		  if ((height - (height%10)) > 0) {
			  sprintf(thousands, "%d", height/10);
			  say_text(string, thousands);
			  strcat(string, "thousand ");
		  }
		  if (height%10) {
			  sprintf(hundreds, "%d", height%10);
			  say_text(string, hundreds);
			  strcat(string, "hundred ");
		  }
      }
 
	  strcat(string, ". ");
	  /*
	  //TODO: TCU, Towering Cumulus, etc.
      if ( Mptr->cldTypHgt[ i ].other_cld_phenom[0] != '\0' ) {
         sprintf(temp, "OTHER CLOUD PHENOM  : %s\n",
            Mptr->cldTypHgt[ i ].other_cld_phenom);
         strcat(string, temp);
      }
	  */
 
      i++;
 
   }
 
   if ( Mptr->temp != MAXINT ) {
	   strcat(string, "Temperature ");
	   sprintf(temp, "%d", Mptr->temp);
	   say_text(string, temp);
	   strcat(string, "celsius. ");
   }
 
   if ( Mptr->dew_pt_temp != MAXINT ) {
	   strcat(string, "Dew point ");
	   sprintf(temp, "%d", Mptr->dew_pt_temp);
	   say_text(string, temp);
	   strcat(string, "celsius. ");
   }
 
   if ( Mptr->A_altstng ) {
	   strcat(string, "Altimeter ");
	   sprintf(temp, "%d", (int)Mptr->inches_altstng);
	   say_text(string, temp);
	   strcat(string, ", ");
	   sprintf(temp, "%02d", (int)(100*Mptr->inches_altstng)%100);
	   say_text(string, temp);
	   strcat(string, ". ");
   }
 
 /* 
   if ( Mptr->TornadicType[0] != '\0' ) {
      sprintf(temp, "TORNADIC ACTVTY TYPE: %s\n",
         Mptr->TornadicType );
      strcat(string, temp);
   }
 
   if ( Mptr->BTornadicHour != MAXINT ) {
      sprintf(temp, "TORN. ACTVTY BEGHOUR: %d\n",
         Mptr->BTornadicHour );
      strcat(string, temp);
   }
 
   if ( Mptr->BTornadicMinute != MAXINT ) {
      sprintf(temp, "TORN. ACTVTY BEGMIN : %d\n",
         Mptr->BTornadicMinute );
      strcat(string, temp);
   }
 
   if ( Mptr->ETornadicHour != MAXINT ) {
      sprintf(temp, "TORN. ACTVTY ENDHOUR: %d\n",
         Mptr->ETornadicHour );
      strcat(string, temp);
   }
 
   if ( Mptr->ETornadicMinute != MAXINT ) {
      sprintf(temp, "TORN. ACTVTY ENDMIN : %d\n",
         Mptr->ETornadicMinute );
      strcat(string, temp);
   }
 
   if ( Mptr->TornadicDistance != MAXINT ) {
      sprintf(temp, "TORN. DIST. FROM STN: %d\n",
         Mptr->TornadicDistance );
      strcat(string, temp);
   }
 
   if ( Mptr->TornadicLOC[0] != '\0' ) {
      sprintf(temp, "TORNADIC LOCATION   : %s\n",
         Mptr->TornadicLOC );
      strcat(string, temp);
   }
 
   if ( Mptr->TornadicDIR[0] != '\0' ) {
      sprintf(temp, "TORNAD. DIR FROM STN: %s\n",
         Mptr->TornadicDIR );
      strcat(string, temp);
   }
 
   if ( Mptr->TornadicMovDir[0] != '\0' ) {
      sprintf(temp, "TORNADO DIR OF MOVM.: %s\n",
         Mptr->TornadicMovDir );
      strcat(string, temp);
   }
 
 
   if ( Mptr->autoIndicator[0] != '\0' ) {
         sprintf(temp, "AUTO INDICATOR      : %s\n",
                          Mptr->autoIndicator);
      strcat(string, temp);
   }
 
   if ( Mptr->PKWND_dir !=  MAXINT ) {
      sprintf(temp, "PEAK WIND DIRECTION : %d\n",Mptr->PKWND_dir);
      strcat(string, temp);
   }
   if ( Mptr->PKWND_speed !=  MAXINT ) {
      sprintf(temp, "PEAK WIND SPEED     : %d\n",Mptr->PKWND_speed);
      strcat(string, temp);
   }
   if ( Mptr->PKWND_hour !=  MAXINT ) {
      sprintf(temp, "PEAK WIND HOUR      : %d\n",Mptr->PKWND_hour);
      strcat(string, temp);
   }
   if ( Mptr->PKWND_minute !=  MAXINT ) {
      sprintf(temp, "PEAK WIND MINUTE    : %d\n",Mptr->PKWND_minute);
      strcat(string, temp);
   }
 
   if ( Mptr->WshfTime_hour != MAXINT ) {
      sprintf(temp, "HOUR OF WIND SHIFT  : %d\n",Mptr->WshfTime_hour);
      strcat(string, temp);
   }
   if ( Mptr->WshfTime_minute != MAXINT ) {
      sprintf(temp, "MINUTE OF WIND SHIFT: %d\n",Mptr->WshfTime_minute);
      strcat(string, temp);
   }
   if ( Mptr->Wshft_FROPA != FALSE ) {
      sprintf(temp, "FROPA ASSOC. W/WSHFT: TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->TWR_VSBY != (float) MAXINT ) {
      sprintf(temp, "TOWER VISIBILITY    : %.2f\n",Mptr->TWR_VSBY);
      strcat(string, temp);
   }
   if ( Mptr->SFC_VSBY != (float) MAXINT ) {
      sprintf(temp, "SURFACE VISIBILITY  : %.2f\n",Mptr->SFC_VSBY);
      strcat(string, temp);
   }
 
   if ( Mptr->minVsby != (float) MAXINT ) {
      sprintf(temp, "MIN VRBL_VIS (SM)   : %.4f\n",Mptr->minVsby);
      strcat(string, temp);
   }
   if ( Mptr->maxVsby != (float) MAXINT ) {
      sprintf(temp, "MAX VRBL_VIS (SM)   : %.4f\n",Mptr->maxVsby);
      strcat(string, temp);
   }
 
   if( Mptr->VSBY_2ndSite != (float) MAXINT ) {
      sprintf(temp, "VSBY_2ndSite (SM)   : %.4f\n",Mptr->VSBY_2ndSite);
      strcat(string, temp);
   }
   
   if( Mptr->VSBY_2ndSite_LOC[0] != '\0' ) {
      sprintf(temp, "VSBY_2ndSite LOC.   : %s\n",
                   Mptr->VSBY_2ndSite_LOC);
      strcat(string, temp);
   }
 
   if ( Mptr->OCNL_LTG ) {
      sprintf(temp, "OCCASSIONAL LTG     : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->FRQ_LTG ) {
      sprintf(temp, "FREQUENT LIGHTNING  : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->CNS_LTG ) {
      sprintf(temp, "CONTINUOUS LTG      : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->CG_LTG ) {
      sprintf(temp, "CLOUD-GROUND LTG    : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->IC_LTG ) {
      sprintf(temp, "IN-CLOUD LIGHTNING  : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->CC_LTG ) {
      sprintf(temp, "CLD-CLD LIGHTNING   : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->CA_LTG ) {
      sprintf(temp, "CLOUD-AIR LIGHTNING : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->AP_LTG ) {
      sprintf(temp, "LIGHTNING AT AIRPORT: TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->OVHD_LTG ) {
      sprintf(temp, "LIGHTNING OVERHEAD  : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->DSNT_LTG ) {
      sprintf(temp, "DISTANT LIGHTNING   : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->LightningVCTS ) {
      sprintf(temp, "L'NING W/I 5-10(ALP): TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->LightningTS ) {
      sprintf(temp, "L'NING W/I 5 (ALP)  : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->VcyStn_LTG ) {
      sprintf(temp, "VCY STN LIGHTNING   : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->LTG_DIR[0] != '\0' ) {
      sprintf(temp, "DIREC. OF LIGHTNING : %s\n", Mptr->LTG_DIR);
      strcat(string, temp);
   }
 
 
 
   i = 0;
   while( i < 3 && Mptr->ReWx[ i ].Recent_weather[0] != '\0' )
   {
      sprintf(temp, "RECENT WEATHER      : %s",
                  Mptr->ReWx[i].Recent_weather);
      strcat(string, temp);
 
      if ( Mptr->ReWx[i].Bhh != MAXINT ) {
         sprintf(temp, " BEG_hh = %d",Mptr->ReWx[i].Bhh);
         strcat(string, temp);
      }
      if ( Mptr->ReWx[i].Bmm != MAXINT ) {
         sprintf(temp, " BEG_mm = %d",Mptr->ReWx[i].Bmm);
         strcat(string, temp);
      }
 
      if ( Mptr->ReWx[i].Ehh != MAXINT ) {
         sprintf(temp, " END_hh = %d",Mptr->ReWx[i].Ehh);
         strcat(string, temp);
      }
      if ( Mptr->ReWx[i].Emm != MAXINT ) {
         sprintf(temp, " END_mm = %d",Mptr->ReWx[i].Emm);
         strcat(string, temp);
      }
 
      strcat(string, "\n");
 
      i++;
   }
 
   if ( Mptr->minCeiling != MAXINT ) {
      sprintf(temp, "MIN VRBL_CIG (FT)   : %d\n",Mptr->minCeiling);
      strcat(string, temp);
   }
   if ( Mptr->maxCeiling != MAXINT ) {
      sprintf(temp, "MAX VRBL_CIG (FT))  : %d\n",Mptr->maxCeiling);
      strcat(string, temp);
   }
 
   if ( Mptr->CIG_2ndSite_Meters != MAXINT ) {
      sprintf(temp, "CIG2ndSite (FT)     : %d\n",Mptr->CIG_2ndSite_Meters);
      strcat(string, temp);
   }
   if ( Mptr->CIG_2ndSite_LOC[0] != '\0' ) {
      sprintf(temp, "CIG @ 2nd Site LOC. : %s\n",Mptr->CIG_2ndSite_LOC);
      strcat(string, temp);
   }
 
   if ( Mptr->PRESFR ) {
      sprintf(temp, "PRESFR              : TRUE\n");
      strcat(string, temp);
   }
   if ( Mptr->PRESRR ) {
      sprintf(temp, "PRESRR              : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->SLPNO ) {
      sprintf(temp, "SLPNO               : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->SLP != (float) MAXINT ) {
      sprintf(temp, "SLP (hPa)           : %.1f\n", Mptr->SLP);
      strcat(string, temp);
   }
 
   if ( Mptr->SectorVsby != (float) MAXINT ) {
      sprintf(temp, "SECTOR VSBY (MILES) : %.2f\n", Mptr->SectorVsby );
      strcat(string, temp);
   }
 
   if ( Mptr->SectorVsby_Dir[ 0 ] != '\0' ) {
      sprintf(temp, "SECTOR VSBY OCTANT  : %s\n", Mptr->SectorVsby_Dir );
      strcat(string, temp);
   }
 
   if ( Mptr->TS_LOC[ 0 ] != '\0' ) {
      sprintf(temp, "THUNDERSTORM LOCAT. : %s\n", Mptr->TS_LOC );
      strcat(string, temp);
   }
 
   if ( Mptr->TS_MOVMNT[ 0 ] != '\0' ) {
      sprintf(temp, "THUNDERSTORM MOVMNT.: %s\n", Mptr->TS_MOVMNT);
      strcat(string, temp);
   }
 
   if ( Mptr->GR ) {
      sprintf(temp, "GR (HAILSTONES)     : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->GR_Size != (float) MAXINT ) {
      sprintf(temp, "HLSTO SIZE (INCHES) : %.3f\n",Mptr->GR_Size);
      strcat(string, temp);
   }
 
   if ( Mptr->VIRGA ) {
      sprintf(temp, "VIRGA               : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->VIRGA_DIR[0] != '\0' ) {
      sprintf(temp, "DIR OF VIRGA FRM STN: %s\n", Mptr->VIRGA_DIR);
      strcat(string, temp);
   }
 
   for( i = 0; i < 6; i++ ) {
      if( Mptr->SfcObscuration[i][0] != '\0' ) {
         sprintf(temp, "SfcObscuration      : %s\n",
                   &(Mptr->SfcObscuration[i][0]) );
         strcat(string, temp);
      }
   }
 
   if ( Mptr->Num8thsSkyObscured != MAXINT ) {
      sprintf(temp, "8ths of SkyObscured : %d\n",Mptr->Num8thsSkyObscured);
      strcat(string, temp);
   }
 
   if ( Mptr->CIGNO ) {
      sprintf(temp, "CIGNO               : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->Ceiling != MAXINT ) {
      sprintf(temp, "Ceiling (ft)        : %d\n",Mptr->Ceiling);
      strcat(string, temp);
   }
 
   if ( Mptr->Estimated_Ceiling != MAXINT ) {
      sprintf(temp, "Estimated CIG (ft)  : %d\n",Mptr->Estimated_Ceiling);
      strcat(string, temp);
   }
 
   if ( Mptr->VrbSkyBelow[0] != '\0' ) {
      sprintf(temp, "VRB SKY COND BELOW  : %s\n",Mptr->VrbSkyBelow);
      strcat(string, temp);
   }
 
   if ( Mptr->VrbSkyAbove[0] != '\0' ) {
      sprintf(temp, "VRB SKY COND ABOVE  : %s\n",Mptr->VrbSkyAbove);
      strcat(string, temp);
   }
 
   if ( Mptr->VrbSkyLayerHgt != MAXINT ) {
      sprintf(temp, "VRBSKY COND HGT (FT): %d\n",Mptr->VrbSkyLayerHgt);
      strcat(string, temp);
   }
 
   if ( Mptr->ObscurAloftHgt != MAXINT ) {
      sprintf(temp, "Hgt Obscur Aloft(ft): %d\n",Mptr->ObscurAloftHgt);
      strcat(string, temp);
   }
 
   if ( Mptr->ObscurAloft[0] != '\0' ) {
      sprintf(temp, "Obscur Phenom Aloft : %s\n",Mptr->ObscurAloft);
      strcat(string, temp);
   }
 
   if ( Mptr->ObscurAloftSkyCond[0] != '\0' ) {
      sprintf(temp, "Obscur ALOFT SKYCOND: %s\n",Mptr->ObscurAloftSkyCond);
      strcat(string, temp);
   }
 
 
   if ( Mptr->NOSPECI ) {
      sprintf(temp, "NOSPECI             : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->LAST ) {
      sprintf(temp, "LAST                : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->synoptic_cloud_type[ 0 ] != '\0' ) {
      sprintf(temp, "SYNOPTIC CLOUD GROUP: %s\n",Mptr->synoptic_cloud_type);
      strcat(string, temp);
   }
 
   if ( Mptr->CloudLow != '\0' ) {
      sprintf(temp, "LOW CLOUD CODE      : %c\n",Mptr->CloudLow);
      strcat(string, temp);
   }
 
   if ( Mptr->CloudMedium != '\0' ) {
      sprintf(temp, "MEDIUM CLOUD CODE   : %c\n",Mptr->CloudMedium);
      strcat(string, temp);
   }
 
   if ( Mptr->CloudHigh != '\0' ) {
      sprintf(temp, "HIGH CLOUD CODE     : %c\n",Mptr->CloudHigh);
      strcat(string, temp);
   }
 
   if ( Mptr->SNINCR != MAXINT ) {
      sprintf(temp, "SNINCR (INCHES)     : %d\n",Mptr->SNINCR);
      strcat(string, temp);
   }
 
   if ( Mptr->SNINCR_TotalDepth != MAXINT ) {
      sprintf(temp, "SNINCR(TOT. INCHES) : %d\n",Mptr->SNINCR_TotalDepth);
      strcat(string, temp);
   }
 
   if ( Mptr->snow_depth_group[ 0 ] != '\0' ) {
      sprintf(temp, "SNOW DEPTH GROUP    : %s\n",Mptr->snow_depth_group);
      strcat(string, temp);
   }
 
   if ( Mptr->snow_depth != MAXINT ) {
      sprintf(temp, "SNOW DEPTH (INCHES) : %d\n",Mptr->snow_depth);
      strcat(string, temp);
   }
 
   if ( Mptr->WaterEquivSnow != (float) MAXINT ) {
      sprintf(temp, "H2O EquivSno(inches): %.2f\n",Mptr->WaterEquivSnow);
      strcat(string, temp);
   }
 
   if ( Mptr->SunshineDur != MAXINT ) {
      sprintf(temp, "SUNSHINE (MINUTES)  : %d\n",Mptr->SunshineDur);
      strcat(string, temp);
   }
 
   if ( Mptr->SunSensorOut ) {
      sprintf(temp, "SUN SENSOR OUT      : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->hourlyPrecip != (float) MAXINT ) {
      sprintf(temp, "HRLY PRECIP (INCHES): %.2f\n",Mptr->hourlyPrecip);
      strcat(string, temp);
   }
 
   if( Mptr->precip_amt != (float) MAXINT) {
      sprintf(temp, "3/6HR PRCIP (INCHES): %.2f\n",
         Mptr->precip_amt);
      strcat(string, temp);
   }
 
   if( Mptr->Indeterminant3_6HrPrecip ) {
      sprintf(temp, "INDTRMN 3/6HR PRECIP: TRUE\n");
      strcat(string, temp);
   }
 
   if( Mptr->precip_24_amt !=  (float) MAXINT) {
      sprintf(temp, "24HR PRECIP (INCHES): %.2f\n",
         Mptr->precip_24_amt);
      strcat(string, temp);
   }
 
   if ( Mptr->Indeterminant_24HrPrecip ) {
      sprintf(temp, "INDTRMN 24 HR PRECIP: TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->Temp_2_tenths != (float) MAXINT ) {
      sprintf(temp, "TMP2TENTHS (CELSIUS): %.1f\n",Mptr->Temp_2_tenths);
      strcat(string, temp);
   }
 
   if ( Mptr->DP_Temp_2_tenths != (float) MAXINT ) {
      sprintf(temp, "DPT2TENTHS (CELSIUS): %.1f\n",Mptr->DP_Temp_2_tenths);
      strcat(string, temp);
   }
 
   if ( Mptr->maxtemp !=  (float) MAXINT) {
      sprintf(temp, "MAX TEMP (CELSIUS)  : %.1f\n",
         Mptr->maxtemp);
      strcat(string, temp);
   }
 
   if ( Mptr->mintemp !=  (float) MAXINT) {
      sprintf(temp, "MIN TEMP (CELSIUS)  : %.1f\n",
         Mptr->mintemp);
      strcat(string, temp);
   }
 
   if ( Mptr->max24temp !=  (float) MAXINT) {
      sprintf(temp, "24HrMAXTMP (CELSIUS): %.1f\n",
         Mptr->max24temp);
      strcat(string, temp);
   }
 
   if ( Mptr->min24temp !=  (float) MAXINT) {
      sprintf(temp, "24HrMINTMP (CELSIUS): %.1f\n",
         Mptr->min24temp);
      strcat(string, temp);
   }
 
   if ( Mptr->char_prestndcy != MAXINT) {
      sprintf(temp, "CHAR PRESS TENDENCY : %d\n",
         Mptr->char_prestndcy );
      strcat(string, temp);
   }
 
   if ( Mptr->prestndcy != (float) MAXINT) {
      sprintf(temp, "PRES. TENDENCY (hPa): %.1f\n",
         Mptr->prestndcy );
      strcat(string, temp);
   }
 
   if ( Mptr->PWINO ) {
      sprintf(temp, "PWINO               : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->PNO ) {
      sprintf(temp, "PNO                 : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->CHINO ) {
      sprintf(temp, "CHINO               : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->CHINO_LOC[0] != '\0' ) {
      sprintf(temp, "CHINO_LOC           : %s\n",Mptr->CHINO_LOC);
      strcat(string, temp);
   }
 
   if ( Mptr->VISNO ) {
      sprintf(temp, "VISNO               : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->VISNO_LOC[0] != '\0' ) {
      sprintf(temp, "VISNO_LOC           : %s\n",Mptr->VISNO_LOC);
      strcat(string, temp);
   }
 
   if ( Mptr->FZRANO ) {
      sprintf(temp, "FZRANO              : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->TSNO ) {
      sprintf(temp, "TSNO                : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->DollarSign) {
      sprintf(temp, "DOLLAR $IGN INDCATR : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->horiz_vsby[ 0 ] != '\0' ) {
      sprintf(temp, "HORIZ VISIBILITY    : %s\n",Mptr->horiz_vsby);
      strcat(string, temp);
   }
 
   if ( Mptr->dir_min_horiz_vsby[ 0 ] != '\0' ) {
      sprintf(temp, "DIR MIN HORIZ VSBY  : %s\n",Mptr->dir_min_horiz_vsby);
      strcat(string, temp);
   }
 
   if ( Mptr->CAVOK ) {
      sprintf(temp, "CAVOK               : TRUE\n");
      strcat(string, temp);
   }
 
 
   if( Mptr->VertVsby != MAXINT ) {
      sprintf(temp, "Vert. Vsby (meters) : %d\n",
                  Mptr->VertVsby );
      strcat(string, temp);
   }
   */
 
 /*
   if( Mptr->charVertVsby[0] != '\0' )
      sprintf(temp, "Vert. Vsby (CHAR)   : %s\n",
                  Mptr->charVertVsby );
 */
   /*
   if ( Mptr->QFE != MAXINT ) {
      sprintf(temp, "QFE                 : %d\n", Mptr->QFE);
      strcat(string, temp);
   }
 
   if ( Mptr->VOLCASH ) {
      sprintf(temp, "VOLCANIC ASH        : TRUE\n");
      strcat(string, temp);
   }
 
   if ( Mptr->min_vrbl_wind_dir != MAXINT ) {
      sprintf(temp, "MIN VRBL WIND DIR   : %d\n",Mptr->min_vrbl_wind_dir);
      strcat(string, temp);
   }
   if ( Mptr->max_vrbl_wind_dir != MAXINT ) {
      sprintf(temp, "MAX VRBL WIND DIR   : %d\n",Mptr->max_vrbl_wind_dir);
      strcat(string, temp);
   }
 */
 
   strcat(string, "\n\n\n");
}


void prtDMETR (Decoded_METAR *Mptr)
{
	char string[5000];
	
	sprint_metar(string, Mptr);
	printf(string);
}
