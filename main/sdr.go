/*
	Copyright (c) 2015-2016 Christopher Young
	Distributable under the terms of The "BSD New" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	sdr.go: SDR monitoring, SDR management, data input from UAT/1090ES channels.
*/

package main

import (
	"bufio"
	"io"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"../godump978"
	rtl "github.com/jpoirier/gortlsdr"
)

// Device holds per dongle values and attributes
type Device struct {
	dev     *rtl.Context
	wg      *sync.WaitGroup
	closeCh chan int
	indexID int
	ppm     int
	serial  string
	idSet   bool
}

// UAT is a 978 MHz device
type UAT Device

// ES is a 1090 MHz device
type ES Device

// FLARM is an 868 MHz device
type FLARM Device

// UATDev holds a 978 MHz dongle object
var UATDev *UAT

// ESDev holds a 1090 MHz dongle object
var ESDev *ES

// FLARMDev holds an 868 MHz dongle object
var FLARMDev *FLARM

type Dump1090TermMessage struct {
	Text   string
	Source string
}

func (e *ES) read() {
	defer e.wg.Done()
	log.Println("Entered ES read() ...")
	cmd := exec.Command("/usr/bin/dump1090", "--oversample", "--net", "--device-index", strconv.Itoa(e.indexID), "--ppm", strconv.Itoa(e.ppm))
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	err := cmd.Start()
	if err != nil {
		log.Printf("Error executing /usr/bin/dump1090: %s\n", err)
		// don't return immediately, use the proper shutdown procedure
		shutdownES = true
		for {
			select {
			case <-e.closeCh:
				return
			default:
				time.Sleep(1 * time.Second)
			}
		}
	}

	log.Println("Executed /usr/bin/dump1090 successfully...")

	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case <-e.closeCh:
				log.Println("ES read(): shutdown msg received, calling cmd.Process.Kill() ...")
				err := cmd.Process.Kill()
				if err == nil {
					log.Println("kill successful...")
				}
				return
			default:
				time.Sleep(1 * time.Second)
			}
		}
	}()

	stdoutBuf := make([]byte, 1024)
	stderrBuf := make([]byte, 1024)
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				n, err := stdout.Read(stdoutBuf)
				if err == nil && n > 0 {
					m := Dump1090TermMessage{Text: string(stdoutBuf[:n]), Source: "stdout"}
					logDump1090TermMessage(m)
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case <-done:
				return
			default:
				n, err := stderr.Read(stderrBuf)
				if err == nil && n > 0 {
					m := Dump1090TermMessage{Text: string(stderrBuf[:n]), Source: "stderr"}
					logDump1090TermMessage(m)
				}
			}
		}
	}()

	cmd.Wait()

	// we get here if A) the dump1090 process died
	// on its own or B) cmd.Process.Kill() was called
	// from within the goroutine, either way close
	// the "done" channel, which ensures we don't leak
	// goroutines...
	close(done)
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

func (f *FLARM) read() {
	defer f.wg.Done()
	log.Println("Entered FLARM read() ...")

	// generate OGN configuration file
	configTemplateFileName := "/etc/stratux-ogn.conf.template"
	configFileName := "/tmp/stratux-ogn.conf"
	OGNConfigDataCache.DeviceIndex = strconv.Itoa(f.indexID)
	OGNConfigDataCache.Ppm = strconv.Itoa(f.ppm)
	OGNConfigDataCache.Longitude = "0.0"
	OGNConfigDataCache.Latitude = "0.0"
	OGNConfigDataCache.Altitude = "0"
	createOGNConfigFile(configTemplateFileName, configFileName)

	cmd := exec.Command("ogn-rf", configFileName)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	stdin, _ := cmd.StdinPipe()
	defer stdin.Close()

	err := cmd.Start()
	if err != nil {
		log.Printf("FLARM: Error executing ogn-rf: %s\n", err)
		// don't return immediately, use the proper shutdown procedure
		shutdownFLARM = true
		for {
			select {
			case <-f.closeCh:
				return
			default:
				time.Sleep(1 * time.Second)
			}
		}
	}

	log.Println("FLARM: Executed ogn-rf successfully...")

	io.WriteString(stdin, "\n")

	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case <-f.closeCh:
				log.Println("FLARM read(): shutdown msg received, calling cmd.Process.Kill() ...")
				err := cmd.Process.Kill()
				if err == nil {
					log.Println("kill successful...")
				}
				return
			default:
				time.Sleep(1 * time.Second)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-done:
				return
			default:
				line, err := bufio.NewReader(stdout).ReadString('\n')
				if err == nil {
					log.Println("FLARM: ogn-rf stdout: ", strings.TrimSpace(line))
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case <-done:
				return
			default:
				line, err := bufio.NewReader(stderr).ReadString('\n')
				if err == nil {
					log.Println("FLARM: ogn-rf stderr: ", strings.TrimSpace(line))
				}
			}
		}
	}()

	cmd.Wait()

	log.Println("FLARM: ogn-rf terminated...")

	// we get here if A) the dump1090 process died
	// on its own or B) cmd.Process.Kill() was called
	// from within the goroutine, either way close
	// the "done" channel, which ensures we don't leak
	// goroutines...
	close(done)
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

	ppm, err := strconv.Atoi(arr[1])
	if err != nil {
		return globalSettings.PPM
	}

	return ppm
}

func (e *ES) sdrConfig() (err error) {
	e.ppm = getPPM(e.serial)
	log.Printf("===== ES Device Serial: %s PPM %d =====\n", e.serial, e.ppm)
	return
}

func (f *FLARM) sdrConfig() (err error) {
	f.ppm = getPPM(f.serial)
	log.Printf("===== FLARM Device Serial: %s PPM %d =====\n", f.serial, f.ppm)
	return
}

// 978 UAT configuration settings
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
	}
	log.Printf("\tSetTunerGainMode Successful\n")

	err = u.dev.SetTunerGain(TunerGain)
	if err != nil {
		u.dev.Close()
		log.Printf("\tSetTunerGain Failed - error: %s\n", err)
		return
	}
	log.Printf("\tSetTunerGain Successful\n")

	tgain := u.dev.GetTunerGain()
	log.Printf("\tGetTunerGain: %d\n", tgain)

	//---------- Get/Set Sample Rate ----------
	err = u.dev.SetSampleRate(SampleRate)
	if err != nil {
		u.dev.Close()
		log.Printf("\tSetSampleRate Failed - error: %s\n", err)
		return
	}
	log.Printf("\tSetSampleRate - rate: %d\n", SampleRate)

	log.Printf("\tGetSampleRate: %d\n", u.dev.GetSampleRate())

	//---------- Get/Set Xtal Freq ----------
	rtlFreq, tunerFreq, err := u.dev.GetXtalFreq()
	if err != nil {
		u.dev.Close()
		log.Printf("\tGetXtalFreq Failed - error: %s\n", err)
		return
	}
	log.Printf("\tGetXtalFreq - Rtl: %d, Tuner: %d\n", rtlFreq, tunerFreq)

	err = u.dev.SetXtalFreq(NewRTLFreq, NewTunerFreq)
	if err != nil {
		u.dev.Close()
		log.Printf("\tSetXtalFreq Failed - error: %s\n", err)
		return
	}
	log.Printf("\tSetXtalFreq - Center freq: %d, Tuner freq: %d\n",
		NewRTLFreq, NewTunerFreq)

	//---------- Get/Set Center Freq ----------
	err = u.dev.SetCenterFreq(CenterFreq)
	if err != nil {
		u.dev.Close()
		log.Printf("\tSetCenterFreq 978MHz Failed, error: %s\n", err)
		return
	}
	log.Printf("\tSetCenterFreq 978MHz Successful\n")

	log.Printf("\tGetCenterFreq: %d\n", u.dev.GetCenterFreq())

	//---------- Set Bandwidth ----------
	log.Printf("\tSetting Bandwidth: %d\n", Bandwidth)
	if err = u.dev.SetTunerBw(Bandwidth); err != nil {
		u.dev.Close()
		log.Printf("\tSetTunerBw %d Failed, error: %s\n", Bandwidth, err)
		return
	}
	log.Printf("\tSetTunerBw %d Successful\n", Bandwidth)

	if err = u.dev.ResetBuffer(); err != nil {
		u.dev.Close()
		log.Printf("\tResetBuffer Failed - error: %s\n", err)
		return
	}
	log.Printf("\tResetBuffer Successful\n")

	//---------- Get/Set Freq Correction ----------
	freqCorr := u.dev.GetFreqCorrection()
	log.Printf("\tGetFreqCorrection: %d\n", freqCorr)

	u.ppm = getPPM(u.serial)
	err = u.dev.SetFreqCorrection(u.ppm)
	if err != nil {
		u.dev.Close()
		log.Printf("\tSetFreqCorrection %d Failed, error: %s\n", u.ppm, err)
		return
	}
	log.Printf("\tSetFreqCorrection %d Successful\n", u.ppm)

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

func (f *FLARM) writeID() error {
	info, err := f.dev.GetHwInfo()
	if err != nil {
		return err
	}
	info.Serial = "stratux:868"
	return f.dev.SetHwInfo(info)
}

func (u *UAT) shutdown() {
	log.Println("Entered UAT shutdown() ...")
	close(u.closeCh) // signal to shutdown
	log.Println("UAT shutdown(): calling u.wg.Wait() ...")
	u.wg.Wait() // Wait for the goroutine to shutdown
	log.Println("UAT shutdown(): u.wg.Wait() returned...")
	log.Println("UAT shutdown(): closing device ...")
	u.dev.Close() // preempt the blocking ReadSync call
	log.Println("UAT shutdown() complete ...")
}

func (e *ES) shutdown() {
	log.Println("Entered ES shutdown() ...")
	close(e.closeCh) // signal to shutdown
	log.Println("ES shutdown(): calling e.wg.Wait() ...")
	e.wg.Wait() // Wait for the goroutine to shutdown
	log.Println("ES shutdown() complete ...")
}

func (f *FLARM) shutdown() {
	log.Println("Entered FLARM shutdown() ...")
	close(f.closeCh) // signal to shutdown
	log.Println("FLARM shutdown(): calling f.wg.Wait() ...")
	f.wg.Wait() // Wait for the goroutine to shutdown
	log.Println("FLARM shutdown() complete ...")
}

var sdrShutdown bool

func sdrKill() {
	// Send signal to shutdown to sdrWatcher().
	sdrShutdown = true
	// Spin until all devices have been de-initialized.
	for UATDev != nil || ESDev != nil || FLARMDev != nil {
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
type regexFLARM regexp.Regexp

var rUAT = (*regexUAT)(reCompile("str?a?t?u?x:978"))
var rES = (*regexES)(reCompile("str?a?t?u?x:1090"))
var rFLARM = (*regexES)(reCompile("str?a?t?u?x:868"))

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

func (r *regexFLARM) hasID(serial string) bool {
	if r == nil {
		return strings.HasPrefix(serial, "stratux:868")
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

func createFLARMDev(id int, serial string, idSet bool) error {
	FLARMDev = &FLARM{indexID: id, serial: serial}
	if err := FLARMDev.sdrConfig(); err != nil {
		log.Printf("FLARMDev.sdrConfig() failed: %s\n", err)
		FLARMDev = nil
		return err
	}
	FLARMDev.wg = &sync.WaitGroup{}
	FLARMDev.idSet = idSet
	FLARMDev.closeCh = make(chan int)
	FLARMDev.wg.Add(1)
	go FLARMDev.read()
	return nil
}

func configDevices(count int, esEnabled, uatEnabled, flarmEnabled bool) {
	// once the tagged dongles have been assigned, explicitly range over
	// the remaining IDs and assign them to any anonymous dongles
	unusedIDs := make(map[int]string)

	// loop 1: assign tagged dongles
	for i := 0; i < count; i++ {
		_, _, s, err := rtl.GetDeviceUsbStrings(i)
		if err == nil {
			//FIXME: Trim NULL from the serial. Best done in gortlsdr, but putting this here for now.
			s = strings.Trim(s, "\x00")
			// no need to check if createXDev returned an error; if it
			// failed to config the error is logged and we can ignore
			// it here so it doesn't get queued up again
			if uatEnabled && UATDev == nil && rUAT.hasID(s) {
				createUATDev(i, s, true)
			} else if esEnabled && ESDev == nil && rES.hasID(s) {
				createESDev(i, s, true)
			} else if flarmEnabled && FLARMDev == nil && rFLARM.hasID(s) {
				createFLARMDev(i, s, true)
			} else {
				unusedIDs[i] = s
			}
		} else {
			log.Printf("rtl.GetDeviceUsbStrings id %d: %s\n", i, err)
		}
	}

	// loop 2: assign anonymous dongles but sanity check the serial ids
	// so we don't cross config for dual assigned dongles. e.g. when two
	// dongles are set to the same stratux id and the unconsumed,
	// non-anonymous, dongle makes it to this loop.
	for i, s := range unusedIDs {
		if uatEnabled && !globalStatus.UATRadio_connected && UATDev == nil && !rES.hasID(s) && !rFLARM.hasID(s) {
			createUATDev(i, s, false)
		} else if esEnabled && ESDev == nil && !rUAT.hasID(s) && !rFLARM.hasID(s) {
			createESDev(i, s, false)
		} else if flarmEnabled && FLARMDev == nil {
			createFLARMDev(i, s, false)
		}
	}
}

// to keep our sync primitives synchronized, only exit a read
// method's goroutine via the close flag channel check, to
// include catastrophic dongle failures
var shutdownES bool
var shutdownUAT bool
var shutdownFLARM bool

// Watch for config/device changes.
func sdrWatcher() {
	prevCount := 0
	prevUATEnabled := false
	prevESEnabled := false
	prevFLARMEnabled := false

	// Get the system (RPi) uptime.
	info := syscall.Sysinfo_t{}
	err := syscall.Sysinfo(&info)
	if err == nil {
		// Got system uptime. Delay if and only if the system uptime is less than 120 seconds. This should be plenty of time
		//  for the RPi to come up and start Stratux. Keeps the delay from happening if the daemon is auto-restarted from systemd.
		if info.Uptime < 120 {
			time.Sleep(90 * time.Second)
		} else if globalSettings.DeveloperMode {
			// Throw a "critical error" if developer mode is enabled. Alerts the developer that the daemon was restarted (possibly)
			//  unexpectedly.
			addSingleSystemErrorf("restart-warn", "System uptime %d seconds. Daemon was restarted.\n", info.Uptime)
		}
	}

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
			if FLARMDev != nil {
				FLARMDev.shutdown()
				FLARMDev = nil
			}
			return
		}

		// true when a ReadSync call fails
		if shutdownUAT {
			if UATDev != nil {
				UATDev.shutdown()
				UATDev = nil
			}
			shutdownUAT = false
		}
		// true when we get stderr output
		if shutdownES {
			if ESDev != nil {
				ESDev.shutdown()
				ESDev = nil
			}
			shutdownES = false
		}
		// true when we get stderr output
		if shutdownFLARM {
			if FLARMDev != nil {
				FLARMDev.shutdown()
				FLARMDev = nil
			}
			shutdownFLARM = false
		}

		// capture current state
		esEnabled := globalSettings.ES_Enabled
		uatEnabled := globalSettings.UAT_Enabled
		flarmEnabled := globalSettings.FLARM_Enabled
		count := rtl.GetDeviceCount()
		interfaceCount := count
		if globalStatus.UATRadio_connected {
			interfaceCount++
		}
		atomic.StoreUint32(&globalStatus.Devices, uint32(interfaceCount))

		// support up to two dongles
		if count > 3 {
			count = 3
		}

		if count == prevCount && prevESEnabled == esEnabled && prevUATEnabled == uatEnabled && prevFLARMEnabled == flarmEnabled {
			continue
		}

		// the device count or the global settings have changed, reconfig
		if UATDev != nil {
			UATDev.shutdown()
			UATDev = nil
		}
		if ESDev != nil {
			ESDev.shutdown()
			ESDev = nil
		}
		if FLARMDev != nil {
			FLARMDev.shutdown()
			FLARMDev = nil
		}
		configDevices(count, esEnabled, uatEnabled, flarmEnabled)

		prevCount = count
		prevUATEnabled = uatEnabled
		prevESEnabled = esEnabled
		prevFLARMEnabled = flarmEnabled
	}
}

func sdrInit() {
	go sdrWatcher()
	go uatReader()
	go godump978.ProcessDataFromChannel()
}
