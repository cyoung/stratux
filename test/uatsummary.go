/*
	Copyright (c) 2015-2016 Christopher Young
	Distributable under the terms of The "BSD New"" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	gen_gdl90.go: Input demodulated UAT and 1090ES information, output GDL90. Heartbeat,
	 ownship, status messages, stats collection.
*/

package main

import (
	"../uatparse"
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("%s <uat log>\n", os.Args[0])
		return
	}
	fp, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("can't open '%s'.\n", os.Args[1])
		return
	}
	defer fp.Close()

	reader := bufio.NewReader(fp)

	for {
		buf, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("lost stdin.\n")
			break
		}

		x := strings.Split(buf, ",")

		if len(x) < 2 {
			continue
		}

		uatMsg, err := uatparse.New(x[1])
		if err != nil {
			//			fmt.Printf("err %s\n", err.Error())
			continue
		}

		uatMsg.DecodeUplink()

		/*
			p, _ := uatMsg.GetTextReports()
			for _, r := range p {
				fmt.Printf("!!!!%s!!!!\n", r)
			}
		*/

		fmt.Printf("%s,%f,%f,%d,%d, says: ", x[0], uatMsg.Lat, uatMsg.Lon, uatMsg.RS_Err, uatMsg.SignalStrength)
		types := make(map[string]int)
		for _, uatframe := range uatMsg.Frames {
			if uatframe.Product_id == 413 {
				for _, txt := range uatframe.Text_data {
					txt = strings.Trim(txt, " ")
					if len(txt) == 0 {
						continue
					}
					p := strings.Split(txt, " ")
					thisType := p[0]
					types[thisType]++
				}
			} else {
				if uatframe.Frame_type == 0 { // FIS-B product.
					types[strconv.Itoa(int(uatframe.Product_id))]++
				} else {
					types[strconv.Itoa(int(uatframe.Frame_type))]++
				}
			}
		}

		if len(types) == 0 {
			fmt.Printf("(unimplemented)\n")
		} else {
			for thisType, thisNum := range types {
				fmt.Printf("%s(%d) ", thisType, thisNum)
			}
			fmt.Printf("\n")
			//			fmt.Printf("%s\n", buf)
			//			k, _ := uatMsg.GetTextReports()
			//			fmt.Printf("%v\n", k)
		}
	}
}
