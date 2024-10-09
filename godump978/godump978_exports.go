// Copyright (c) 2015 Joseph D Poirier
// Distributable under the terms of The New BSD License
// that can be found in the LICENSE file.

// Package godump978 wraps libdump978, a 978MHz UAT demodulator.

package godump978

import (
	"fmt"
	"reflect"
	"unsafe"
	"strconv"
)
 
/*
#cgo CFLAGS: -I../

#include <stdint.h>
#include "../dump978/dump978.h"
*/
import "C"

// OutChan is a buffered output channel for demodulated data.
var OutChan = make(chan string, 100)

//export dump978Cb
func dump978Cb(updown C.char, data *C.uint8_t, length C.int, rs_errors C.int, signal_strength C.int) {
	// c buffer to go slice without copying
	var buf []byte

	b := (*reflect.SliceHeader)((unsafe.Pointer(&buf)))
	b.Cap = int(length)
	b.Len = int(length)
	b.Data = uintptr(unsafe.Pointer(data))

	// copy incoming to outgoing
	outData := string(updown)
	for i := 0; i < int(length); i++ {
		outData += fmt.Sprintf("%02x", buf[i])
	}

	outData += ";rs=" + strconv.Itoa(int(rs_errors)) + ";ss=" + strconv.Itoa(int(signal_strength)) + ";"

	OutChan <- outData
}
