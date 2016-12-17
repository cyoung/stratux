angular.module('appControllers').controller('SettingsCtrl', SettingsCtrl); // get the main module contollers set
SettingsCtrl.$inject = ['$rootScope', '$scope', '$state', '$location', '$window', '$http']; // Inject my dependencies

// create our controller function with all necessary logic
function SettingsCtrl($rootScope, $scope, $state, $location, $window, $http) {

	$scope.$parent.helppage = 'plates/settings-help.html';

	var toggles = ['UAT_Enabled', 'ES_Enabled', 'Ping_Enabled', 'GPS_Enabled', 'DisplayTrafficSource', 'DEBUG', 'ReplayLog']; 
	var settings = {};
	for (i = 0; i < toggles.length; i++) {
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
		$scope.DisplayTrafficSource = settings.DisplayTrafficSource;
		$scope.DEBUG = settings.DEBUG;
		$scope.ReplayLog = settings.ReplayLog;
		$scope.PPM = settings.PPM;
		$scope.WatchList = settings.WatchList;
		$scope.OwnshipModeS = settings.OwnshipModeS;
		$scope.DeveloperMode = settings.DeveloperMode;
	}

	function getSettings() {
		// Simple GET request example (note: responce is asynchronous)
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
	};

	function setSettings(msg) {
		// Simple POST request example (note: responce is asynchronous)
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

	$scope.$watchGroup(toggles, function (newValues, oldValues, scope) {
		var newsettings = {}
		var dirty = false;
		for (i = 0; i < newValues.length; i++) {
			if ((newValues[i] !== undefined) && (settings[toggles[i]] !== undefined)) {
				if (newValues[i] !== settings[toggles[i]]) {
					settings[toggles[i]] = newValues[i];
					newsettings[toggles[i]] = newValues[i];
					dirty = true;
				};
			}
		}
		if (dirty) {
			// console.log(angular.toJson(newsettings));
			setSettings(angular.toJson(newsettings));
		}
	});

	$scope.updateppm = function () {
		settings["PPM"] = 0
		if (($scope.PPM !== undefined) && ($scope.PPM !== null) && ($scope.PPM !== settings["PPM"])) {
			settings["PPM"] = parseInt($scope.PPM);
			newsettings = {
				"PPM": settings["PPM"]
			};
			// console.log(angular.toJson(newsettings));
			setSettings(angular.toJson(newsettings));
		}
	};

	$scope.updateBaud = function () {
		settings["Baud"] = 0
		if (($scope.Baud !== undefined) && ($scope.Baud !== null) && ($scope.Baud !== settings["Baud"])) {
			settings["Baud"] = parseInt($scope.Baud);
			newsettings = {
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
			newsettings = {
				"WatchList": settings["WatchList"]
			};
			// console.log(angular.toJson(newsettings));
			setSettings(angular.toJson(newsettings));
		}
	};
	$scope.updatemodes = function () {
		if ($scope.OwnshipModeS !== settings["OwnshipModeS"]) {
			settings["OwnshipModeS"] = $scope.OwnshipModeS.toUpperCase();
			newsettings = {
				"OwnshipModeS": $scope.OwnshipModeS.toUpperCase()
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
	}
	$scope.resetUploadFile = function () {
		$scope.update_files = '';
		$scope.$apply();
	}
	$scope.uploadFile = function () {
		var fd = new FormData();
		//Take the first selected file
		var file = $scope.update_files[0];
		// check for empty string
		if (file === undefined || file === null) {
			alert ("update file not selected")
			return;
		}
		var filename = file.name;
		// check for expected file naming convention
		var re = /^update.*\.sh$/;
		if (!re.exec(filename)) {
			alert ("file does not appear to be an update")
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
};
