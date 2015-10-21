package main

import (
	"../godump978"
	rtl "github.com/jpoirier/gortlsdr"
	"log"
	"time"
)

var uatSDR int // Index.
var esSDR int  // Index.

var maxSignalStrength int

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
	m, p, s, err := dev.GetUsbStrings()
	if err != nil {
		log.Printf("\tGetUsbStrings Failed - error: %s\n", err)
	} else {
		log.Printf("\tGetUsbStrings - %s %s %s\n", m, p, s)
	}
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

	for uatSDR != -1 {
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

		if uatSDR == -1 && globalSettings.UAT_Enabled {
			if globalStatus.Devices == 0 {
				log.Printf("No RTL-SDR devices.\n")
				continue
			}
			uatSDR = 0
			go sdrReader()
		}
		if esSDR == -1 && globalSettings.ES_Enabled {
			if globalStatus.Devices == 0 || (globalStatus.Devices == 1 && globalSettings.UAT_Enabled) {
				log.Printf("Not enough RTL-SDR devices.\n")
			}
			esSDR = 1
		}
	}
}

func sdrInit() {
	uatSDR = -1
	esSDR = -1
	go sdrWatcher()
	go uatReader()
	godump978.Dump978Init()
	go godump978.ProcessDataFromChannel()
}
