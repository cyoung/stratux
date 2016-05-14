package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"time"
)

type dump1090Data struct {
	Icao_addr           uint32
	DF                  int     // Mode S downlink format.
	CA                  int     // Lowest 3 bits of first byte of Mode S message (DF11 and DF17 capability; DF18 control field, zero for all other DF types)
	TypeCode            int     // Mode S type code
	SubtypeCode         int     // Mode S subtype code
	SBS_MsgType         int     // type of SBS message (used in "old" 1090 parsing)
	SignalLevel         float64 // Decimal RSSI (0-1 nominal) as reported by dump1090-mutability. Convert to dB RSSI before setting in TrafficInfo.
	Tail                *string
	Squawk              *int // 12-bit squawk code in octal format
	Emitter_category    *int
	OnGround            *bool
	Lat                 *float32
	Lng                 *float32
	Position_valid      bool
	NACp                *int
	Alt                 *int
	AltIsGNSS           bool   //
	GnssDiffFromBaroAlt *int16 // GNSS height above baro altitude in feet; valid range is -3125 to 3125. +/- 3138 indicates larger difference.
	Vvel                *int16
	Speed_valid         bool
	Speed               *uint16
	Track               *uint16
	Timestamp           time.Time // time traffic last seen, UTC
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("es_dump_csv <sqlite file>\n")
		return
	}
	db, err := sql.Open("sqlite3", os.Args[1])
	if err != nil {
		fmt.Printf("sql.Open(): %s\n", err.Error())
		return
	}
	defer db.Close()
	rows, err := db.Query("SELECT Data FROM es_messages")
	if err != nil {
		fmt.Printf("db.Exec(): %s\n", err.Error())
		return
	}
	defer rows.Close()

	csvOut := make([][]string, 0)

	for rows.Next() {
		var Data string
		if err := rows.Scan(&Data); err != nil {
			fmt.Printf("rows.Scan(): %s\n", err.Error())
			continue
		}
		var d dump1090Data
		err := json.Unmarshal([]byte(Data), &d)
		if err != nil {
			fmt.Printf("json.Unmarshal(): %s\n", err.Error())
			continue
		}

		r := make([]string, 23)
		r[0] = fmt.Sprintf("%06x", d.Icao_addr)
		r[1] = fmt.Sprintf("%d", d.DF)
		r[2] = fmt.Sprintf("%d", d.CA)
		r[3] = fmt.Sprintf("%d", d.TypeCode)
		r[4] = fmt.Sprintf("%d", d.SubtypeCode)
		r[5] = fmt.Sprintf("%d", d.SBS_MsgType)

		r[6] = fmt.Sprintf("%f", d.SignalLevel)
		if d.Tail != nil {
			r[7] = fmt.Sprintf("%s", *d.Tail)
		}
		if d.Squawk != nil {
			r[8] = fmt.Sprintf("%d", *d.Squawk)
		}
		if d.Emitter_category != nil {
			r[9] = fmt.Sprintf("%d", *d.Emitter_category)
		}
		if d.OnGround != nil {
			r[10] = fmt.Sprintf("%t", *d.OnGround)
		}
		if d.Lat != nil {
			r[11] = fmt.Sprintf("%f", *d.Lat)
		}
		if d.Lng != nil {
			r[12] = fmt.Sprintf("%f", *d.Lng)
		}

		r[13] = fmt.Sprintf("%t", d.Position_valid)
		if d.NACp != nil {
			r[14] = fmt.Sprintf("%d", *d.NACp)
		}
		if d.Alt != nil {
			r[15] = fmt.Sprintf("%d", *d.Alt)
		}

		r[16] = fmt.Sprintf("%t", d.AltIsGNSS)
		if d.GnssDiffFromBaroAlt != nil {
			r[17] = fmt.Sprintf("%d", *d.GnssDiffFromBaroAlt)
		}
		if d.Vvel != nil {
			r[18] = fmt.Sprintf("%d", *d.Vvel)
		}
		r[19] = fmt.Sprintf("%t", d.Speed_valid)
		if d.Speed != nil {
			r[20] = fmt.Sprintf("%d", *d.Speed)
		}
		if d.Track != nil {
			r[21] = fmt.Sprintf("%d", *d.Track)
		}
		r[22] = fmt.Sprintf("%s", d.Timestamp)

		csvOut = append(csvOut, r)
	}

	w := csv.NewWriter(os.Stdout)
	w.WriteAll(csvOut)

	if err := rows.Err(); err != nil {
		fmt.Printf("rows.Scan(): %s\n", err.Error())
		return
	}
}
