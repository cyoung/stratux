// +build ignore

package main

import (
	"errors"
	//	"fmt"
	"../godump978"
	"log"
	"os"
	"os/signal"
	"syscall"

	rtl "github.com/jpoirier/gortlsdr"
	// "unsafe"
	"fmt"
)

func sigAbort(dev *rtl.Context) {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT)
	<-ch
	dev.Close()
	os.Exit(0)
}

func printUAT() {
	for {
		uat := <-godump978.OutChan
		log.Printf("godump978: %s\n", string(uat))

		fmt.Printf("%s;\n", uat)
	}
}

func main() {
	var err error
	var dev *rtl.Context

	//---------- Device Check ----------
	if c := rtl.GetDeviceCount(); c == 0 {
		log.Fatal("No devices found, exiting.\n")
	} else {
		for i := 0; i < c; i++ {
			m, p, s, err := rtl.GetDeviceUsbStrings(i)
			if err == nil {
				err = errors.New("")
			}
			log.Printf("GetDeviceUsbStrings %s - %s %s %s\n",
				err, m, p, s)
		}
	}

	log.Printf("===== Device name: %s =====\n", rtl.GetDeviceName(0))
	log.Printf("===== Running tests using device indx: 0 =====\n")

	//---------- Open Device ----------
	if dev, err = rtl.Open(0); err != nil {
		log.Fatal("\tOpen Failed, exiting\n")
	}
	defer dev.Close()
	go sigAbort(dev)

	//---------- Device Strings ----------
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

	/*
		//---------- Get/Set Freq Correction ----------
		freqCorr := dev.GetFreqCorrection()
		log.Printf("\tGetFreqCorrection: %d\n", freqCorr)
		err = dev.SetFreqCorrection(0) // 10ppm
		if err != nil {
			log.Printf("\tSetFreqCorrection %d Failed, error: %s\n", 0, err)
		} else {
			log.Printf("\tSetFreqCorrection %d Successful\n", 0)
		}


		//---------- Get/Set AGC Mode ----------
		if err = dev.SetAgcMode(false); err == nil {
			log.Printf("\tSetAgcMode off Successful\n")
		} else {
			log.Printf("\tSetAgcMode Failed, error: %s\n", err)
		}

		//---------- Get/Set Direct Sampling ----------
		if mode, err := dev.GetDirectSampling(); err == nil {
			log.Printf("\tGetDirectSampling Successful, mode: %s\n",
				rtl.SamplingModes[mode])
		} else {
			log.Printf("\tSetTestMode 'On' Failed - error: %s\n", err)
		}

		if err = dev.SetDirectSampling(rtl.SamplingNone); err == nil {
			log.Printf("\tSetDirectSampling 'On' Successful\n")
		} else {
			log.Printf("\tSetDirectSampling 'On' Failed - error: %s\n", err)
		}
	*/

	//---------- Get/Set Tuner IF Gain ----------
	// if err = SetTunerIfGain(stage, gain: int); err == nil {
	// 	log.Printf("\SetTunerIfGain Successful\n")
	// } else {
	// 	log.Printf("\tSetTunerIfGain Failed - error: %s\n", err)
	// }

	/*
		//---------- Get/Set test mode ----------
		if err = dev.SetTestMode(true); err == nil {
			log.Printf("\tSetTestMode 'On' Successful\n")
		} else {
			log.Printf("\tSetTestMode 'On' Failed - error: %s\n", err)
		}

		if err = dev.SetTestMode(false); err == nil {
			log.Printf("\tSetTestMode 'Off' Successful\n")
		} else {
			log.Printf("\tSetTestMode 'Off' Fail - error: %s\n", err)
		}

	*/

	//---------- Get/Set misc. streaming ----------
	go printUAT()
	go godump978.ProcessDataFromChannel()

	if err = dev.ResetBuffer(); err == nil {
		log.Printf("\tResetBuffer Successful\n")
	} else {
		log.Printf("\tResetBuffer Failed - error: %s\n", err)
	}

	for {
		var buffer = make([]uint8, rtl.DefaultBufLength)
		nRead, err := dev.ReadSync(buffer, rtl.DefaultBufLength)
		if err != nil {
			log.Printf("\tReadSync Failed - error: %s\n", err)
		} else {
			log.Printf("\tReadSync %d\n", nRead)
			//			buf := buffer[:nRead]
			//			godump978.InChan <- buf
		}
	}
}
