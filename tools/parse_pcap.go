/*
	Copyright (c) 2018 Christopher Young
	Distributable under the terms of The "BSD New" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	parse_pcap.go: Parse pcap captures. Harris format.
*/

package main

import (
	"encoding/hex"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("usage: %s <file1.pcap> [file2.pcap] ...\n", os.Args[0])
		return
	}

	for i := 1; i < len(os.Args); i++ {
		err := parseFile(os.Args[i])
		if err != nil {
			fmt.Printf("error parsing file '%s': %s\n", os.Args[i], err.Error())
		}
	}
}

func parseFile(fn string) error {
	file, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer file.Close()

	wiresharkHeader := make([]byte, 24)
	// Read the WireShark header. Don't use any of the data.
	n, err := file.Read(wiresharkHeader)
	if n < 24 || err != nil {
		err = fmt.Errorf("couldn't read WireShark header.")
		return err
	}

	data := make([]byte, 498)
	for {
		n, err = file.Read(data)
		if n < 498 || err != nil {
			return nil
		}
		fisbData := data[70:494]
		// Fake FIS-B station header (copy/pasted from Detroit area transmission).
		fmt.Printf("+3c2643887cdcab80%s;\n", hex.EncodeToString(fisbData))
	}
}
