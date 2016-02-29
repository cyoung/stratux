angular.module('appControllers').controller('StatusCtrl', StatusCtrl); // get the main module contollers set
StatusCtrl.$inject = ['$rootScope', '$scope', '$state', '$http', '$interval']; // Inject my dependencies

// create our controller function with all necessary logic
function StatusCtrl($rootScope, $scope, $state, $http, $interval) {

	$scope.$parent.helppage = 'plates/status-help.html';

	function connect($scope) {
		if (($scope === undefined) || ($scope === null))
			return; // we are getting called once after clicking away from the status page

		if (($scope.socket === undefined) || ($scope.socket === null)) {
			socket = new WebSocket(URL_STATUS_WS);
			$scope.socket = socket; // store socket in scope for enter/exit usage
		}

		$scope.ConnectState = "Disconnected";

		socket.onopen = function (msg) {
			// $scope.ConnectStyle = "label-success";
			$scope.ConnectState = "Connected";
		};

		socket.onclose = function (msg) {
			// $scope.ConnectStyle = "label-danger";
			$scope.ConnectState = "Disconnected";
			$scope.$apply();
			delete $scope.socket;
			setTimeout(function() {connect($scope);}, 1000);
		};

		socket.onerror = function (msg) {
			// $scope.ConnectStyle = "label-danger";
			$scope.ConnectState = "Error";
			$scope.$apply();
		};

		socket.onmessage = function (msg) {
			console.log('Received status update.')

			var status = JSON.parse(msg.data)
				// Update Status
			$scope.Version = status.Version;
			$scope.Build = status.Build.substr(0, 10);
			$scope.Devices = status.Devices;
			$scope.Connected_Users = status.Connected_Users;
			$scope.UAT_messages_last_minute = status.UAT_messages_last_minute;
			// $scope.UAT_products_last_minute = JSON.stringify(status.UAT_products_last_minute);
			$scope.UAT_messages_max = status.UAT_messages_max;
			$scope.ES_messages_last_minute = status.ES_messages_last_minute;
			$scope.ES_messages_max = status.ES_messages_max;
			$scope.GPS_satellites_locked = status.GPS_satellites_locked;
			$scope.GPS_satellites_tracked = status.GPS_satellites_tracked;
			$scope.GPS_satellites_seen = status.GPS_satellites_seen;
			$scope.GPS_solution = status.GPS_solution;
			$scope.RY835AI_connected = status.RY835AI_connected;
			var tempClock = new Date(Date.parse(status.Clock));
			var clockString = tempClock.toUTCString();
			$scope.Clock = clockString;
			var tempLocalClock = new Date;
			$scope.LocalClock = tempLocalClock.toUTCString();
			$scope.SecondsFast = (Math.round(tempClock-tempLocalClock)/1000).toFixed(2);
			
			
			var uptime = status.Uptime;
			if (uptime != undefined) {
				var up_s = parseInt((uptime / 1000) % 60),
					up_m = parseInt((uptime / (1000 * 60)) % 60),
					up_h = parseInt((uptime / (1000 * 60 * 60)) % 24);
				$scope.Uptime = String(((up_h < 10) ? "0" + up_h : up_h) + "h" + ((up_m < 10) ? "0" + up_m : up_m) + "m" + ((up_s < 10) ? "0" + up_s : up_s) + "s");
			} else {
				// $('#Uptime').text('unavailable');
			}
			var boardtemp = status.CPUTemp;
			if (boardtemp != undefined) {
				/* boardtemp is celcius to tenths */
				$scope.CPUTemp = String(boardtemp.toFixed(1) + 'C / ' + ((boardtemp * 9 / 5) + 32.0).toFixed(1) + 'F');
			} else {
				// $('#CPUTemp').text('unavailable');
			}

			$scope.$apply(); // trigger any needed refreshing of data
		};
	}

	function setHardwareVisibility() {
		$scope.visible_uat = true;
		$scope.visible_es = true;
		$scope.visible_gps = true;
		$scope.visible_ahrs = true;

		// Simple GET request example (note: responce is asynchronous)
		$http.get(URL_SETTINGS_GET).
		then(function (response) {
			settings = angular.fromJson(response.data);
			$scope.visible_uat = settings.UAT_Enabled;
			$scope.visible_es = settings.ES_Enabled;
			$scope.visible_gps = settings.GPS_Enabled;
			$scope.visible_ahrs = settings.AHRS_Enabled;
		}, function (response) {
			// nop
		});
	};

	function getTowers() {
		// Simple GET request example (note: responce is asynchronous)
		$http.get(URL_TOWERS_GET).
		then(function (response) {
			var towers = angular.fromJson(response.data);
			var cnt = 0;
			for (var key in towers) {
				if (towers[key].Messages_last_minute > 0) {
					cnt++;
				}
			}
			$scope.UAT_Towers = cnt;
			// $scope.$apply();
		}, function (response) {
			$scope.raw_data = "error getting tower data";
		});
	};

	// periodically get the tower list
	var updateTowers = $interval(function () {
		// refresh tower count once each 5 seconds (aka polling)
		getTowers();
	}, (5 * 1000), 0, false);


	$state.get('home').onEnter = function () {
		// everything gets handled correctly by the controller
	};
	$state.get('home').onExit = function () {
		if (($scope.socket !== undefined) && ($scope.socket !== null)) {
			$scope.socket.close();
			$scope.socket = null;
		}
		$interval.cancel(updateTowers);
	};

	// Status Controller tasks
	setHardwareVisibility();
	connect($scope); // connect - opens a socket and listens for messages
};
