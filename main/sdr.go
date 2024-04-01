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
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/b3nn0/stratux/godump978"
	rtl "github.com/jpoirier/gortlsdr"
)

// Device holds per dongle values and attributes
type Device struct {
	dev     *rtl.Context
	wg      *sync.WaitGroup
	closeCh chan int
	indexID int
	ppm     int
	gain     float64
	serial  string
	idSet   bool
}

// UAT is a 978 MHz device
type UAT Device

// ES is a 1090 MHz device
type ES Device

// OGN is an 868 MHz device
type OGN Device

// AIS is an 161 MHz device
type AIS Device

// UATDev holds a 978 MHz dongle object
var UATDev *UAT

// ESDev holds a 1090 MHz dongle object
var ESDev *ES

// OGNDev holds an 868 MHz dongle object
var OGNDev *OGN

// AISDev holds a 162 MHz dongle object
var AISDev *AIS

type Dump1090TermMessage struct {
	Text   string
	Source string
}

type AISTermMessage struct {
	Text   string
	Source string
}

func (e *ES) read() {
	defer e.wg.Done()
	log.Println("Entered ES read() ...")
	// RTL SDR Standard Gains: 0.0 0.9 1.4 2.7 3.7 7.7 8.7 12.5 14.4 15.7 16.6 19.7 20.7 22.9 25.4 28.0 29.7 32.8 33.8 36.4 37.2 38.6 40.2 42.1 43.4 43.9 44.5 48.0 49.6
	if(e.gain<0.9){
		e.gain = 37.2
	}
	cmd := exec.Command(STRATUX_HOME + "/bin/dump1090", "--fix", "--net-stratux-port", "30006",  "--net", "--device-index", strconv.Itoa(e.indexID),
		"--ppm", strconv.Itoa(e.ppm),
		"--gain",strconv.FormatFloat(e.gain,'f',-1,32),
		"--mlat") // display raw messages in Beast ascii mode
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	err := cmd.Start()
	if err != nil {
		log.Printf("Error executing " + STRATUX_HOME + "/bin/dump1090: %s\n", err)
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

	log.Println("Executed " + cmd.String() + " successfully...")

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

func (f *OGN) read() {
	defer f.wg.Done()
	log.Println("Entered OGN read() ...")

	// ogn-rx doesn't like the time jumping forward while running.. delay initial startup until we have a valid system time
	if !isGPSClockValid() {
		log.Printf("Delaying ogn-rx start until we have a valid GPS time")
		loop: for {
			select {
			case <- f.closeCh:
				return
			default:
				if isGPSClockValid() {
					break loop
				} else {
					time.Sleep(1 * time.Second)
				}
			}
		}
	}

	args := []string {"-d", strconv.Itoa(f.indexID), "-p", strconv.Itoa(f.ppm), "-L/var/log/"}
	if !globalSettings.OGNI2CTXEnabled {
		args = append(args, "-t", "off")
	}
	cmd := exec.Command(STRATUX_HOME + "/bin/ogn-rx-eu", args...)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	autoRestart := true // automatically restart crashing child process

	err := cmd.Start()
	if err != nil {
		log.Printf("OGN: Error executing ogn-rx-eu: %s\n", err)
		// don't return immediately, use the proper shutdown procedure
		shutdownOGN = true
		for {
			select {
			case <-f.closeCh:
				return
			default:
				time.Sleep(1 * time.Second)
			}
		}
	}

	log.Println("OGN: Executed ogn-rx-eu successfully...")

	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case <-f.closeCh:
				log.Println("OGN read(): shutdown msg received, calling cmd.Process.Kill() ...")
				autoRestart = false
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
		reader := bufio.NewReader(stdout)
		for {
			select {
			case <-done:
				return
			default:
				line, err := reader.ReadString('\n')
				line = strings.TrimSpace(line)
				if err == nil  && len(line) > 0 /* && globalSettings.DEBUG */ {
					log.Println("OGN: ogn-rx-eu stdout: ", line)
				}
			}
		}
	}()

	go func() {
		reader := bufio.NewReader(stderr)
		for {
			select {
			case <-done:
				return
			default:
				line, err := reader.ReadString('\n')
				if err == nil {
					log.Println("OGN: ogn-rx-eu stderr: ", strings.TrimSpace(line))
				}
			}
		}
	}()

	cmd.Wait()

	log.Println("OGN: ogn-rx-eu terminated...")

	// we get here if A) the ogn-rx-eu process died
	// on its own or B) cmd.Process.Kill() was called
	// from within the goroutine, either way close
	// the "done" channel, which ensures we don't leak
	// goroutines...
	close(done)

	if autoRestart && !shutdownOGN{
		time.Sleep(5 * time.Second)
		log.Println("OGN: restarting crashed ogn-rx-eu")
		f.wg.Add(1)
		go f.read()
	}
}

func (e *AIS) read() {
	defer e.wg.Done()
	log.Println("Entered AIS read() ...")
	cmd := exec.Command(STRATUX_HOME + "/bin/rtl_ais", "-T", "-k", "-p", strconv.Itoa(e.ppm), "-d", strconv.Itoa(e.indexID))
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	err := cmd.Start()
	if err != nil {
		log.Printf("Error executing " + STRATUX_HOME + "/bin/rtl_ais: %s\n", err)
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

	log.Println("Executed " + cmd.String() + " successfully...")

	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case <-e.closeCh:
				log.Println("AIS read(): shutdown msg received, calling cmd.Process.Kill() ...")
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
					m := AISTermMessage{Text: string(stdoutBuf[:n]), Source: "stdout"}
					logAISTermMessage(m)
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
					m := AISTermMessage{Text: string(stderrBuf[:n]), Source: "stderr"}
					logAISTermMessage(m)
				}
			}
		}
	}()

	cmd.Wait()

	// we get here if A) the rvt_ais process died
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
	e.gain = globalSettings.Dump1090Gain
	log.Printf("===== ES Device Serial: %s PPM %d Gain %.1f =====\n", e.serial, e.ppm,e.gain)
	return
}

func (f *OGN) sdrConfig() (err error) {
	f.ppm = getPPM(f.serial)
	log.Printf("===== OGN Device Serial: %s PPM %d =====\n", f.serial, f.ppm)
	return
}

func (f *AIS) sdrConfig() (err error) {
	f.ppm = getPPM(f.serial)
	log.Printf("===== AIS Device Serial: %s PPM %d =====\n", f.serial, f.ppm)
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
		TraceLog.Record(CONTEXT_GODUMP978, []byte(uat))
		handleUatMessage(uat)
	}
}

func handleUatMessage(uat string) {
	o, msgtype := parseInput(uat)
	if o != nil && msgtype != 0 {
		relayMessage(msgtype, o)
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

func (f *OGN) writeID() error {
	info, err := f.dev.GetHwInfo()
	if err != nil {
		return err
	}
	info.Serial = "stratux:868"
	return f.dev.SetHwInfo(info)
}

func (f *AIS) writeID() error {
	info, err := f.dev.GetHwInfo()
	if err != nil {
		return err
	}
	info.Serial = "stratux:162"
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

func (f *OGN) shutdown() {
	log.Println("Entered OGN shutdown() ...")
	close(f.closeCh) // signal to shutdown
	log.Println("signal shutdown(): calling f.wg.Wait() ...")
	f.wg.Wait() // Wait for the goroutine to shutdown
	log.Println("signal shutdown() complete ...")
}

func (f *AIS) shutdown() {
	log.Println("Entered AIS shutdown() ...")
	close(f.closeCh) // signal to shutdown
	log.Println("signal shutdown(): calling f.wg.Wait() ...")
	f.wg.Wait() // Wait for the goroutine to shutdown
	log.Println("signal shutdown() complete ...")
}

var sdrShutdown bool

func sdrKill() {
	// Send signal to shutdown to sdrWatcher().
	sdrShutdown = true
	// Spin until all devices have been de-initialized.
	for UATDev != nil || ESDev != nil || OGNDev != nil || AISDev != nil {
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
type regexOGN regexp.Regexp
type regexAIS regexp.Regexp

var rUAT = (*regexUAT)(reCompile("str?a?t?u?x:978"))
var rES = (*regexES)(reCompile("str?a?t?u?x:1090"))
var rOGN = (*regexES)(reCompile("str?a?t?u?x:868"))
var rAIS = (*regexAIS)(reCompile("str?a?t?u?x:162"))

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

func (r *regexOGN) hasID(serial string) bool {
	if r == nil {
		return strings.HasPrefix(serial, "stratux:868")
	}
	return (*regexp.Regexp)(r).MatchString(serial)
}

func (r *regexAIS) hasID(serial string) bool {
	if r == nil {
		return strings.HasPrefix(serial, "stratux:162")
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

func createOGNDev(id int, serial string, idSet bool) error {
	OGNDev = &OGN{indexID: id, serial: serial}
	if err := OGNDev.sdrConfig(); err != nil {
		log.Printf("OGNDev.sdrConfig() failed: %s\n", err)
		OGNDev = nil
		return err
	}
	OGNDev.wg = &sync.WaitGroup{}
	OGNDev.idSet = idSet
	OGNDev.closeCh = make(chan int)
	OGNDev.wg.Add(1)
	go OGNDev.read()
	return nil
}

func createAISDev(id int, serial string, idSet bool) error {
	AISDev = &AIS{indexID: id, serial: serial}
	if err := AISDev.sdrConfig(); err != nil {
		log.Printf("AISDev.sdrConfig() failed: %s\n", err)
		AISDev = nil
		return err
	}
	AISDev.wg = &sync.WaitGroup{}
	AISDev.idSet = idSet
	AISDev.closeCh = make(chan int)
	AISDev.wg.Add(1)
	go AISDev.read()
	return nil
}

func configDevices(count int, esEnabled, uatEnabled, ognEnabled, aisEnabled bool) {
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
			} else if ognEnabled && OGNDev == nil && rOGN.hasID(s) {
				createOGNDev(i, s, true)
			} else if aisEnabled && AISDev == nil && rAIS.hasID(s) {
				createAISDev(i, s, true)
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
		if uatEnabled && !globalStatus.UATRadio_connected && UATDev == nil && !rES.hasID(s) && !rOGN.hasID(s) {
			createUATDev(i, s, false)
		} else if esEnabled && ESDev == nil && !rUAT.hasID(s) && !rOGN.hasID(s) {
			createESDev(i, s, false)
		} else if ognEnabled && OGNDev == nil {
			createOGNDev(i, s, false)
		} else if aisEnabled && AISDev == nil {
			createAISDev(i, s, false)
		}
	}
}

// to keep our sync primitives synchronized, only exit a read
// method's goroutine via the close flag channel check, to
// include catastrophic dongle failures
var shutdownES bool
var shutdownUAT bool
var shutdownOGN bool
var shutdownAIS bool

// Watch for config/device changes.
func sdrWatcher() {
	prevCount := 0
	prevUATEnabled := false
	prevESEnabled := false
	prevOGNEnabled := false
	prevAISEnabled := false
	prevOGNTXEnabled := false

	// Get the system (RPi) uptime.
	info := syscall.Sysinfo_t{}
	err := syscall.Sysinfo(&info)

	if err == nil && info.Uptime > 120 && globalSettings.DeveloperMode {
		// Throw a "critical error" if developer mode is enabled. Alerts the developer that the daemon was restarted (possibly)
		//  unexpectedly.
		addSingleSystemErrorf("restart-warn", "System uptime %d seconds. Daemon was restarted.\n", info.Uptime)
	}

	// Got system uptime. Delay SDR start for a bit to reduce noise for the GPS to get a fix.
	// Will give up waiting after 120s without fix
	for err == nil && info.Uptime < 120 && !isGPSValid()  {
		time.Sleep(1 * time.Second)
		err = syscall.Sysinfo(&info)
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
			if OGNDev != nil {
				OGNDev.shutdown()
				OGNDev = nil
			}
			if AISDev != nil {
				AISDev.shutdown()
				AISDev = nil
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
		if shutdownOGN {
			if OGNDev != nil {
				OGNDev.shutdown()
				OGNDev = nil
			}
			shutdownOGN = false
		}
		if shutdownAIS {
			if AISDev != nil {
				AISDev.shutdown()
				AISDev = nil
			}
			shutdownAIS = false
		}

		// capture current state
		esEnabled := globalSettings.ES_Enabled
		uatEnabled := globalSettings.UAT_Enabled
		ognEnabled := globalSettings.OGN_Enabled
		aisEnabled := globalSettings.AIS_Enabled
		ognTXEnabled := globalSettings.OGNI2CTXEnabled
		count := rtl.GetDeviceCount()
		interfaceCount := count
		if globalStatus.UATRadio_connected {
			interfaceCount++
		}
		atomic.StoreUint32(&globalStatus.Devices, uint32(interfaceCount))

		// support up to 3 dongles
		if count > 3 {
			count = 3
		}

		if interfaceCount == prevCount && prevESEnabled == esEnabled && prevUATEnabled == uatEnabled && prevOGNEnabled == ognEnabled && prevAISEnabled == aisEnabled && prevOGNTXEnabled == ognTXEnabled {
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
		if OGNDev != nil {
			OGNDev.shutdown()
			OGNDev = nil
		}
		if AISDev != nil {
			AISDev.shutdown()
			AISDev = nil
		}
		configDevices(count, esEnabled, uatEnabled, ognEnabled, aisEnabled)

		prevCount = interfaceCount
		prevUATEnabled = uatEnabled
		prevESEnabled = esEnabled
		prevOGNEnabled = ognEnabled
		prevAISEnabled = aisEnabled
		prevOGNTXEnabled = ognTXEnabled

		countEnabled := 0

		if uatEnabled { countEnabled++ }
		if esEnabled { countEnabled++ }
		if ognEnabled { countEnabled++ }
		if aisEnabled { countEnabled++ }
		if countEnabled > interfaceCount {
			// User enabled too many protocols. Show error..
			used := make([]string, 0)
			if UATDev != nil { used = append(used, "UAT") }
			if ESDev != nil { used = append(used, "1090ES") }
			if OGNDev != nil { used = append(used, "OGN") }
			if AISDev != nil { used = append(used, "AIS") }
			addSingleSystemErrorf("sdrconfig", "You have enabled more protocols than you have receivers for. " +
				"You have %d receivers, but enabled %d protocols. Please disable %d of them for things to work correctly. For now we are only using %s.",
				count, countEnabled, countEnabled - count, strings.Join(used, ", "))

		} else {
			removeSingleSystemError("sdrconfig")
		}

	}
}

func sdrInit() {
	go sdrWatcher()
	go uatReader()
	go godump978.ProcessDataFromChannel()
}
