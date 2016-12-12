/*

	icao2reg: Converts a 24-bit numeric value to a tail number of FAA
    or Canadian registry.

	(c) 2016 AvSquirrel

    Permission is hereby granted, free of charge, to any person obtaining a copy
	of this software and associated documentation files (the "Software"), to deal
	in the Software without restriction, including without limitation the rights to
	use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
	of the Software, and to permit persons to whom the Software is furnished to do
	so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all
	copies or substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
	WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
	CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

*/

package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	icao := uint32(0xAC82EC)
	args := os.Args
	if len(args) > 1 {
		code, err := strconv.ParseInt(args[1], 16, 32)
		if err != nil {
			fmt.Printf("Error parsing argument %s. Input should be 24-bit hexadecimal, e.g. 'A00001'\n", args[1])
			fmt.Printf("Showing example decoding for Mode S code %X,\n", icao)
		} else {
			icao = uint32(code)
		}
	} else {
		fmt.Printf("Usage: ./icao2faa [code], where [code] is a 24-bit hexadecimal string, e.g. A00001\n")
		fmt.Printf("Showing example decoding for Mode S code %X,\n", icao)
	}

	tail, valid := icao2reg(icao)

	if valid {
		fmt.Printf("ICAO %X successfully decodes as %s\n", icao, tail)
	} else {
		fmt.Printf("ICAO %X did not successfully decode. Response is `%s`\n", icao, tail)
	}
}

func icao2reg(icao_addr uint32) (string, bool) {
	// Initialize local variables
	base34alphabet := string("ABCDEFGHJKLMNPQRSTUVWXYZ0123456789")
	nationalOffset := uint32(0xA00001) // default is US
	tail := ""
	nation := ""

	// Determine nationality
	if (icao_addr >= 0xA00001) && (icao_addr <= 0xAFFFFF) {
		nation = "US"
	} else if (icao_addr >= 0xC00001) && (icao_addr <= 0xC3FFFF) {
		nation = "CA"
	} else {
		//TODO: future national decoding.
		return "NON-NA", false
	}

	if nation == "CA" { // Canada decoding
		// First, discard addresses that are not assigned to aircraft on the civil registry
		if icao_addr > 0xC0CDF8 {
			//fmt.Printf("%X is a Canada aircraft, but not a CF-, CG-, or CI- registration.\n", icao_addr)
			return "CA-MIL", false
		}

		nationalOffset := uint32(0xC00001)
		serial := int32(icao_addr - nationalOffset)

		// Fifth letter
		e := serial % 26

		// Fourth letter
		d := (serial / 26) % 26

		// Third letter
		c := (serial / 676) % 26 // 676 == 26*26

		// Second letter
		b := (serial / 17576) % 26 // 17576 == 26*26*26

		b_str := "FGI"

		fmt.Printf("B = %d, C = %d, D = %d, E = %d\n", b, c, d, e)
		tail = fmt.Sprintf("C-%c%c%c%c", b_str[b], c+65, d+65, e+65)
	}

	if nation == "US" { // FAA decoding
		// First, discard addresses that are not assigned to aircraft on the civil registry
		if icao_addr > 0xADF7C7 {
			//fmt.Printf("%X is a US aircraft, but not on the civil registry.\n", icao_addr)
			return "US-MIL", false
		}

		serial := int32(icao_addr - nationalOffset)
		// First digit
		a := (serial / 101711) + 1

		// Second digit
		a_remainder := serial % 101711
		b := ((a_remainder + 9510) / 10111) - 1

		// Third digit
		b_remainder := (a_remainder + 9510) % 10111
		c := ((b_remainder + 350) / 951) - 1

		// This next bit is more convoluted. First, figure out if we're using the "short" method of
		// decoding the last two digits (two letters, one letter and one blank, or two blanks).
		// This will be the case if digit "B" or "C" are calculated as negative, or if c_remainder
		// is less than 601.

		c_remainder := (b_remainder + 350) % 951
		var d, e int32

		if (b >= 0) && (c >= 0) && (c_remainder > 600) { // alphanumeric decoding method
			d = 24 + (c_remainder-601)/35
			e = (c_remainder - 601) % 35

		} else { // two-letter decoding method
			if (b < 0) || (c < 0) {
				c_remainder -= 350 // otherwise "  " == 350, "A " == 351, "AA" == 352, etc.
			}

			d = (c_remainder - 1) / 25
			e = (c_remainder - 1) % 25

			if e < 0 {
				d -= 1
				e += 25
			}
		}

		a_char := fmt.Sprintf("%d", a)
		var b_char, c_char, d_char, e_char string

		if b >= 0 {
			b_char = fmt.Sprintf("%d", b)
		}

		if b >= 0 && c >= 0 {
			c_char = fmt.Sprintf("%d", c)
		}

		if d > -1 {
			d_char = string(base34alphabet[d])
			if e > 0 {
				e_char = string(base34alphabet[e-1])
			}
		}

		tail = "N" + a_char + b_char + c_char + d_char + e_char

	}

	return tail, true
}
