// Copyright (c) 2015 Joseph D Poirier
// Distributable under the terms of The New BSD License
// that can be found in the LICENSE file.

// Package godump978 wraps libdump978, a 978MHz UAT demodulator.
//
// Build example
//
// dump978.so:
//   $ gcc -c -O2 -g -Wall -Werror -Ifec -fpic -DBUILD_LIB=1 dump978.c fec.c fec/decode_rs_char.c fec/init_rs_char.c
//   $ gcc -shared -lm -o ../libdump978.so dump978.o fec.o decode_rs_char.o init_rs_char.o
//
// dump978 go wrapper:
//   $ go build -o dump978.a dump978.go dump978_exports.go
//
// uat_read executable:
//   $ go build uat_read.go

package godump978

/*
#cgo linux LDFLAGS: -ldump978 -lm
#cgo darwin LDFLAGS: -ldump978 -lm
#cgo windows CFLAGS: -IC:/WINDOWS/system32
#cgo windows LDFLAGS: -L. -ldump978 -LC:/WINDOWS/system32

#include <stdlib.h>
#include <stdint.h>
#include "../dump978/dump978.h"

extern void dump978Cb(char updown, uint8_t *data, int len, int rs_errors, int signal_strength);
static inline CallBack GetGoCb() {
	return (CallBack)dump978Cb;
}
*/
import "C"
import "unsafe"

// Current version.
var PackageVersion = "v0.1"

// InChan is a buffered input channel for raw data.
var InChan = make(chan []byte, 100)

type UserCbT func(C.char, *C.uint8_t, C.int, C.int, C.int)

func init() {
	C.Dump978Init((C.CallBack)(C.GetGoCb()))
}

// ProcessData passes buf (modulated data) to dump978 for demodulation.
func ProcessData(buf []byte) {
	C.process_data((*C.char)(unsafe.Pointer(&buf[0])), C.int(len(buf)))
}

func ProcessDataFromChannel() {
	for {
		inData := <-InChan
		ProcessData(inData)
	}
}
