package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("%s <replay log>\n", os.Args[0])
		return
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("error opening '%s': %s\n", os.Args[1], err.Error())
		return
	}
	rdr := bufio.NewReader(f)
	curTick := int64(0)
	for {
		buf, err := rdr.ReadString('\n')
		if err != nil {
			break
		}
		linesplit := strings.Split(buf, ",")
		if len(linesplit) < 2 { // Blank line or invalid.
			continue
		}
		if linesplit[0] == "START" { // Reset ticker, new start.
			curTick = 0
		} else { // If it's not "START", then it's a tick count.
			i, err := strconv.ParseInt(linesplit[0], 10, 64)
			if err != nil {
				fmt.Printf("invalid tick: '%s'\n", linesplit[0])
				continue
			}
			thisWait := i - curTick
			time.Sleep(time.Duration(thisWait) * time.Nanosecond) // Just in case the units change.
			p := strings.Trim(linesplit[1], " ;\r\n")
			fmt.Printf("%s;\n", p)
			curTick = i
		}
	}
}
