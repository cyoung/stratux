angular.module('appControllers').controller('SettingsCtrl', SettingsCtrl); // get the main module controllers set
SettingsCtrl.$inject = ['$rootScope', '$scope', '$state', '$location', '$window', '$http']; // Inject my dependencies


// create our controller function with all necessary logic
function SettingsCtrl($rootScope, $scope, $state, $location, $window, $http) {

	$scope.$parent.helppage = 'plates/settings-help.html';

	var toggles = ['UAT_Enabled', 'ES_Enabled', 'Ping_Enabled', 'GPS_Enabled', 'IMU_Sensor_Enabled',
		'BMP_Sensor_Enabled', 'DisplayTrafficSource', 'DEBUG', 'ReplayLog', 'AHRSLog'];
	var settings = {};
	for (var i = 0; i < toggles.length; i++) {
		settings[toggles[i]] = undefined;
	}
	$scope.update_files = '';

	function loadSettings(data) {
		settings = angular.fromJson(data);
		// consider using angular.extend()
		$scope.rawSettings = angular.toJson(data, true);
		$scope.visible_serialout = false;
		if ((settings.SerialOutputs !== undefined) && (settings.SerialOutputs !== null) && (settings.SerialOutputs['/dev/serialout0'] !== undefined)) {
			$scope.Baud = settings.SerialOutputs['/dev/serialout0'].Baud;
			$scope.visible_serialout = true;
		}
		$scope.UAT_Enabled = settings.UAT_Enabled;
		$scope.ES_Enabled = settings.ES_Enabled;
		$scope.Ping_Enabled = settings.Ping_Enabled;
		$scope.GPS_Enabled = settings.GPS_Enabled;

		$scope.IMU_Sensor_Enabled = settings.IMU_Sensor_Enabled;
		$scope.BMP_Sensor_Enabled = settings.BMP_Sensor_Enabled;
		$scope.DisplayTrafficSource = settings.DisplayTrafficSource;
		$scope.DEBUG = settings.DEBUG;
		$scope.ReplayLog = settings.ReplayLog;
		$scope.AHRSLog = settings.AHRSLog;

		$scope.PPM = settings.PPM;
		$scope.WatchList = settings.WatchList;
		$scope.OwnshipModeS = settings.OwnshipModeS;
		$scope.DeveloperMode = settings.DeveloperMode;
        $scope.GLimits = settings.GLimits;
		$scope.StaticIps = settings.StaticIps;

        $scope.WiFiSSID = settings.WiFiSSID;
        $scope.WiFiPassphrase = settings.WiFiPassphrase;
        $scope.WiFiSecurityEnabled = settings.WiFiSecurityEnabled;
        $scope.WiFiChannel = settings.WiFiChannel;
        $scope.WiFiButtonOKText = "Submit WiFi Changes";
        $scope.WiFiButtonSSIDLengthText = "SSID must be 1-32 characters long";
        $scope.WiFiButtonSSIDCharText = "SSID can contain only A-Z, a-z, 0-9, ()_- or {space}";
        $scope.WiFiButtonWPALengthText = "Passphrase must be 8-63 characters long";
        $scope.WiFiButtonWPACharText = "Passphrase contains invalid characters";
        $scope.WiFiButtonText = $scope.WiFiButtonOKText;
        $scope.WiFiButtonNormalStyle = {"font-size": "14px"};
        $scope.WiFiButtonSmallStyle = {"font-size": "11px"};

        $scope.Channels = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11];
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
		if (($scope.PPM !== undefined) && ($scope.PPM !== null) && ($scope.PPM !== settings["PPM"])) {
			settings["PPM"] = parseInt($scope.PPM);
			var newsettings = {
				"PPM": settings["PPM"]
			};
			// console.log(angular.toJson(newsettings));
			setSettings(angular.toJson(newsettings));
		}
	};

	$scope.updateBaud = function () {
		settings["Baud"] = 0;
		if (($scope.Baud !== undefined) && ($scope.Baud !== null) && ($scope.Baud !== settings["Baud"])) {
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
				"StaticIps": $scope.StaticIps === undefined ? "" : $scope.StaticIps.join(' ')
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

		$http.post(URL_UPDATE_UPLOAD, fd, {
			withCredentials: true,
			headers: {
				'Content-Type': undefined
			},
			transformRequest: angular.identity
		}).success(function (data) {
			alert("success. wait 60 seconds and refresh home page to verify new version.");
			window.location.replace("/");
		}).error(function (data) {
			alert("error");
		});

	};

	$scope.setOrientation = function(action) {
		$http.post(URL_AHRS_ORIENT, action).
		then(function (response) {
		}, function(response) {
			// failure: cancel the orientation procedure.
			$scope.Orientation_Failure_Message = response.data;
			$scope.Ui.turnOff('modalCalibrateDone');
			$scope.Ui.turnOn("modalCalibrateFailed");
		});
    };

    $scope.calibrateGyros = function() {
        $http.post(URL_AHRS_CAL).then(function (response) {
        }, function (response) {
            $scope.Calibration_Failure_Message = response.data;
            $scope.Ui.turnOff("modalCalibrateGyros");
            $scope.Ui.turnOn("modalCalibrateGyrosFailed");
        });
    };

    $scope.ssidValid = isValidSSID($scope.WiFiSSID);
    $scope.wpaValid = isValidWPA($scope.WiFiPassphrase);

    $scope.updateWiFiErrorState = function(ssid, passphrase) {
        $scope.WiFiButtonText = $scope.WiFiButtonOKText;
        $scope.WiFiButtonStyle = $scope.WiFiButtonNormalStyle;

        $scope.ssidValid = isValidSSID(ssid);
        $scope.WiFiSettings.$setValidity('ssid', $scope.ssidValid);

        if ($scope.WiFiSecurityEnabled) {
            $scope.wpaValid = isValidWPA(passphrase);
        } else {
            $scope.wpaValid = true;
        }
        $scope.WiFiSettings.$setValidity('wpaPassphrase', $scope.wpaValid);

        if (!$scope.wpaValid) {
            if ((typeof passphrase === 'undefined') || (passphrase === null) ||
                (passphrase.length < 8) || (passphrase.length > 63)) {
                $scope.WiFiButtonText = $scope.WiFiButtonWPALengthText;
            } else {
                $scope.WiFiButtonText = $scope.WiFiButtonWPACharText;
            }
        }

        if (!$scope.ssidValid) {
            if ((typeof ssid === 'undefined') || (ssid === null) ||
                (ssid.length < 1) || (ssid.length > 32)) {
                $scope.WiFiButtonText = $scope.WiFiButtonSSIDLengthText;
            } else {
                $scope.WiFiButtonText = $scope.WiFiButtonSSIDCharText;
                $scope.WiFiButtonStyle = $scope.WiFiButtonSmallStyle;
            }
        }
    };

    $scope.$watch(function() { return $scope.WiFiSecurityEnabled; }, function(value) {
        $scope.updateWiFiErrorState($scope.WiFiSSID, $scope.WiFiPassphrase);
        return value;
    });

    $scope.updateWiFi = function() {
        var newSettings = {
            "WiFiSSID" :  $scope.WiFiSSID,
            "WiFiSecurityEnabled" : $scope.WiFiSecurityEnabled,
            "WiFiPassphrase" : $scope.WiFiPassphrase,
            "WiFiChannel" : parseInt($scope.WiFiChannel)
        };

        setSettings(angular.toJson(newSettings));
        $scope.Ui.turnOn("modalSuccessWiFi");
    }
}

function isValidSSID(ssid) {
    if (typeof ssid === 'undefined') { return false; }
    return /^[a-zA-Z0-9()_\- ]{1,32}$/g.test(ssid);
}
function isValidWPA(passphrase) {
    if (typeof passphrase === 'undefined') { return false; }
    return /^[\u0020-\u007e]{8,63}$/g.test(passphrase);
}

angular.module('appControllers')
    .directive('hexInput', function() { // directive for ownship hex code validation
        return {
            require: 'ngModel',
            link: function(scope, element, attr, ctrl) {
                ctrl.$parsers.push(function(value) {
                    var valid = /^$|^[0-9A-Fa-f]{6}$/.test(value);
                    ctrl.$setValidity('FAAHex', valid);
                    if (valid) {
                        return value;
                    } else {
                        return "";
                    }
                })
            }
        };
    })
    .directive('watchlistInput', function() { // directive for ICAO space-separated watch list validation
        return {
            require: 'ngModel',
            link: function(scope, element, attr, ctrl) {
                ctrl.$parsers.push(function(value) {
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
                })
            }
        };
    })
    .directive('ssidInput', function() { // directive for WiFi SSID validation
        return {
            require: 'ngModel',
            link: function(scope, element, attr, ctrl) {
                ctrl.$parsers.push(function(value) {
                    scope.updateWiFiErrorState(value, scope.WiFiPassphrase);
                    return value;
                })
            }
        };
    })
    .directive('wpaInput', function() { // directive for WiFi WPA Passphrase validation
        return {
            require: 'ngModel',
            link: function(scope, element, attr, ctrl) {
                ctrl.$parsers.push(function(value) {
                    scope.updateWiFiErrorState(scope.WiFiSSID, value);
                    return value;
                })
            }
        };
    })
    .directive('ipListInput', function() { // directive for validation of list of IP addresses
        return {
            require: 'ngModel',
            link: function(scope, element, attr, ctrl) {
                ctrl.$parsers.push(function(value) {
                    var r = "(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)";
                    var valid = (new RegExp("^(" + r + "( " + r + ")*|)$", "g")).test(value);
                    ctrl.$setValidity('ipList', valid);
                    if (valid) {
                        return value;
                    } else {
                        return "";
                    }
                })
            }
        };
    })
    .directive('gLimitsInput', function() { // directive for validation of list of G Limits
        return {
            require: 'ngModel',
            link: function(scope, element, attr, ctrl) {
                ctrl.$parsers.push(function(value) {
                    var r = "[-+]?[0-9]*\.?[0-9]+";
                    var valid = (new RegExp("^(" + r + "( " + r + ")*|)$", "g")).test(value);
                    ctrl.$setValidity('gLimits', valid);
                    if (valid) {
                        return value;
                    } else {
                        return "";
                    }
                })
            }
        };
    });
