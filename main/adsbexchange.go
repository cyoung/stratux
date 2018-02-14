/*
	Copyright (c) 2018 Christopher Young
	Distributable under the terms of The "BSD New" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	adsbexchange.go: adsbexchange 1090ES message transfer and mlat-client running.
*/

package main

import (
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

const (
	RELAY_TYPE_NC          = 1
	RELAY_TYPE_MLAT_CLIENT = 2
)

type RelayProcess struct {
	StartTime time.Time
	RelayType int
	closeCh   chan int
	wg        *sync.WaitGroup
	cmd       []string
	finished  bool
}

var mlatclientRelay *RelayProcess

func (r *RelayProcess) read() {
	defer r.wg.Done()

	// Start the relay process.
	log.Printf("Starting relay process '%s'.\n", r.cmd[0])
	cmd := exec.Command(r.cmd[0], r.cmd[1:]...)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	err := cmd.Start()
	if err != nil {
		log.Printf("Error executing '%s': %s\n", r.cmd[0], err)
		os.Exit(1) //FIXME: Graceful shutdown (or restart).
	}

	done := make(chan bool)

	// Shutdown watcher.
	go func() {
		for {
			select {
			case <-done:
				return
			case <-r.closeCh:
				log.Printf("Relay process '%s' read(): shutdown msg received, calling cmd.Process.Kill() ...\n", r.cmd[0])
				err := cmd.Process.Kill()
				if err == nil {
					log.Println("\t kill successful...")
				}
				return
			default:
				time.Sleep(1 * time.Second)
			}
		}
	}()

	stdoutBuf := make([]byte, 1024)
	stderrBuf := make([]byte, 1024)

	// Stdout watcher.
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				n, err := stdout.Read(stdoutBuf)
				if err == nil && n > 0 {
					//FIXME: Do something with stdout buffer.
					log.Printf("stdout: %s\n", stdoutBuf[:n])
				}
			}
		}
	}()

	// Stderr watcher.
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				n, err := stderr.Read(stderrBuf)
				if err == nil && n > 0 {
					//FIXME: Do something with stderr buffer.
					log.Printf("stderr: %s\n", stderrBuf[:n])

				}
			}
		}
	}()

	cmd.Wait()

	// we get here if A) the process died
	// on its own or B) cmd.Process.Kill() was called
	// from within the goroutine, either way close
	// the "done" channel, which ensures we don't leak
	// goroutines...
	close(done)
	r.finished = true
}

func (r *RelayProcess) shutdown() {
	log.Printf("Shutting down relay process '%s'.\n", r.cmd[0])
	close(r.closeCh)
	r.wg.Wait()
	log.Printf("Finished shutting down relay process '%s'.", r.cmd[0])
}

func makeRelayProc(relayType int, args ...string) *RelayProcess {
	ret := &RelayProcess{StartTime: stratuxClock.Time, RelayType: relayType, cmd: args}
	ret.wg = &sync.WaitGroup{}
	ret.closeCh = make(chan int)
	ret.wg.Add(1)
	go ret.read()
	return ret
}

// /usr/bin/mlat-client --input-type dump1090 --input-connect localhost:30005 --lat $RECEIVERLATITUDE --lon $RECEIVERLONGITUDE --alt $RECEIVERALTITUDE --user $ADSBEXCHANGEUSERNAME --server feed.adsbexchange.com:31090 --no-udp --results beast,connect,localhost:30104
// /bin/nc 127.0.0.1 30005 | /bin/nc feed.adsbexchange.com 30005

func feederProcessMonitor() {
	for {
		time.Sleep(1 * time.Second)
		if len(globalSettings.ADSBExchangeUser) > 0 {
			// Keep adsbexchange processes running.
			//TODO: Check "netcat".

			if mlatclientRelay != nil && mlatclientRelay.finished {
				// Clean up reference.
				mlatclientRelay = nil
			}
			if (mlatclientRelay == nil) && globalStatus.GPS_satellites_locked > 0 && globalStatus.GPS_position_accuracy < 15.0 { //FIXME: 15m accuracy - set a minimum based on network requirements.
				// Check "mlat-client".
				//FIXME: mlatClientRelay never is re-set to nil if it crashes.
				mlatclientRelay = makeRelayProc(RELAY_TYPE_MLAT_CLIENT, "/usr/bin/mlat-client", "--input-type", "dump1090", "--input-connect", "localhost:30005", "--lat", strconv.FormatFloat(float64(mySituation.GPSLatitude), 'f', 5, 32), "--lon", strconv.FormatFloat(float64(mySituation.GPSLongitude), 'f', 5, 32), "--alt", strconv.FormatFloat(float64(mySituation.GPSAltitudeMSL), 'f', 1, 32), "--user", globalSettings.ADSBExchangeUser, "--server", "feed.adsbexchange.com:31090", "--no-udp", "--results", "beast,connect,localhost:30104")
			}
		}
	}
}

func feederInit() {
	go feederProcessMonitor()
}

func relayKill() {
	if mlatclientRelay != nil {
		mlatclientRelay.shutdown()
		mlatclientRelay = nil
	}
}
