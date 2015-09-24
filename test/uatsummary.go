package main

import (
	"fmt"
	"../uatparse"
	"os"
	"bufio"
	"strconv"
	"strings"
)



func main() {
	reader := bufio.NewReader(os.Stdin)

	for {
		buf, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("lost stdin.\n")
			break
		}

		uatMsg, err := uatparse.New(buf)
		if err != nil {
			fmt.Printf("err %s\n", err.Error())
			continue
		}

		uatMsg.DecodeUplink()

		/*
		p, _ := uatMsg.GetTextReports()
		for _, r := range p {
			fmt.Printf("!!!!%s!!!!\n", r)
		}
		*/

		fmt.Printf("(%f,%f) says: ", uatMsg.Lat, uatMsg.Lon)
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
			fmt.Printf("nothing\n")
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
