package main

import (
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"../godump978"
	rtl "github.com/jpoirier/gortlsdr"
)

type UAT struct {
	dev     *rtl.Context
	indexID int
}

type ES struct {
	dev     *rtl.Context
	indexID int
}

var UATDev *UAT
var ESDev *ES

var uat_shutdown chan int
var uat_wg *sync.WaitGroup = &sync.WaitGroup{}

var es_shutdown chan int
var es_wg *sync.WaitGroup = &sync.WaitGroup{}

var maxSignalStrength int

func (e *ES) read() {
	defer es_wg.Done()
	log.Println("Entered ES read() ...")
	cmd := exec.Command("/usr/bin/dump1090", "--net", "--device-index", strconv.Itoa(e.indexID))
	err := cmd.Start()
	if err != nil {
		log.Printf("Error executing /usr/bin/dump1090: %s\n", err.Error())
		return
	}
	log.Println("Executed /usr/bin/dump1090 successfully...")
	for {
		select {
		default:
			time.Sleep(1 * time.Second)
		case <-es_shutdown:
			log.Println("ES read(): shutdown msg received, calling cmd.Process.Kill() ...")
			err := cmd.Process.Kill()
			if err != nil {
				log.Println("\t couldn't kill dump1090: %s", err.Error)
			} else {
				cmd.Wait()
				log.Println("\t kill successful...")
			}
			return
		}
	}
}

func (u *UAT) read() {
	defer uat_wg.Done()
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
		case <-uat_shutdown:
			log.Println("UAT read(): shutdown msg received...")
			return
		}
	}
}

func (e *ES) sdrConfig() (err error) {
	return
}

// Read 978MHz from SDR.
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

	tgain := 480
	err = u.dev.SetTunerGain(tgain)
	if err != nil {
		u.dev.Close()
		log.Printf("\tSetTunerGain Failed - error: %s\n", err)
		return
	} else {
		log.Printf("\tSetTunerGain Successful\n")
	}

	//---------- Get/Set Sample Rate ----------
	samplerate := 2083334
	err = u.dev.SetSampleRate(samplerate)
	if err != nil {
		u.dev.Close()
		log.Printf("\tSetSampleRate Failed - error: %s\n", err)
		return
	} else {
		log.Printf("\tSetSampleRate - rate: %d\n", samplerate)
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

	newRTLFreq := 28800000
	newTunerFreq := 28800000
	err = u.dev.SetXtalFreq(newRTLFreq, newTunerFreq)
	if err != nil {
		u.dev.Close()
		log.Printf("\tSetXtalFreq Failed - error: %s\n", err)
		return
	} else {
		log.Printf("\tSetXtalFreq - Center freq: %d, Tuner freq: %d\n",
			newRTLFreq, newTunerFreq)
	}

	//---------- Get/Set Center Freq ----------
	err = u.dev.SetCenterFreq(978000000)
	if err != nil {
		u.dev.Close()
		log.Printf("\tSetCenterFreq 978MHz Failed, error: %s\n", err)
		return
	} else {
		log.Printf("\tSetCenterFreq 978MHz Successful\n")
	}

	log.Printf("\tGetCenterFreq: %d\n", u.dev.GetCenterFreq())

	//---------- Set Bandwidth ----------
	bw := 1000000
	log.Printf("\tSetting Bandwidth: %d\n", bw)
	if err = u.dev.SetTunerBw(bw); err != nil {
		u.dev.Close()
		log.Printf("\tSetTunerBw %d Failed, error: %s\n", bw, err)
		return
	} else {
		log.Printf("\tSetTunerBw %d Successful\n", bw)
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
	err = u.dev.SetFreqCorrection(globalSettings.PPM)
	if err != nil {
		u.dev.Close()
		log.Printf("\tSetFreqCorrection %d Failed, error: %s\n", globalSettings.PPM, err)
		return
	} else {
		log.Printf("\tSetFreqCorrection %d Successful\n", globalSettings.PPM)
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

func (u *UAT) shutdown() {
	log.Println("Entered UAT shutdown() ...")
	close(uat_shutdown) // signal to shutdown
	log.Println("UAT shutdown(): closing device ...")
	u.dev.Close() // preempt the blocking ReadSync call
	log.Println("UAT shutdown(): calling uat_wg.Wait() ...")
	uat_wg.Wait() // Wait for the goroutine to shutdown
	log.Println("UAT shutdown(): uat_wg.Wait() returned...")
}

func (e *ES) shutdown() {
	log.Println("Entered ES shutdown() ...")
	close(es_shutdown) // signal to shutdown
	log.Println("ES shutdown(): calling es_wg.Wait() ...")
	es_wg.Wait() // Wait for the goroutine to shutdown
	log.Println("ES shutdown(): es_wg.Wait() returned...")
}

var devMap = map[int]string{0: "", 1: ""}

// Watch for config/device changes.
func sdrWatcher() {
	for {
		time.Sleep(1 * time.Second)
		count := rtl.GetDeviceCount()
		atomic.StoreUint32(&globalStatus.Devices, uint32(count))
		log.Println("DeviceCount...", count)

		// support two and only two dongles
		if count > 2 {
			count = 2
		}

		// cleanup if necessary
		if count < 1 || (!globalSettings.UAT_Enabled && !globalSettings.ES_Enabled) {
			log.Println("count == 0, doing cleanup if necessary...")
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
			if UATDev != nil && ESDev == nil {
				UATDev.indexID = 0
			} else if UATDev == nil && ESDev != nil {
				ESDev.indexID = 0
			}
		}

		ids := []string{"", ""}
		for i := 0; i < count; i++ {
			// manufact, product, serial, err
			_, _, s, _ := rtl.GetDeviceUsbStrings(i)
			ids[i] = s
		}

		// UAT specific handling
		// When count is one, favor UAT in the case where the user
		// has enabled both UAT and ES via the web interface.
		id := 0
		if globalSettings.UAT_Enabled {
			// log.Println("globalSettings.UAT_Enabled == true")
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
				// log.Println("\tUATDev == nil")
				// preference check based on stratux
				// hardware serial when it exists
				serial := ids[id]
				if strings.HasPrefix(serial, "stratux:1090") {
					log.Println("Settings conflict: 978UAT set via WebUI but hardware serial says stratux:1090")
				} else {
					UATDev = &UAT{indexID: id}
					if err := UATDev.sdrConfig(); err != nil {
						log.Printf("UATDev = &UAT{indexID: id} failed: %s\n", err)
						UATDev = nil
					} else {
						uat_shutdown = make(chan int)
						uat_wg.Add(1)
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
		id = 0
		if globalSettings.ES_Enabled {
			// log.Println("globalSettings.ES_Enabled == true")
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
				// log.Println("\tESDev == nil")
				// preference check based on stratux
				// hardware serial when it exists
				serial := ids[id]
				if strings.HasPrefix(serial, "stratux:978") {
					log.Println("Settings conflict: 1090ES set via WebUI but hardware serial says stratux:978")
				} else {
					ESDev = &ES{indexID: id}
					if err := ESDev.sdrConfig(); err != nil {
						log.Printf("ESDev = &ES{indexID: id} failed: %s\n", err)
						ESDev = nil
					} else {
						es_shutdown = make(chan int)
						es_wg.Add(1)
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
