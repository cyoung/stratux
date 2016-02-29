/*
	Copyright (c) 2015-2016 Christopher Young
	Distributable under the terms of The "BSD New"" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	sdr.go: SDR monitoring, SDR management, data input from UAT/1090ES channels.
*/

package main

import (
	"bufio"
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
	idSet   bool
}

type UAT Device
type ES Device

var UATDev *UAT
var ESDev *ES

func (e *ES) read() {
	defer e.wg.Done()
	log.Println("Entered ES read() ...")
	cmd := exec.Command("/usr/bin/dump1090", "--oversample", "--net", "--device-index", strconv.Itoa(e.indexID), "--ppm", strconv.Itoa(e.ppm))
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	err := cmd.Start()
	if err != nil {
		log.Printf("Error executing /usr/bin/dump1090: %s\n", err)
		return
	}
	log.Println("Executed /usr/bin/dump1090 successfully...")

	scanStdout := bufio.NewScanner(stdout)
	scanStderr := bufio.NewScanner(stderr)

	for {
		select {
		case <-e.closeCh:
			log.Println("ES read(): shutdown msg received, calling cmd.Process.Kill() ...")
			err := cmd.Process.Kill()
			if err != nil {
				log.Printf("\t couldn't kill dump1090: %s\n", err)
			} else {
				cmd.Wait()
				log.Println("\t kill successful...")
			}
			return
		default:
			for scanStdout.Scan() {
				replayLog(scanStdout.Text(), MSGCLASS_DUMP1090)
			}
			if err := scanStdout.Err(); err != nil {
				log.Printf("scanStdout error: %s\n", err)
			}

			for scanStderr.Scan() {
				replayLog(scanStderr.Text(), MSGCLASS_DUMP1090)
				if shutdownES != true {
					shutdownES = true
				}
			}
			if err := scanStderr.Err(); err != nil {
				log.Printf("scanStderr error: %s\n", err)
			}

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
				if globalSettings.DEBUG {
					log.Printf("\tReadSync Failed - error: %s\n", err)
				}
				if shutdownUAT != true {
					shutdownUAT = true
				}
				break
			}

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
	log.Printf("===== ES Device Serial: %s PPM %d =====\n", e.serial, e.ppm)
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
	log.Printf("===== UAT Device Name  : %s =====\n", rtl.GetDeviceName(u.indexID))
	log.Printf("===== UAT Device Serial: %s=====\n", u.serial)

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

func reCompile(s string) *regexp.Regexp {
	// note , compile returns a nil pointer on error
	r, _ := regexp.Compile(s)
	return r
}

type regexUAT regexp.Regexp
type regexES regexp.Regexp

var rUAT = (*regexUAT)(reCompile("str?a?t?u?x:978"))
var rES = (*regexES)(reCompile("str?a?t?u?x:1090"))

func (r *regexUAT) hasID(serial string) bool {
	if r == nil {
		return strings.HasPrefix(serial, "stratux:978")
	}
	return (*regexp.Regexp)(r).MatchString(serial)
}

func (r *regexES) hasID(serial string) bool {
	if r == nil {
		return strings.HasPrefix(serial, "stratux:1090")
	}
	return (*regexp.Regexp)(r).MatchString(serial)
}

func createUATDev(id int, serial string, idSet bool) error {
	UATDev = &UAT{indexID: id, serial: serial}
	if err := UATDev.sdrConfig(); err != nil {
		log.Printf("UATDev.sdrConfig() failed: %s\n", err)
		UATDev = nil
		return err
	}
	UATDev.wg = &sync.WaitGroup{}
	UATDev.idSet = idSet
	UATDev.closeCh = make(chan int)
	UATDev.wg.Add(1)
	go UATDev.read()
	return nil
}

func createESDev(id int, serial string, idSet bool) error {
	ESDev = &ES{indexID: id, serial: serial}
	if err := ESDev.sdrConfig(); err != nil {
		log.Printf("ESDev.sdrConfig() failed: %s\n", err)
		ESDev = nil
		return err
	}
	ESDev.wg = &sync.WaitGroup{}
	ESDev.idSet = idSet
	ESDev.closeCh = make(chan int)
	ESDev.wg.Add(1)
	go ESDev.read()
	return nil
}

func configDevices(count int, es_enabled, uat_enabled bool) {
	// entry to this function is only valid when both UATDev and ESDev are nil

	// once the tagged dongles have been assigned, explicitly range over
	// the remaining IDs and assign them to any anonymous dongles
	unusedIDs := make(map[int]string)

	// loop 1: assign tagged dongles
	for i := 0; i < count; i++ {
		_, _, s, err := rtl.GetDeviceUsbStrings(i)
		if err == nil {
			// no need to check if createXDev returned an error; if it
			// failed to config the error is logged and we can ignore
			// it here so it doesn't get queued up again
			if uat_enabled && UATDev == nil && rUAT.hasID(s) {
				createUATDev(i, s, true)
			} else if es_enabled && ESDev == nil && rES.hasID(s) {
				createESDev(i, s, true)
			} else {
				unusedIDs[i] = s
			}
		} else {
			log.Printf("rtl.GetDeviceUsbStrings id %d: %s\n", i, err)
		}
	}

	// loop 2; assign anonymous dongles, but sanity check the serial ids
	// so we don't cross config for dual assigned dongles. E.g. when two
	// dongles are set to the same stratux id and the unconsumed, non-anonymous,
	// dongle makes it to this loop.
	for i, s := range unusedIDs {
		if uat_enabled && UATDev == nil && !rES.hasID(s) {
			createUATDev(i, s, false)
		} else if es_enabled && ESDev == nil && !rUAT.hasID(s) {
			createESDev(i, s, false)
		}
	}
}

// to gracefully shut down a read method we check a channel for
// a close flag, but now we want to handle catastrophic dongle
// failures (when ReadSync returns an error or we get stderr
// ouput) but we can't just bypass the channel check and return
// directly from a goroutine because the close channel call in
// the shutdown method will cause a runtime panic, hence these...
var shutdownES bool
var shutdownUAT bool

// Watch for config/device changes.
func sdrWatcher() {
	prevCount := 0
	prevUAT_Enabled := false
	prevES_Enabled := false

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

		// true when a ReadSync call fails
		if shutdownUAT {
			UATDev.shutdown()
			UATDev = nil
			shutdownUAT = false
		}
		// true when we get stderr output
		if shutdownES {
			ESDev.shutdown()
			ESDev = nil
			shutdownES = false
		}

		count := rtl.GetDeviceCount()
		atomic.StoreUint32(&globalStatus.Devices, uint32(count))

		// support two and only two dongles
		if count > 2 {
			count = 2
		}

		// check for either no dongles or none enabled
		if count < 1 || (!globalSettings.UAT_Enabled && !globalSettings.ES_Enabled) {
			if UATDev != nil {
				UATDev.shutdown()
				UATDev = nil
			}
			if ESDev != nil {
				ESDev.shutdown()
				ESDev = nil
			}
			prevCount = count
			prevUAT_Enabled = false
			prevES_Enabled = false
			continue
		}

		// if the device count or the global settings change, do a reconfig.
		// both events are significant and the least convoluted way to handle it
		// is to reconfigure all dongle/s across the board. The reconfig
		// should happen fairly quick so the user shouldn't notice any
		// major disruption; if it is significant we can split the dongle
		// count check from the global settings check where the gloabl settings
		// check won't do a reconfig.
		if count != prevCount || prevES_Enabled != globalSettings.ES_Enabled ||
			prevUAT_Enabled != globalSettings.UAT_Enabled {
			if UATDev != nil {
				UATDev.shutdown()
				UATDev = nil

			}
			if ESDev != nil {
				ESDev.shutdown()
				ESDev = nil
			}
			configDevices(count, globalSettings.ES_Enabled, globalSettings.UAT_Enabled)
		}

		prevCount = count
		prevUAT_Enabled = globalSettings.UAT_Enabled
		prevES_Enabled = globalSettings.ES_Enabled

	}
}

func sdrInit() {
	go sdrWatcher()
	go uatReader()
	godump978.Dump978Init()
	go godump978.ProcessDataFromChannel()
}
