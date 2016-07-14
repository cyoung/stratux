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

#ifndef METARX
#define METARX
 
/********************************************************************/
/*                                                                  */
/*  Title:         METAR H                                          */
/*  Organization:  W/OSO242 - GRAPHICS AND DISPLAY SECTION          */
/*  Date:          19 Jan 1996                                      */
/*  Programmer:    CARL MCCALLA                                     */
/*  Language:      C/370                                            */
/*                                                                  */
/*  Abstract:      METAR Decoder Header File.                       */
/*                                                                  */
/*  Modification History:                                           */
/*                 None.                                            */
/*                                                                  */
/********************************************************************/
 
 
#include "local.h"     /* standard header file */
 

/*********************************************/
/*                                           */
/* RUNWAY VISUAL RANGE STRUCTURE DECLARATION */
/*       AND VARIABLE TYPE DEFINITION        */
/*                                           */
/*********************************************/
 
typedef struct runway_VisRange {
   char runway_designator[6];
   MDSP_BOOL vrbl_visRange;
   MDSP_BOOL below_min_RVR;
   MDSP_BOOL above_max_RVR;
   int  visRange;
   int  Max_visRange;
   int  Min_visRange;
}  Runway_VisRange;
 
/***********************************************/
/*                                             */
/* DISPATCH VISUAL RANGE STRUCTURE DECLARATION */
/*       AND VARIABLE TYPE DEFINITION          */
/*                                             */
/***********************************************/
 
typedef struct dispatch_VisRange {
   MDSP_BOOL vrbl_visRange;
   MDSP_BOOL below_min_DVR;
   MDSP_BOOL above_max_DVR;
   int  visRange;
   int  Max_visRange;
   int  Min_visRange;
}  Dispatch_VisRange;
 
/*****************************************/
/*                                       */
/* CLOUD CONDITION STRUCTURE DECLARATION */
/*      AND VARIABLE TYPE DEFINITION     */
/*                                       */
/*****************************************/
 
typedef struct cloud_Conditions {
   char cloud_type[5];
   char cloud_hgt_char[4];
   char other_cld_phenom[4];
   int  cloud_hgt_meters;
}  Cloud_Conditions;
 
/*****************************************/
/*                                       */
/* WIND GROUP DATA STRUCTURE DECLARATION */
/*      AND VARIABLE TYPE DEFINITION     */
/*                                       */
/*****************************************/
 
typedef struct windstruct {
   char windUnits[ 4 ];
   MDSP_BOOL windVRB;
   int windDir;
   int windSpeed;
   int windGust;
} WindStruct;
 
/*****************************************/
/*                                       */
/* RECENT WX GROUP STRUCTURE DECLARATION */
/*      AND VARIABLE TYPE DEFINITION     */
/*                                       */
/*****************************************/
 
typedef struct recent_wx {
   char Recent_weather[ 5 ];
   int  Bhh;
   int  Bmm;
   int  Ehh;
   int  Emm;
} Recent_Wx;
 
/***************************************/
/*                                     */
/* DECODED METAR STRUCTURE DECLARATION */
/*     AND VARIABLE TYPE DEFINITION    */
/*                                     */
/***************************************/
 
typedef struct decoded_METAR {
   char synoptic_cloud_type[ 6 ];
   char snow_depth_group[ 6 ];
   char codeName[ 6 ];
   char stnid[5];
   char horiz_vsby[5];
   char dir_min_horiz_vsby[3];
   char vsby_Dir[ 3 ];
   char WxObstruct[10][8];
   char autoIndicator[5];
   char VSBY_2ndSite_LOC[10];
   char SKY_2ndSite_LOC[10];
   char SKY_2ndSite[10];
   char SectorVsby_Dir[ 3 ];
   char ObscurAloft[ 12 ];
   char ObscurAloftSkyCond[ 12 ];
   char VrbSkyBelow[ 4 ];
   char VrbSkyAbove[ 4 ];
   char LTG_DIR[ 3 ];
   char CloudLow;
   char CloudMedium;
   char CloudHigh;
   char CIG_2ndSite_LOC[10];
   char VIRGA_DIR[3];
   char TornadicType[15];
   char TornadicLOC[10];
   char TornadicDIR[4];
   char TornadicMovDir[3];
   char CHINO_LOC[6];
   char VISNO_LOC[6];
   char PartialObscurationAmt[2][7];
   char PartialObscurationPhenom[2][12];
   char SfcObscuration[6][10];
   char charPrevailVsby[12];
   char charVertVsby[10];
   char TS_LOC[3];
   char TS_MOVMNT[3];
 
   MDSP_BOOL Indeterminant3_6HrPrecip;
   MDSP_BOOL Indeterminant_24HrPrecip;
   MDSP_BOOL CIGNO;
   MDSP_BOOL SLPNO;
   MDSP_BOOL ACFTMSHP;
   MDSP_BOOL NOSPECI;
   MDSP_BOOL FIRST;
   MDSP_BOOL LAST;
   MDSP_BOOL SunSensorOut;
   MDSP_BOOL AUTO;
   MDSP_BOOL COR;
   MDSP_BOOL NIL_rpt;
   MDSP_BOOL CAVOK;
   MDSP_BOOL RVRNO;
   MDSP_BOOL A_altstng;
   MDSP_BOOL Q_altstng;
   MDSP_BOOL VIRGA;
   MDSP_BOOL VOLCASH;
   MDSP_BOOL GR;
   MDSP_BOOL CHINO;
   MDSP_BOOL VISNO;
   MDSP_BOOL PNO;
   MDSP_BOOL PWINO;
   MDSP_BOOL FZRANO;
   MDSP_BOOL TSNO;
   MDSP_BOOL DollarSign;
   MDSP_BOOL PRESRR;
   MDSP_BOOL PRESFR;
   MDSP_BOOL Wshft_FROPA;
   MDSP_BOOL OCNL_LTG;
   MDSP_BOOL FRQ_LTG;
   MDSP_BOOL CNS_LTG;
   MDSP_BOOL CG_LTG;
   MDSP_BOOL IC_LTG;
   MDSP_BOOL CC_LTG;
   MDSP_BOOL CA_LTG;
   MDSP_BOOL DSNT_LTG;
   MDSP_BOOL AP_LTG;
   MDSP_BOOL VcyStn_LTG;
   MDSP_BOOL OVHD_LTG;
   MDSP_BOOL LightningVCTS;
   MDSP_BOOL LightningTS;
 
   int  TornadicDistance;
   int  ob_hour;
   int  ob_minute;
   int  ob_date;
   int minWnDir;
   int maxWnDir;
   int VertVsby;
   int temp;
   int dew_pt_temp;
   int QFE;
   int hectoPasc_altstng;
   int char_prestndcy;
   int minCeiling;
   int maxCeiling;
   int WshfTime_hour;
   int WshfTime_minute;
   int min_vrbl_wind_dir;
   int max_vrbl_wind_dir;
   int PKWND_dir;
   int PKWND_speed;
   int PKWND_hour;
   int PKWND_minute;
   int SKY_2ndSite_Meters;
   int Ceiling;
   int Estimated_Ceiling;
   int SNINCR;
   int SNINCR_TotalDepth;
   int SunshineDur;
   int ObscurAloftHgt;
   int VrbSkyLayerHgt;
   int Num8thsSkyObscured;
   int CIG_2ndSite_Meters;
   int snow_depth;
   int BTornadicHour;
   int BTornadicMinute;
   int ETornadicHour;
   int ETornadicMinute;
 
 
   float SectorVsby;
   float WaterEquivSnow;
   float VSBY_2ndSite;
   float prevail_vsbySM;
   float prevail_vsbyM;
   float prevail_vsbyKM;
   float prestndcy;
   float precip_amt;
   float precip_24_amt;
   float maxtemp;
   float mintemp;
   float max24temp;
   float min24temp;
   float minVsby;
   float maxVsby;
   float hourlyPrecip;
   float TWR_VSBY;
   float SFC_VSBY;
   float Temp_2_tenths;
   float DP_Temp_2_tenths;
   float SLP;
   float GR_Size;
 
   double inches_altstng;
 
   Runway_VisRange RRVR[12];
   Dispatch_VisRange DVR;
   Recent_Wx ReWx[3];
   WindStruct winData;
   Cloud_Conditions cldTypHgt[6];
 
}  Decoded_METAR;
 
#define MAXWXSYMBOLS 10       /*-- NOT TO EXCEED 10 PRES. WX GRPS --*/
#define MAXTOKENS    500      /*--  RPT NOT TO EXCEED 500 GRPS   --*/
 
 
#endif
