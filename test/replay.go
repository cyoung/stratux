package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var uatDone bool
var esDone bool

func uatReplay(f *os.File, replaySpeed uint64) {
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
				fmt.Fprintf(os.Stderr, "invalid tick: '%s'\n", linesplit[0])
				continue
			}
			thisWait := (i - curTick) / int64(replaySpeed)

			if thisWait >= 120000000000 { // More than 2 minutes wait, skip ahead.
				fmt.Fprintf(os.Stderr, "UAT skipahead - %d seconds.\n", thisWait/1000000000)
			} else {
				time.Sleep(time.Duration(thisWait) * time.Nanosecond) // Just in case the units change.
			}

			p := strings.Trim(linesplit[1], " ;\r\n")
			fmt.Printf("%s;\n", p)
			curTick = i
		}
	}
	uatDone = true
}

var connections map[string]net.Conn

func sendToClients(msg string) {
	for addrStr, conn := range connections {
		_, err := conn.Write([]byte(msg))
		if err != nil {
			delete(connections, addrStr)
			fmt.Fprintf(os.Stderr, "disconnected: %s\n", addrStr)
		}
	}
}

func esListener(l net.Listener) {
	defer l.Close()
	connections = make(map[string]net.Conn)
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accepting: %s\n", err.Error())
			continue
		}
		fmt.Fprintf(os.Stderr, "new connection: %s\n", conn.RemoteAddr().String())
		connections[conn.RemoteAddr().String()] = conn
	}
}

func esReplay(f *os.File, replaySpeed uint64) {
	l, err := net.Listen("tcp", "0.0.0.0:30003")
	if err != nil {
		esDone = true
		fmt.Fprintf(os.Stderr, "couldn't open :30003: %s\n", err.Error())
		return
	}

	go esListener(l)

	rdr := bufio.NewReader(f)
	curTick := int64(0)
	for {
		buf, err := rdr.ReadString('\n')
		if err != nil {
			break
		}
		buf = strings.Trim(buf, "\r\n")
		linesplit := strings.Split(buf, ",")
		if len(linesplit) < 2 { // Blank line or invalid.
			continue
		}
		if linesplit[0] == "START" { // Reset ticker, new start.
			curTick = 0
		} else { // If it's not "START", then it's a tick count.
			i, err := strconv.ParseInt(linesplit[0], 10, 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "invalid tick: '%s'\n", linesplit[0])
				continue
			}
			thisWait := (i - curTick) / int64(replaySpeed)

			if thisWait >= 120000000000 { // More than 2 minutes wait, skip ahead.
				fmt.Fprintf(os.Stderr, "ES skipahead - %d seconds.\n", thisWait/1000000000)
			} else {
				time.Sleep(time.Duration(thisWait) * time.Nanosecond) // Just in case the units change.
			}

			p := strings.Join(linesplit[1:], ",")
			p = p + "\n"
			sendToClients(p)

			curTick = i
		}
	}
	esDone = true
}

func openFile(fn string) *os.File {
	f, err := os.Open(fn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening '%s': %s\n", fn, err.Error())
		os.Exit(1)
		return nil
	}
	return f
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "%s <uat replay log> <es replay log> [speed multiplier]\n", os.Args[0])
		return
	}
	f := openFile(os.Args[1])
	f2 := openFile(os.Args[2])
	replaySpeed := uint64(1)
	if len(os.Args) >= 4 {
		i, err := strconv.ParseUint(os.Args[3], 10, 64)
		if err == nil {
			replaySpeed = i
		}
	}
	fmt.Fprintf(os.Stderr, "Replay speed: %dx\n", replaySpeed)
	go uatReplay(f, replaySpeed)
	go esReplay(f2, replaySpeed)
	for {
		time.Sleep(1 * time.Second)
		if uatDone && esDone {
			return
		}
	}
}
