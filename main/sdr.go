/*
	Copyright (c) 2015-2016 Christopher Young
	Distributable under the terms of The "BSD New"" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	sdr.go: SDR monitoring, SDR management, data input from UAT/1090ES channels.
*/

package main

import (
	"io"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"../godump978"
	rtl "github.com/jpoirier/gortlsdr"
)

type Device struct {
	dev     *rtl.Context
	wg      *sync.WaitGroup
	closeCh chan int
	indexID int
	ppm     int
	serial  string
	hasID   bool
}

type UAT struct {
	Device
}

type ES struct {
	Device
}

var UATDev *UAT
var ESDev *ES

func readToChan(fp io.ReadCloser, ch chan []byte) {
	for {
		buf := make([]byte, 1024)
		n, err := fp.Read(buf)
		if n > 0 {
			ch <- buf[:n]
		} else if err != nil {
			return
		}
	}
}

func (e *ES) read() {
	defer e.wg.Done()
	log.Println("Entered ES read() ...")
	cmd := exec.Command("/usr/bin/dump1090", "--net", "--device-index", strconv.Itoa(e.indexID), "--ppm", strconv.Itoa(e.ppm))
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	outputChan := make(chan []byte, 1024)

	go readToChan(stdout, outputChan)
	go readToChan(stderr, outputChan)

	err := cmd.Start()
	if err != nil {
		log.Printf("Error executing /usr/bin/dump1090: %s\n", err.Error())
		return
	}
	log.Println("Executed /usr/bin/dump1090 successfully...")

	for {
		select {
		case buf := <-outputChan:
			replayLog(string(buf), MSGCLASS_DUMP1090)
		case <-e.closeCh:
			log.Println("ES read(): shutdown msg received, calling cmd.Process.Kill() ...")
			err := cmd.Process.Kill()
			if err != nil {
				log.Println("\t couldn't kill dump1090: %s", err.Error)
			} else {
				cmd.Wait()
				log.Println("\t kill successful...")
			}
			return
		default:
			time.Sleep(1 * time.Second)
		}
	}
}

func (u *UAT) read() {
	defer u.wg.Done()
	log.Println("Entered UAT read() ...")
	var buffer = make([]uint8, rtl.DefaultBufLength)

	for {
		select {
		default:
			nRead, err := u.dev.ReadSync(buffer, rtl.DefaultBufLength)
			if err != nil {
				//log.Printf("\tReadSync Failed - error: %s\n", err)
				break
			}
			// log.Printf("\tReadSync %d\n", nRead)
			if nRead > 0 {
				buf := buffer[:nRead]
				godump978.InChan <- buf
			}
		case <-u.closeCh:
			log.Println("UAT read(): shutdown msg received...")
			return
		}
	}
}

func getPPM(serial string) int {
	r, err := regexp.Compile("str?a?t?u?x:\\d+:?(-?\\d*)")
	if err != nil {
		return globalSettings.PPM
	}

	arr := r.FindStringSubmatch(serial)
	if arr == nil {
		return globalSettings.PPM
	}

	if ppm, err := strconv.Atoi(arr[1]); err != nil {
		return globalSettings.PPM
	} else {
		return ppm
	}
}

func (e *ES) sdrConfig() (err error) {
	e.ppm = getPPM(e.serial)
	return
}

const (
	TunerGain    = 480
	SampleRate   = 2083334
	NewRTLFreq   = 28800000
	NewTunerFreq = 28800000
	CenterFreq   = 978000000
	Bandwidth    = 1000000
)

func (u *UAT) sdrConfig() (err error) {
	log.Printf("===== UAT Device name: %s =====\n", rtl.GetDeviceName(u.indexID))

	if u.dev, err = rtl.Open(u.indexID); err != nil {
		log.Printf("\tUAT Open Failed...\n")
		return
	}
	log.Printf("\tGetTunerType: %s\n", u.dev.GetTunerType())

	//---------- Set Tuner Gain ----------
	err = u.dev.SetTunerGainMode(true)
	if err != nil {
		u.dev.Close()
		log.Printf("\tSetTunerGainMode Failed - error: %s\n", err)
		return
	} else {
		log.Printf("\tSetTunerGainMode Successful\n")
	}

	err = u.dev.SetTunerGain(TunerGain)
	if err != nil {
		u.dev.Close()
		log.Printf("\tSetTunerGain Failed - error: %s\n", err)
		return
	} else {
		log.Printf("\tSetTunerGain Successful\n")
	}

	//---------- Get/Set Sample Rate ----------
	err = u.dev.SetSampleRate(SampleRate)
	if err != nil {
		u.dev.Close()
		log.Printf("\tSetSampleRate Failed - error: %s\n", err)
		return
	} else {
		log.Printf("\tSetSampleRate - rate: %d\n", SampleRate)
	}
	log.Printf("\tGetSampleRate: %d\n", u.dev.GetSampleRate())

	//---------- Get/Set Xtal Freq ----------
	rtlFreq, tunerFreq, err := u.dev.GetXtalFreq()
	if err != nil {
		u.dev.Close()
		log.Printf("\tGetXtalFreq Failed - error: %s\n", err)
		return
	} else {
		log.Printf("\tGetXtalFreq - Rtl: %d, Tuner: %d\n", rtlFreq, tunerFreq)
	}

	err = u.dev.SetXtalFreq(NewRTLFreq, NewTunerFreq)
	if err != nil {
		u.dev.Close()
		log.Printf("\tSetXtalFreq Failed - error: %s\n", err)
		return
	} else {
		log.Printf("\tSetXtalFreq - Center freq: %d, Tuner freq: %d\n",
			NewRTLFreq, NewTunerFreq)
	}

	//---------- Get/Set Center Freq ----------
	err = u.dev.SetCenterFreq(CenterFreq)
	if err != nil {
		u.dev.Close()
		log.Printf("\tSetCenterFreq 978MHz Failed, error: %s\n", err)
		return
	} else {
		log.Printf("\tSetCenterFreq 978MHz Successful\n")
	}

	log.Printf("\tGetCenterFreq: %d\n", u.dev.GetCenterFreq())

	//---------- Set Bandwidth ----------
	log.Printf("\tSetting Bandwidth: %d\n", Bandwidth)
	if err = u.dev.SetTunerBw(Bandwidth); err != nil {
		u.dev.Close()
		log.Printf("\tSetTunerBw %d Failed, error: %s\n", Bandwidth, err)
		return
	} else {
		log.Printf("\tSetTunerBw %d Successful\n", Bandwidth)
	}

	if err = u.dev.ResetBuffer(); err != nil {
		u.dev.Close()
		log.Printf("\tResetBuffer Failed - error: %s\n", err)
		return
	} else {
		log.Printf("\tResetBuffer Successful\n")
	}

	//---------- Get/Set Freq Correction ----------
	freqCorr := u.dev.GetFreqCorrection()
	log.Printf("\tGetFreqCorrection: %d\n", freqCorr)
	u.ppm = getPPM(u.serial)
	err = u.dev.SetFreqCorrection(u.ppm)
	if err != nil {
		u.dev.Close()
		log.Printf("\tSetFreqCorrection %d Failed, error: %s\n", u.ppm, err)
		return
	} else {
		log.Printf("\tSetFreqCorrection %d Successful\n", u.ppm)
	}
	return
}

// Read from the godump978 channel - on or off.
func uatReader() {
	log.Println("Entered uatReader() ...")
	for {
		uat := <-godump978.OutChan
		o, msgtype := parseInput(uat)
		if o != nil && msgtype != 0 {
			relayMessage(msgtype, o)
		}
	}
}

func (u *UAT) writeID() error {
	info, err := u.dev.GetHwInfo()
	if err != nil {
		return err
	}
	info.Serial = "stratux:978"
	return u.dev.SetHwInfo(info)
}

func (e *ES) writeID() error {
	info, err := e.dev.GetHwInfo()
	if err != nil {
		return err
	}
	info.Serial = "stratux:1090"
	return e.dev.SetHwInfo(info)
}

func (u *UAT) shutdown() {
	log.Println("Entered UAT shutdown() ...")
	close(u.closeCh) // signal to shutdown
	log.Println("UAT shutdown(): calling u.wg.Wait() ...")
	u.wg.Wait() // Wait for the goroutine to shutdown
	log.Println("UAT shutdown(): u.wg.Wait() returned...")
	log.Println("UAT shutdown(): closing device ...")
	u.dev.Close() // preempt the blocking ReadSync call
}

func (e *ES) shutdown() {
	log.Println("Entered ES shutdown() ...")
	close(e.closeCh) // signal to shutdown
	log.Println("ES shutdown(): calling e.wg.Wait() ...")
	e.wg.Wait() // Wait for the goroutine to shutdown
	log.Println("ES shutdown(): e.wg.Wait() returned...")
}

var sdrShutdown bool

func sdrKill() {
	// Send signal to shutdown to sdrWatcher().
	sdrShutdown = true
	// Spin until all devices have been de-initialized.
	for UATDev != nil || ESDev != nil {
		time.Sleep(1 * time.Second)
	}
}

type rUAT regexp.Regexp

func (r *rUAT) hasID(serial string) bool {
	if r == nil {
		return strings.HasPrefix(serial, "stratux:978")
	}
	return r.MatchString(serial)
}

type rES regexp.Regexp

func (r *rES) hasID(serial string) bool {
	if r == nil {
		return strings.HasPrefix(serial, "stratux:1090")
	}
	return r.MatchString(serial)
}

// Watch for config/device changes.
func sdrWatcher() {
	// Compile returns a nil pointer when it fails and
	// we do the nil check in the hasID method
	rES, _ = regexp.Compile("str?a?t?u?x:1090")
	rUAT, _ = regexp.Compile("str?a?t?u?x:978")

	for {
		time.Sleep(1 * time.Second)
		if sdrShutdown {
			if UATDev != nil {
				UATDev.shutdown()
				UATDev = nil
			}
			if ESDev != nil {
				ESDev.shutdown()
				ESDev = nil
			}
			return
		}
		count := rtl.GetDeviceCount()
		atomic.StoreUint32(&globalStatus.Devices, uint32(count))

		// support two and only two dongles
		if count > 2 {
			count = 2
		}

		// cleanup if necessary
		if count < 1 || (!globalSettings.UAT_Enabled && !globalSettings.ES_Enabled) {
			if UATDev != nil {
				UATDev.shutdown()
				UATDev = nil
			}
			if ESDev != nil {
				ESDev.shutdown()
				ESDev = nil
			}
			continue
		}

		if count == 1 {
			if UATDev != nil && ESDev != nil {
				// this is a bit heavy handed but we shouldn't be in this state
				ESDev.shutdown()
				ESDev = nil
				UATDev.shutdown()
				UATDev = nil
			} else if UATDev != nil && ESDev == nil {
				UATDev.indexID = 0
			} else if UATDev == nil && ESDev != nil {
				ESDev.indexID = 0
			}
		}

		// UAT specific handling
		// When count is one, favor UAT in the case where the user
		// has enabled both UAT and ES via the web interface.
		if globalSettings.UAT_Enabled {
			id := 0
			if count == 1 {
				if ESDev != nil {
					ESDev.shutdown()
					ESDev = nil
				}
			} else { // count == 2
				if UATDev == nil && ESDev != nil {
					if ESDev.indexID == 0 {
						id = 1
					}
				}
			}

			if UATDev == nil {
				_, _, serial, err := rtl.GetDeviceUsbStrings(id)
				if err != nil {
					serial = ""
				}

				if !rES.hasID(serial) {
					UATDev = &UAT{indexID: id, serial: serial, wg: &sync.WaitGroup{}}
					if err := UATDev.sdrConfig(); err != nil {
						log.Printf("UATDev = &UAT{indexID: id} failed: %s\n", err)
						UATDev = nil
					} else {
						UATDev.closeCh = make(chan int)
						UATDev.wg.Add(1)
						go UATDev.read()
					}
				}
			}
		} else if UATDev != nil {
			UATDev.shutdown()
			UATDev = nil
			if count == 1 && ESDev != nil {
				ESDev.indexID = 0
			}
		}

		// ES specific handling
		if globalSettings.ES_Enabled {
			id := 0
			if count == 1 {
				if globalSettings.UAT_Enabled {
					// defer to the UAT handler
					goto End
				}
			} else { // count == 2
				if ESDev == nil && UATDev != nil {
					if UATDev.indexID == 0 {
						id = 1
					}
				}
			}

			if ESDev == nil {
				_, _, serial, err := rtl.GetDeviceUsbStrings(id)
				if err != nil {
					serial = ""
				}

				if !rUAT.hasID(serial) {
					ESDev = &ES{indexID: id, serial: serial, wg: &sync.WaitGroup{}}
					if err := ESDev.sdrConfig(); err != nil {
						log.Printf("ESDev = &ES{indexID: id} failed: %s\n", err)
						ESDev = nil
					} else {
						ESDev.closeCh = make(chan int)
						ESDev.wg.Add(1)
						go ESDev.read()
					}
				}
			}
		} else if ESDev != nil {
			ESDev.shutdown()
			ESDev = nil
			if count == 1 && UATDev != nil {
				UATDev.indexID = 0
			}
		}
	End:
	}
}

func sdrInit() {
	go sdrWatcher()
	go uatReader()
	godump978.Dump978Init()
	go godump978.ProcessDataFromChannel()
}
