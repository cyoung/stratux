angular.module('appControllers').controller('SettingsCtrl', SettingsCtrl); // get the main module controllers set
SettingsCtrl.$inject = ['$rootScope', '$scope', '$state', '$location', '$window', '$http']; // Inject my dependencies


// create our controller function with all necessary logic
function SettingsCtrl($rootScope, $scope, $state, $location, $window, $http) {
	$scope.countryCodes = {
		"":"Unspecified",
		"AD":"Andorra",
		"AE":"United Arab Emirates",
		"AF":"Afghanistan",
		"AG":"Antigua & Barbuda",
		"AI":"Anguilla",
		"AL":"Albania",
		"AM":"Armenia",
		"AO":"Angola",
		"AQ":"Antarctica",
		"AR":"Argentina",
		"AS":"Samoa (American)",
		"AT":"Austria",
		"AU":"Australia",
		"AW":"Aruba",
		"AX":"Åland Islands",
		"AZ":"Azerbaijan",
		"BA":"Bosnia & Herzegovina",
		"BB":"Barbados",
		"BD":"Bangladesh",
		"BE":"Belgium",
		"BF":"Burkina Faso",
		"BG":"Bulgaria",
		"BH":"Bahrain",
		"BI":"Burundi",
		"BJ":"Benin",
		"BL":"St Barthelemy",
		"BM":"Bermuda",
		"BN":"Brunei",
		"BO":"Bolivia",
		"BQ":"Caribbean NL",
		"BR":"Brazil",
		"BS":"Bahamas",
		"BT":"Bhutan",
		"BV":"Bouvet Island",
		"BW":"Botswana",
		"BY":"Belarus",
		"BZ":"Belize",
		"CA":"Canada",
		"CC":"Cocos (Keeling) Islands",
		"CD":"Congo (Dem. Rep.)",
		"CF":"Central African Rep.",
		"CG":"Congo (Rep.)",
		"CH":"Switzerland",
		"CI":"Côte d'Ivoire",
		"CK":"Cook Islands",
		"CL":"Chile",
		"CM":"Cameroon",
		"CN":"China",
		"CO":"Colombia",
		"CR":"Costa Rica",
		"CU":"Cuba",
		"CV":"Cape Verde",
		"CW":"Curaçao",
		"CX":"Christmas Island",
		"CY":"Cyprus",
		"CZ":"Czech Republic",
		"DE":"Germany",
		"DJ":"Djibouti",
		"DK":"Denmark",
		"DM":"Dominica",
		"DO":"Dominican Republic",
		"DZ":"Algeria",
		"EC":"Ecuador",
		"EE":"Estonia",
		"EG":"Egypt",
		"EH":"Western Sahara",
		"ER":"Eritrea",
		"ES":"Spain",
		"ET":"Ethiopia",
		"FI":"Finland",
		"FJ":"Fiji",
		"FK":"Falkland Islands",
		"FM":"Micronesia",
		"FO":"Faroe Islands",
		"FR":"France",
		"GA":"Gabon",
		"GB":"Britain (UK)",
		"GD":"Grenada",
		"GE":"Georgia",
		"GF":"French Guiana",
		"GG":"Guernsey",
		"GH":"Ghana",
		"GI":"Gibraltar",
		"GL":"Greenland",
		"GM":"Gambia",
		"GN":"Guinea",
		"GP":"Guadeloupe",
		"GQ":"Equatorial Guinea",
		"GR":"Greece",
		"GS":"South Georgia & the South Sandwich Islands",
		"GT":"Guatemala",
		"GU":"Guam",
		"GW":"Guinea-Bissau",
		"GY":"Guyana",
		"HK":"Hong Kong",
		"HM":"Heard Island & McDonald Islands",
		"HN":"Honduras",
		"HR":"Croatia",
		"HT":"Haiti",
		"HU":"Hungary",
		"ID":"Indonesia",
		"IE":"Ireland",
		"IL":"Israel",
		"IM":"Isle of Man",
		"IN":"India",
		"IO":"British Indian Ocean Territory",
		"IQ":"Iraq",
		"IR":"Iran",
		"IS":"Iceland",
		"IT":"Italy",
		"JE":"Jersey",
		"JM":"Jamaica",
		"JO":"Jordan",
		"JP":"Japan",
		"KE":"Kenya",
		"KG":"Kyrgyzstan",
		"KH":"Cambodia",
		"KI":"Kiribati",
		"KM":"Comoros",
		"KN":"St Kitts & Nevis",
		"KP":"Korea (North)",
		"KR":"Korea (South)",
		"KW":"Kuwait",
		"KY":"Cayman Islands",
		"KZ":"Kazakhstan",
		"LA":"Laos",
		"LB":"Lebanon",
		"LC":"St Lucia",
		"LI":"Liechtenstein",
		"LK":"Sri Lanka",
		"LR":"Liberia",
		"LS":"Lesotho",
		"LT":"Lithuania",
		"LU":"Luxembourg",
		"LV":"Latvia",
		"LY":"Libya",
		"MA":"Morocco",
		"MC":"Monaco",
		"MD":"Moldova",
		"ME":"Montenegro",
		"MF":"St Martin (French)",
		"MG":"Madagascar",
		"MH":"Marshall Islands",
		"MK":"North Macedonia",
		"ML":"Mali",
		"MM":"Myanmar (Burma)",
		"MN":"Mongolia",
		"MO":"Macau",
		"MP":"Northern Mariana Islands",
		"MQ":"Martinique",
		"MR":"Mauritania",
		"MS":"Montserrat",
		"MT":"Malta",
		"MU":"Mauritius",
		"MV":"Maldives",
		"MW":"Malawi",
		"MX":"Mexico",
		"MY":"Malaysia",
		"MZ":"Mozambique",
		"NA":"Namibia",
		"NC":"New Caledonia",
		"NE":"Niger",
		"NF":"Norfolk Island",
		"NG":"Nigeria",
		"NI":"Nicaragua",
		"NL":"Netherlands",
		"NO":"Norway",
		"NP":"Nepal",
		"NR":"Nauru",
		"NU":"Niue",
		"NZ":"New Zealand",
		"OM":"Oman",
		"PA":"Panama",
		"PE":"Peru",
		"PF":"French Polynesia",
		"PG":"Papua New Guinea",
		"PH":"Philippines",
		"PK":"Pakistan",
		"PL":"Poland",
		"PM":"St Pierre & Miquelon",
		"PN":"Pitcairn",
		"PR":"Puerto Rico",
		"PS":"Palestine",
		"PT":"Portugal",
		"PW":"Palau",
		"PY":"Paraguay",
		"QA":"Qatar",
		"RE":"Réunion",
		"RO":"Romania",
		"RS":"Serbia",
		"RU":"Russia",
		"RW":"Rwanda",
		"SA":"Saudi Arabia",
		"SB":"Solomon Islands",
		"SC":"Seychelles",
		"SD":"Sudan",
		"SE":"Sweden",
		"SG":"Singapore",
		"SH":"St Helena",
		"SI":"Slovenia",
		"SJ":"Svalbard & Jan Mayen",
		"SK":"Slovakia",
		"SL":"Sierra Leone",
		"SM":"San Marino",
		"SN":"Senegal",
		"SO":"Somalia",
		"SR":"Suriname",
		"SS":"South Sudan",
		"ST":"Sao Tome & Principe",
		"SV":"El Salvador",
		"SX":"St Maarten (Dutch)",
		"SY":"Syria",
		"SZ":"Eswatini (Swaziland)",
		"TC":"Turks & Caicos Is",
		"TD":"Chad",
		"TF":"French Southern & Antarctic Lands",
		"TG":"Togo",
		"TH":"Thailand",
		"TJ":"Tajikistan",
		"TK":"Tokelau",
		"TL":"East Timor",
		"TM":"Turkmenistan",
		"TN":"Tunisia",
		"TO":"Tonga",
		"TR":"Turkey",
		"TT":"Trinidad & Tobago",
		"TV":"Tuvalu",
		"TW":"Taiwan",
		"TZ":"Tanzania",
		"UA":"Ukraine",
		"UG":"Uganda",
		"UM":"US minor outlying islands",
		"US":"United States",
		"UY":"Uruguay",
		"UZ":"Uzbekistan",
		"VA":"Vatican City",
		"VC":"St Vincent",
		"VE":"Venezuela",
		"VG":"Virgin Islands (UK)",
		"VI":"Virgin Islands (US)",
		"VN":"Vietnam",
		"VU":"Vanuatu",
		"WF":"Wallis & Futuna",
		"WS":"Samoa (western)",
		"YE":"Yemen",
		"YT":"Mayotte",
		"ZA":"South Africa",
		"ZM":"Zambia",
		"ZW":"Zimbabwe"
	};


	$scope.$parent.helppage = 'plates/settings-help.html';

	var toggles = ['UAT_Enabled', 'ES_Enabled', 'OGN_Enabled', 'AIS_Enabled', 'APRS_Enabled', 'Ping_Enabled', 'OGNI2CTXEnabled', 'GPS_Enabled', 'IMU_Sensor_Enabled',
		'BMP_Sensor_Enabled', 'DisplayTrafficSource', 'DEBUG', 'ReplayLog', 'TraceLog', 'AHRSLog', 'PersistentLogging', 'GDL90MSLAlt_Enabled', 'EstimateBearinglessDist', 'DarkMode'];

	var settings = {};
	for (var i = 0; i < toggles.length; i++) {
		settings[toggles[i]] = undefined;
	}
	$scope.update_files = '';

	$http.get(URL_STATUS_GET).then(function(response) {
		var status = angular.fromJson(response.data);
		var gpsHardwareCode = (status.GPS_detected_type & 0x0f);
		$scope.hasOgnTracker = gpsHardwareCode == 3 || status.OGN_tx_enabled;
		$scope.hasGXTracker = gpsHardwareCode == 15;
	});

	function loadSettings(data) {
		settings = angular.fromJson(data);
		// consider using angular.extend()
		$scope.rawSettings = angular.toJson(data, true);
		$scope.visible_serialout = false;
		if ((settings.SerialOutputs !== undefined) && (settings.SerialOutputs !== null)) {
			for (var k in settings.SerialOutputs) {
				$scope.Baud = settings.SerialOutputs[k].Baud;
				$scope.visible_serialout = true;
			}
		}

		$scope.DarkMode = settings.DarkMode;

		$scope.UAT_Enabled = settings.UAT_Enabled;
		$scope.ES_Enabled = settings.ES_Enabled;
		$scope.OGN_Enabled = settings.OGN_Enabled;
		$scope.AIS_Enabled = settings.AIS_Enabled;
		$scope.APRS_Enabled = settings.APRS_Enabled;
		$scope.Ping_Enabled = settings.Ping_Enabled;
		$scope.GPS_Enabled = settings.GPS_Enabled;
		$scope.OGNI2CTXEnabled = settings.OGNI2CTXEnabled;

		$scope.IMU_Sensor_Enabled = settings.IMU_Sensor_Enabled;
		$scope.BMP_Sensor_Enabled = settings.BMP_Sensor_Enabled;
		$scope.DisplayTrafficSource = settings.DisplayTrafficSource;
		$scope.DEBUG = settings.DEBUG;
		$scope.ReplayLog = settings.ReplayLog;
		$scope.TraceLog = settings.TraceLog;
		$scope.AHRSLog = settings.AHRSLog;
		$scope.PersistentLogging = settings.PersistentLogging;

		$scope.PPM = settings.PPM;
		$scope.AltitudeOffset = settings.AltitudeOffset;
		$scope.WatchList = settings.WatchList;
		$scope.OwnshipModeS = settings.OwnshipModeS;
		$scope.DeveloperMode = settings.DeveloperMode;
		$scope.GLimits = settings.GLimits;
		$scope.GDL90MSLAlt_Enabled = settings.GDL90MSLAlt_Enabled;
		$scope.EstimateBearinglessDist = settings.EstimateBearinglessDist
		$scope.StaticIps = settings.StaticIps;

		$scope.WiFiCountry = settings.WiFiCountry;
		$scope.WiFiSSID = settings.WiFiSSID;
		$scope.WiFiPassphrase = settings.WiFiPassphrase;
		$scope.WiFiSecurityEnabled = settings.WiFiSecurityEnabled;
		$scope.WiFiChannel = settings.WiFiChannel;
		$scope.WiFiIPAddress = settings.WiFiIPAddress;

		$scope.WiFiMode = settings.WiFiMode.toString();
		$scope.WiFiDirectPin = settings.WiFiDirectPin;

		$scope.WiFiClientNetworks = settings.WiFiClientNetworks;
		$scope.WiFiInternetPassThroughEnabled = settings.WiFiInternetPassThroughEnabled;

		$scope.Channels = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11];

		$scope.OGNAddrType = settings.OGNAddrType.toString();
		$scope.OGNAddr = settings.OGNAddr;
		$scope.OGNAcftType = settings.OGNAcftType.toString();
		$scope.OGNPilot = settings.OGNPilot;
		$scope.OGNReg = settings.OGNReg;
		$scope.OGNTxPower = settings.OGNTxPower;

		$scope.GXAcftType = settings.GXAcftType;
		$scope.GXPilot = settings.GXPilot;
		$scope.GXAddr = settings.GXAddr.toString(16);

		$scope.PWMDutyMin = settings.PWMDutyMin;

		// Update theme
		$scope.$parent.updateTheme($scope.DarkMode);


		$scope.CountryCodeList = countryCodes;
	}

	function getSettings() {
		// Simple GET request example (note: response is asynchronous)
		$http.get(URL_SETTINGS_GET).
		then(function (response) {
			loadSettings(response.data);
			// $scope.$apply();
		}, function (response) {
			$scope.rawSettings = "error getting settings";
			for (i = 0; i < toggles.length; i++) {
				settings[toggles[i]] = false;
			}
		});
	}

	function setSettings(msg) {
		// Simple POST request example (note: response is asynchronous)
		$http.post(URL_SETTINGS_SET, msg).
		then(function (response) {
			loadSettings(response.data);
			// $scope.$apply();
		}, function (response) {
			$scope.rawSettings = "error setting settings";
			for (i = 0; i < toggles.length; i++) {
				settings[toggles[i]] = false;
			}

		});
	}

	getSettings();

	// Reset all settings from a button on the page
	$scope.resetSettings = function () {
		getSettings();
	};

	$scope.$watchGroup(toggles, function (newValues, oldValues, scope) {
		var newsettings = {};
		var dirty = false;
		for (i = 0; i < newValues.length; i++) {
			if ((newValues[i] !== undefined) && (settings[toggles[i]] !== undefined)) {
				if (newValues[i] !== settings[toggles[i]]) {
					settings[toggles[i]] = newValues[i];
					newsettings[toggles[i]] = newValues[i];
					dirty = true;
				}
			}
		}
		if (dirty) {
			// console.log(angular.toJson(newsettings));
			setSettings(angular.toJson(newsettings));
		}
	});

	$scope.updateppm = function () {
		settings["PPM"] = 0;
		if (($scope.PPM !== undefined) && ($scope.PPM !== null)) {
			settings["PPM"] = parseInt($scope.PPM);
			var newsettings = {
				"PPM": settings["PPM"]
			};
			// console.log(angular.toJson(newsettings));
			setSettings(angular.toJson(newsettings));
		}
	};

	$scope.updatePWMDutyMin = function() {
		settings['PWMDutyMin'] = 0;
		if ($scope.PWMDutyMin !== undefined && $scope.PWMDutyMin !== null) {
			settings['PWMDutyMin'] = parseInt($scope.PWMDutyMin);
			var newsettings = {
				'PWMDutyMin': settings['PWMDutyMin']
			};
			setSettings(angular.toJson(newsettings));
		}
	}

	$scope.updateBaud = function () {
		settings["Baud"] = 0;
		if (($scope.Baud !== undefined) && ($scope.Baud !== null)) {
			settings["Baud"] = parseInt($scope.Baud);
			var newsettings = {
				"Baud": settings["Baud"]
			};
			// console.log(angular.toJson(newsettings));
			setSettings(angular.toJson(newsettings));
		}
	};

	$scope.updatewatchlist = function () {
		if ($scope.WatchList !== settings["WatchList"]) {
			settings["WatchList"] = "";
			if ($scope.WatchList !== undefined) {
				settings["WatchList"] = $scope.WatchList.toUpperCase();
			}
			var newsettings = {
				"WatchList": settings["WatchList"]
			};
			// console.log(angular.toJson(newsettings));
			setSettings(angular.toJson(newsettings));
		}
	};

	$scope.updatemodes = function () {
		if ($scope.OwnshipModeS !== settings["OwnshipModeS"]) {
			settings["OwnshipModeS"] = $scope.OwnshipModeS.toUpperCase();
			var newsettings = {
				"OwnshipModeS": $scope.OwnshipModeS.toUpperCase()
			};
			// console.log(angular.toJson(newsettings));
			setSettings(angular.toJson(newsettings));
		}
	};

	$scope.updatestaticips = function () {
		if ($scope.StaticIps !== settings.StaticIps) {
			var newsettings = {
				"StaticIps": $scope.StaticIps === undefined? "" : $scope.StaticIps.join(' ')
			};
			// console.log(angular.toJson(newsettings));
			setSettings(angular.toJson(newsettings));
		}
	};

	$scope.updatealtitudeoffset = function () {
		if ($scope.AltitudeOffset !== undefined && $scope.AltitudeOffset !== null && $scope.AltitudeOffset !== settings["AltitudeOffset"]) {
			settings["AltitudeOffset"] = parseInt($scope.AltitudeOffset);
			var newsettings = {
				"AltitudeOffset": settings["AltitudeOffset"]
			};
			// console.log(angular.toJson(newsettings));
			setSettings(angular.toJson(newsettings));
		}
	};

	$scope.updateGLimits = function () {
		if ($scope.GLimits !== settings["GLimits"]) {
			settings["GLimits"] = $scope.GLimits;
			var newsettings = {
				"GLimits": settings["GLimits"]
			};
			// console.log(angular.toJson(newsettings));
			setSettings(angular.toJson(newsettings));
		}
	};

	$scope.postShutdown = function () {
		$window.location.href = "/";
		$location.path('/home');
		$http.post(URL_SHUTDOWN).
		then(function (response) {
			// do nothing
			// $scope.$apply();
		}, function (response) {
			// do nothing
		});
	};

	$scope.postReboot = function () {
		$window.location.href = "/";
		$location.path('/home');
		$http.post(URL_REBOOT).
		then(function (response) {
			// do nothing
			// $scope.$apply();
		}, function (response) {
			// do nothing
		});
	};

	$scope.setUploadFile = function (files) {
		$scope.update_files = files;
		$scope.$apply();
	};

	$scope.resetUploadFile = function () {
		$scope.update_files = '';
		$scope.$apply();
	};

	$scope.uploadFile = function () {
		var fd = new FormData();
		//Take the first selected file
		var file = $scope.update_files[0];
		// check for empty string
		if (file === undefined || file === null) {
			alert ("update file not selected");
			return;
		}
		var filename = file.name;
		// check for expected file naming convention
		var re = /^update.*\.sh$/;
		if (!re.exec(filename)) {
			alert ("file does not appear to be an update");
			return;
		}

		fd.append("update_file", file);
		$scope.uploading_update = true;
		$scope.$apply();

		$http.post(URL_UPDATE_UPLOAD, fd, {
			withCredentials: true,
			headers: {
				'Content-Type': undefined
			},
			transformRequest: angular.identity
		}).success(function (data) {
			$scope.uploading_update = false;
			alert("success. wait 5 minutes and refresh home page to verify new version.");
			window.location.replace("/");
			$scope.$apply();
		}).error(function (data) {
			$scope.uploading_update = false;
			alert("error");
			$scope.$apply();
		});
	};

	$scope.setOrientation = function(action) {
		// console.log("sending " + action + " message.");
		$http.post(URL_AHRS_ORIENT, action).
		then(function (response) {
			// console.log("sent " + action + " message.");
		}, function(response) {
			// failure: cancel the calibration
			// console.log(response.data);
			$scope.Orientation_Failure_Message = response.data;
			$scope.Ui.turnOff('modalCalibrateDone');
			$scope.Ui.turnOn("modalCalibrateFailed");
		});
	};

	$scope.addWiFiClientNetwork = function () {
		$scope.WiFiClientNetworks.push({
			SSID: '',
			Password: ''
		});
		$scope.$apply();
	};

	$scope.removeWiFiClientNetwork = function (Network) {
		var idx = $scope.WiFiClientNetworks.indexOf(Network);
		if (idx >= 0) {
			$scope.WiFiClientNetworks.splice(idx, 1);
		}
		$scope.$apply();
	};

	$scope.updateWiFi = function(action) {
		$scope.WiFiErrors = {
			'WiFiSSID': '',
			'WiFiPassphrase': '',
			'Errors': false
		};

		if (($scope.WiFiSSID === undefined) || ($scope.WiFiSSID === null) || !isValidSSID($scope.WiFiSSID)) {
				$scope.WiFiErrors.WiFiSSID = "Your Network Name(\"SSID\") must be at least 1 character " +
					"but not more than 32 characters. It can only contain a-z, A-Z, 0-9, _ or -.";
				$scope.WiFiErrors.Errors = true;
			}

		if ($scope.WiFiSecurityEnabled) {
			if (!isValidWPA($scope.WiFiPassphrase)) {
				$scope.WiFiErrors.WiFiPassphrase = "Your WiFi Password, " + $scope.WiFiPassphrase +
					", contains invalid characters.";
				$scope.WiFiErrors.Errors = true;
			}
			if ($scope.WiFiPassphrase.length < 8 || $scope.WiFiPassphrase.length > 63 ) {
				$scope.WiFiErrors.WiFiPassphrase = "Your WiFi Password must be between 8 and 63 characters long.";
				$scope.WiFiErrors.Errors = true;
			}
		}

		if (!$scope.WiFiErrors.Errors) {
			var newsettings = {
				"WiFiCountry": $scope.WiFiCountry,
				"WiFiSSID" :  $scope.WiFiSSID,
				"WiFiSecurityEnabled" : $scope.WiFiSecurityEnabled,
				"WiFiPassphrase" : $scope.WiFiPassphrase,
				"WiFiChannel" : parseInt($scope.WiFiChannel),
				"WiFiIPAddress" : $scope.WiFiIPAddress,
				"WiFiMode" : parseInt($scope.WiFiMode),
				"WiFiDirectPin": $scope.WiFiDirectPin,
				"WiFiClientNetworks": $scope.WiFiClientNetworks,
				"WiFiInternetPassThroughEnabled": $scope.WiFiInternetPassThroughEnabled
			};

			// console.log(angular.toJson(newsettings));
			setSettings(angular.toJson(newsettings));
			$scope.Ui.turnOn("modalSuccessWiFi");
		} else {
			$scope.Ui.turnOn("modalErrorWiFi");
		}
	};

	$scope.wifiModeStr = function() {
		switch(parseInt($scope.WiFiMode)) {
			case 0: return "AP";
			case 1: return "WiFi-Direct";
			case 2: return "AP+Client";
		}
		return "???";
	}

	$scope.updateOgnTrackerConfig = function(action) {
		var newsettings = {
			"OGNAddrType": parseInt($scope.OGNAddrType),
			"OGNAddr": $scope.OGNAddr,
			"OGNAcftType": parseInt($scope.OGNAcftType),
			"OGNPilot": $scope.OGNPilot,
			"OGNReg": $scope.OGNReg,
			"OGNTxPower": $scope.OGNTxPower
		};
		setSettings(angular.toJson(newsettings));

		// reload settings after a short time, to check if OGN tracker actually accepted the settings
		setTimeout(function() {
			getSettings();
		}, 1000);
	}

	$scope.updateGXTrackerConfig = function(action) {
		var newsettings = {
			"GXAddr": $scope.GXAddr,
			"GXAcftType": parseInt($scope.GXAcftType),
			"GXPilot": $scope.GXPilot,
		};
		setSettings(angular.toJson(newsettings));

		// reload settings after a short time, to check if GX tracker actually accepted the settings
		setTimeout(function() {
			getSettings();
		}, 5000);
	}
}

function isValidSSID(str) { return /^[a-zA-Z0-9()! \._-]{1,32}$/g.test(str); }
function isValidWPA(str) { return /^[\u0020-\u007e]{8,63}$/g.test(str); }
function isValidPin(str) { return /^([\d]{4}|[\d]{8})$/g.test(str); }

angular.module('appControllers')
	.directive('hexInput', function() { // directive for ownship hex code validation
		return {
			require: 'ngModel',
			link: function(scope, element, attr, ctrl) {
				function hexValidation(value) {
					var valid = /^$|^([0-9A-Fa-f]{6},?)*$/.test(value);
					ctrl.$setValidity('FAAHex', valid);
					if (valid) {
						return value;
					} else {
						return "";
					}
				}
				ctrl.$parsers.push(hexValidation);
			}
		};
	})
	.directive('watchlistInput', function() { // directive for ICAO space-separated watch list validation
		return {
			require: 'ngModel',
			link: function(scope, element, attr, ctrl) {
				function watchListValidation(value) {
					// The list of METAR locations at http://www.aviationweather.gov/docs/metar/stations.txt
					// lists only 4-letter/number ICAO codes.
					var r = "[A-Za-z0-9]{4}";
					var valid = (new RegExp("^(" + r + "( " + r + ")*|)$", "g")).test(value);
					ctrl.$setValidity('ICAOWatchList', valid);
					if (valid) {
						return value;
					} else {
						return "";
					}
				}
				ctrl.$parsers.push(watchListValidation);
			}
		};
	})
	.directive('ssidInput', function() { // directive for WiFi SSID validation
		return {
			require: 'ngModel',
			link: function(scope, element, attr, ctrl) {
				function ssidValidation(value) {
					var valid = isValidSSID(value);
					ctrl.$setValidity('ssid', valid);
					if (valid) {
						return value;
					} else {
						return "";
					}
				}
				ctrl.$parsers.push(ssidValidation);
			}
		};
	})
	.directive('wpaInput', function() { // directive for WiFi WPA Passphrase validation
		return {
			require: 'ngModel',
			link: function(scope, element, attr, ctrl) {
				function wpaValidation(value) {
					var valid = isValidWPA(value);
					ctrl.$setValidity('wpaPassphrase', valid);
					if (valid) {
						return value;
					} else {
						return "";
					}
				}
				ctrl.$parsers.push(wpaValidation);
			}
		};
	})
	.directive('pinInput', function() {
		return {
			require: 'ngModel',
			link: function(scope, element, attr, ctrl) {
				function pinValidation(value) {
					var valid = isValidPin(value);
					ctrl.$setValidity('WiFiDirectPin', valid);
					if (valid) {
						return value;
					} else {
						return "";
					}
				}
				ctrl.$parsers.push(pinValidation);
			}
		};	
	})
	.directive('ipListInput', function() { // directive for validation of list of IP addresses
		return {
			require: 'ngModel',
			link: function(scope, element, attr, ctrl) {
				function ipListValidation(value) {
					var r = "(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)";
					var valid = (new RegExp("^(" + r + "( " + r + ")*|)$", "g")).test(value);
					ctrl.$setValidity('ipList', valid);
					if (valid) {
						return value;
					} else {
						return "";
					}
				}
				ctrl.$parsers.push(ipListValidation);
			}
		};
	})
	.directive('ipAddrInput', function() { // directive for validation of a single IP address
		return {
			require: 'ngModel',
			link: function(scope, element, attr, ctrl) {
				function ipListValidation(value) {
					var r = "^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$";
					var valid = (new RegExp(r, "g")).test(value);
					ctrl.$setValidity('ipAddr', valid);
					if (valid) {
						return value;
					} else {
						return "";
					}
				}
				ctrl.$parsers.push(ipListValidation);
			}
		};
	})
	.directive('gLimitsInput', function() { // directive for validation of list of G Limits
		return {
			require: 'ngModel',
			link: function(scope, element, attr, ctrl) {
				function gLimitsValidation(value) {
					var r = "[-+]?[0-9]*\.?[0-9]+";
					var valid = (new RegExp("^(" + r + "( " + r + ")*|)$", "g")).test(value);
					ctrl.$setValidity('gLimits', valid);
					if (valid) {
						return value;
					} else {
						return "";
					}
				}
				ctrl.$parsers.push(gLimitsValidation);
			}
		};
	})
	.directive('pilotnameInput', function() {
		return {
			require: 'ngModel',
			link: function(scope, element, attr, ctrl) {
				function pilotnameValidation(value) {
					var r = "^[0-9a-zA-Z_]*$";
					var valid = new RegExp(r).test(value);
					ctrl.$setValidity('pilotname', valid);
					if (valid)
						return value;
					else
						return "";
				}
				ctrl.$parsers.push(pilotnameValidation);
			}
		}
	}).directive('ognregInput', function() {
		return {
			require: 'ngModel',
			link: function(scope, element, attr, ctrl) {
				function ognregValidation(value) {
					var r = "^[0-9a-zA-Z_\-]*$";
					var valid = new RegExp(r).test(value);
					ctrl.$setValidity('ognreg', valid);
					if (valid)
						return value;
					else
						return "";
				}
				ctrl.$parsers.push(ognregValidation);
			}
		}
	});
