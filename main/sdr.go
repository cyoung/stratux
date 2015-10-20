package main

import (
	"../godump978"
	rtl "github.com/jpoirier/gortlsdr"
	"log"
	"time"
	"os/exec"
	"strconv"
	"strings"
)

var uatSDR int // Index.
var esSDR int  // Index.

var maxSignalStrength int


func getSDRSerial(dev *rtl.Context) (string, error) {
	info, err := dev.GetHwInfo()
	if err != nil {
		return "", err
	}
	info.Serial = strings.Replace(info.Serial, "\x00", "", -1)
	return info.Serial, nil
}

func setSDRSerial(dev *rtl.Context, info rtl.HwInfo) error {
	return dev.SetHwInfo(info)
}

// Read 978MHz from SDR.
func sdrReader() {
	var err error
	var dev *rtl.Context

	log.Printf("===== UAT Device name: %s =====\n", rtl.GetDeviceName(uatSDR))
	if dev, err = rtl.Open(uatSDR); err != nil {
		log.Printf("\tOpen Failed, exiting\n")
		uatSDR = -1
		return
	}
	defer dev.Close()

	log.Printf("\tGetTunerType: %s\n", dev.GetTunerType())

	//---------- Set Tuner Gain ----------
	tgain := 480

	err = dev.SetTunerGainMode(true)
	if err != nil {
		log.Printf("\tSetTunerGainMode Failed - error: %s\n", err)
	} else {
		log.Printf("\tSetTunerGainMode Successful\n")
	}

	err = dev.SetTunerGain(tgain)
	if err != nil {
		log.Printf("\tSetTunerGain Failed - error: %s\n", err)
	} else {
		log.Printf("\tSetTunerGain Successful\n")
	}

	//---------- Get/Set Sample Rate ----------
	samplerate := 2083334
	err = dev.SetSampleRate(samplerate)
	if err != nil {
		log.Printf("\tSetSampleRate Failed - error: %s\n", err)
	} else {
		log.Printf("\tSetSampleRate - rate: %d\n", samplerate)
	}
	log.Printf("\tGetSampleRate: %d\n", dev.GetSampleRate())

	//---------- Get/Set Xtal Freq ----------
	rtlFreq, tunerFreq, err := dev.GetXtalFreq()
	if err != nil {
		log.Printf("\tGetXtalFreq Failed - error: %s\n", err)
	} else {
		log.Printf("\tGetXtalFreq - Rtl: %d, Tuner: %d\n", rtlFreq, tunerFreq)
	}

	newRTLFreq := 28800000
	newTunerFreq := 28800000
	err = dev.SetXtalFreq(newRTLFreq, newTunerFreq)
	if err != nil {
		log.Printf("\tSetXtalFreq Failed - error: %s\n", err)
	} else {
		log.Printf("\tSetXtalFreq - Center freq: %d, Tuner freq: %d\n",
			newRTLFreq, newTunerFreq)
	}

	//---------- Get/Set Center Freq ----------
	err = dev.SetCenterFreq(978000000)
	if err != nil {
		log.Printf("\tSetCenterFreq 978MHz Failed, error: %s\n", err)
	} else {
		log.Printf("\tSetCenterFreq 978MHz Successful\n")
	}

	log.Printf("\tGetCenterFreq: %d\n", dev.GetCenterFreq())

	//---------- Set Bandwidth ----------
	bw := 1000000
	log.Printf("\tSetting Bandwidth: %d\n", bw)
	if err = dev.SetTunerBw(bw); err != nil {
		log.Printf("\tSetTunerBw %d Failed, error: %s\n", bw, err)
	} else {
		log.Printf("\tSetTunerBw %d Successful\n", bw)
	}

	if err = dev.ResetBuffer(); err == nil {
		log.Printf("\tResetBuffer Successful\n")
	} else {
		log.Printf("\tResetBuffer Failed - error: %s\n", err)
	}
	//---------- Get/Set Freq Correction ----------
	freqCorr := dev.GetFreqCorrection()
	log.Printf("\tGetFreqCorrection: %d\n", freqCorr)
	err = dev.SetFreqCorrection(globalSettings.PPM)
	if err != nil {
		log.Printf("\tSetFreqCorrection %d Failed, error: %s\n", globalSettings.PPM, err)
	} else {
		log.Printf("\tSetFreqCorrection %d Successful\n", globalSettings.PPM)
	}

	for uatSDR != -1 && globalSettings.UAT_Enabled {
		var buffer = make([]uint8, rtl.DefaultBufLength)
		nRead, err := dev.ReadSync(buffer, rtl.DefaultBufLength)
		if err != nil {
			log.Printf("\tReadSync Failed - error: %s\n", err)
			uatSDR = -1
			break
		} else {
			//			log.Printf("\tReadSync %d\n", nRead)
			buf := buffer[:nRead]
			godump978.InChan <- buf
		}
	}
	esSDR = -1
}

// Read from the godump978 channel - on or off.
func uatReader() {
	for {
		uat := <-godump978.OutChan
		o, msgtype := parseInput(uat)
		if o != nil && msgtype != 0 {
			relayMessage(msgtype, o)
		}
	}
}

// Watch for config/device changes.
func sdrWatcher() {
	timer := time.NewTicker(1 * time.Second)
	for {
		<-timer.C
		// Update device count.
		globalStatus.Devices = uint(rtl.GetDeviceCount())

		if (uatSDR != -1 || !globalSettings.UAT_Enabled) && (esSDR != -1 || !globalSettings.ES_Enabled) {
			// Nothing to do. All devices are set up and running or not required.
			continue
		}

		// Get the device strings for every device that we can.
		devs := make(map[int]string)
		for i := 0; i < int(globalStatus.Devices); i++ {
			dev, err := rtl.Open(i)
			if err != nil {
				continue
			}
			serial, err := getSDRSerial(dev)
			if err != nil {
				continue
			}
			devs[i] = serial
			dev.Close()
		}

		if uatSDR == -1 && globalSettings.UAT_Enabled {
			for devid, serial := range devs {
				if strings.HasPrefix(serial, "stratux:978") {
					uatSDR = devid
					delete(devs, devid)
					break
				}
			}
			if uatSDR == -1 {
				for devid, _ := range devs {
					uatSDR = devid
					break
				}
			}

			if uatSDR != -1 {
				log.Printf("UAT SDR: %d\n", uatSDR)
				go sdrReader()
			} else {
				log.Printf("Can't start UAT listening - no available RTL-SDR.\n")
			}
		}

		if esSDR == -1 && globalSettings.ES_Enabled {
			for devid, _ := range devs {
				esSDR = devid
				break
			}

			if esSDR != -1 {
				log.Printf("ES SDR: %d\n", esSDR)
				// Assume that this keeps running forever and won't change.
				//TODO: esSDR modify, watch if SDR disappears.
				err := exec.Command("/usr/bin/dump1090", "--net", "--device-index", strconv.Itoa(esSDR)).Run()
				if err != nil {
					log.Printf("Error executing /usr/bin/dump1090: %s\n", err.Error())
				}
			} else {
				log.Printf("Can't start ES listening - no available RTL-SDR.\n")
			}
		}
	}
}

func sdrInit() {
	godump978.Dump978Init()
	uatSDR = -1
	esSDR = -1
	go sdrWatcher()
	go uatReader()
	go godump978.ProcessDataFromChannel()
}
