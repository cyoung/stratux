/*
	Copyright (c) 2019 Adrian Batzill
	Distributable under the terms of The "BSD New" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	networksettings.go: Management functions for network settings (wpa_supplicant, IP, DHCP)
*/

package main

import (
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

const (
	WifiModeAp = 0
	WifiModeDirect = 1
	WifiModeApClient = 2
)

// NetworkTemplateParams is passed to the template engine to write settings
type NetworkTemplateParams struct {
	WiFiMode         int
	IpAddr           string
	IpPrefix         string
	DhcpRangeStart   string
	DhcpRangeEnd     string
	WiFiSSID         string
	WiFiChannel      int
	WiFiDirectPin    string
	WiFiPassPhrase   string
	WiFiClientNetworks []wifiClientNetwork
	WiFiInternetPassThroughEnabled bool
}
type wifiClientNetwork struct {
	SSID     string
	Password string
}

var hasChanged bool

func setWifiSSID(ssid string) {
	if ssid != globalSettings.WiFiSSID {
		globalSettings.WiFiSSID = ssid
		hasChanged = true
	}
}

func setWifiPassphrase(passphrase string) {
	if passphrase != globalSettings.WiFiPassphrase {
		globalSettings.WiFiPassphrase = passphrase
		hasChanged = true
	}
}

func setWifiChannel(channel int) {
	if channel != globalSettings.WiFiChannel {
		globalSettings.WiFiChannel = channel;
		hasChanged = true
	}
}

func setWifiSecurityEnabled(enabled bool) {
	if globalSettings.WiFiSecurityEnabled != enabled {
		globalSettings.WiFiSecurityEnabled = enabled;
		hasChanged = true
	}
}

func setWifiIPAddress(ip string) {
	match, err := regexp.MatchString(`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`, ip)
	if err == nil && match {
		if globalSettings.WiFiIPAddress != ip {
			globalSettings.WiFiIPAddress = ip
			hasChanged = true
		}
	} else {
		log.Printf("Ignoring invalid IP Address: " + ip)
	}
}

func setWiFiMode(mode int) {
	if globalSettings.WiFiMode != mode {
		globalSettings.WiFiMode = mode
		hasChanged = true
	}
}

func setWifiDirectPin(pin string) {
	if globalSettings.WiFiDirectPin != pin {
		globalSettings.WiFiDirectPin = pin
		hasChanged = true
	}
}

func setWifiClientNetworks(networks []wifiClientNetwork) {
	if len(globalSettings.WiFiClientNetworks) != len(networks) {
		globalSettings.WiFiClientNetworks = networks
		hasChanged = true
		return
	}

	for i, net := range networks {
		if globalSettings.WiFiClientNetworks[i].SSID != net.SSID || globalSettings.WiFiClientNetworks[i].Password != net.Password {
			globalSettings.WiFiClientNetworks = networks
			hasChanged = true
			return
		}
	}
}

func setWifiInternetPassthroughEnabled(enabled bool) {
	if globalSettings.WiFiInternetPassThroughEnabled != enabled {
		globalSettings.WiFiInternetPassThroughEnabled = enabled;
		hasChanged = true;
	}
}


// if onlyWriteFiles is true, we only write the config files. Otherwise we also reconfigure the network
// Also, if we only write the files, this function runs synchroneously. Otherwise the long-running network reconfiguration is done async.
func applyNetworkSettings(force bool, onlyWriteFiles bool) {
	if !hasChanged && !force {
		return
	}
	hasChanged = false

	// Prepare all template strings and write settings files, then ifdown/ifup wlan0
	ipAddr := globalSettings.WiFiIPAddress
	log.Printf("Applying new network settings for IP %s", ipAddr);
	if ipAddr == "" {
		ipAddr = "192.168.10.1"
	}
	ipParts := strings.Split(ipAddr, ".")
	
	ipPrefix := ipParts[0] + "." + ipParts[1] + "." + ipParts[2]

	myIP, _ := strconv.Atoi(ipParts[3])
	dhcpRangeStart := ipPrefix + ".10"
	dhcpRangeEnd := ipPrefix + ".50"
	if myIP >= 10 && myIP <= 50 {
		// In case the stratux ip is inside its dhcp range, we move the dhcp range back to something else..
		dhcpRangeStart = ipPrefix + ".60"
		dhcpRangeEnd = ipPrefix + ".110"
	}

	var tplSettings NetworkTemplateParams
	tplSettings.WiFiMode = globalSettings.WiFiMode
	tplSettings.IpAddr = ipAddr
	tplSettings.IpPrefix = ipPrefix
	tplSettings.DhcpRangeStart = dhcpRangeStart
	tplSettings.DhcpRangeEnd = dhcpRangeEnd
	tplSettings.WiFiChannel = globalSettings.WiFiChannel
	tplSettings.WiFiSSID = globalSettings.WiFiSSID
	tplSettings.WiFiDirectPin = globalSettings.WiFiDirectPin
	tplSettings.WiFiClientNetworks = globalSettings.WiFiClientNetworks
	tplSettings.WiFiInternetPassThroughEnabled = globalSettings.WiFiInternetPassThroughEnabled
	
	if tplSettings.WiFiChannel == 0 {
		tplSettings.WiFiChannel = 1
	}
	if globalSettings.WiFiSecurityEnabled || tplSettings.WiFiMode == WifiModeDirect {
		tplSettings.WiFiPassPhrase = globalSettings.WiFiPassphrase
	}
	
	if tplSettings.WiFiSSID == "" {
		tplSettings.WiFiSSID = "stratux"
	}

	f := func() {
		time.Sleep(time.Second)
		if !onlyWriteFiles {
			cmd := exec.Command("ifdown", "wlan0")
			if err := cmd.Start(); err != nil {
				log.Printf("Error shutting down WiFi: %s\n", err.Error())
			}
			if err := cmd.Wait(); err != nil {
				log.Printf("Error shutting down WiFi: %s\n", err.Error())
			}
		}

		overlayctl("unlock")
		writeTemplate(STRATUX_HOME + "/cfg/stratux-dnsmasq.conf.template", "/overlay/robase/etc/dnsmasq.d/stratux-dnsmasq.conf", tplSettings)
		writeTemplate(STRATUX_HOME + "/cfg/interfaces.template", "/overlay/robase/etc/network/interfaces", tplSettings)
		writeTemplate(STRATUX_HOME + "/cfg/wpa_supplicant.conf.template", "/overlay/robase/etc/wpa_supplicant/wpa_supplicant.conf", tplSettings)
		writeTemplate(STRATUX_HOME + "/cfg/wpa_supplicant_ap.conf.template", "/overlay/robase/etc/wpa_supplicant/wpa_supplicant_ap.conf", tplSettings)
		overlayctl("lock")

		if !onlyWriteFiles {
			cmd := exec.Command("ifup", "wlan0")
			if err := cmd.Start(); err != nil {
				log.Printf("Error starting WiFi: %s\n", err.Error())
			}
			if err := cmd.Wait(); err != nil {
				log.Printf("Error starting WiFi: %s\n", err.Error())
			}
		}
	}


	if onlyWriteFiles {
		f()
	} else {
		go f()
	}
}




func writeTemplate(tplFile string, outFile string, settings NetworkTemplateParams) {
	configTemplate, err := template.ParseFiles(tplFile)
	if err != nil {
		log.Printf("Network Settings: Unable to read settings template %s: %s", tplFile, err)
		return
	}

	outputFile, err := os.Create(outFile)
	defer outputFile.Close()
	if err != nil {
		log.Printf("Network Settings: Unable to open output file %s: %s", outFile, err)
		return
	}

	err = configTemplate.Execute(outputFile, settings)
	if err != nil {
		log.Printf("Network Settings: Unable to execute template substitution %s: %s", outFile, err)
		return
	}
	outputFile.Sync()
}