angular.module('appControllers').controller('DeveloperCtrl', DeveloperCtrl); // get the main module contollers set
DeveloperCtrl.$inject = ['$rootScope', '$scope', '$state', '$http', '$interval']; // Inject my dependencies

// create our controller function with all necessary logic
function DeveloperCtrl($rootScope, $scope, $state, $http, $interval) {
	$scope.$parent.helppage = 'plates/developer-help.html';
    
	function connect($scope) {
		if (($scope === undefined) || ($scope === null))
			return; // we are getting called once after clicking away from the status page

		if (($scope.socket === undefined) || ($scope.socket === null)) {
			var socket = new WebSocket(URL_STATUS_WS);
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
			console.log('Received status update.');

			var status = JSON.parse(msg.data);
			// Update Status
			$scope.Version = status.Version;
			$scope.Build = status.Build.substr(0, 10);
			$scope.Devices = status.Devices;
			$scope.Ping_connected = status.Ping_connected;
			$scope.Connected_Users = status.Connected_Users;
			$scope.UAT_messages_last_minute = status.UAT_messages_last_minute;
			$scope.UAT_messages_max = status.UAT_messages_max;
			$scope.ES_messages_last_minute = status.ES_messages_last_minute;
			$scope.ES_messages_max = status.ES_messages_max;
			$scope.GPS_satellites_locked = status.GPS_satellites_locked;
			$scope.GPS_satellites_tracked = status.GPS_satellites_tracked;
			$scope.GPS_satellites_seen = status.GPS_satellites_seen;
			$scope.GPS_solution = status.GPS_solution;
			$scope.GPS_position_accuracy = String(status.GPS_solution ? ", " + status.GPS_position_accuracy.toFixed(1) : "");
			$scope.RY835AI_connected = status.RY835AI_connected;
            $scope.UAT_METAR_total = status.UAT_METAR_total;
            $scope.UAT_TAF_total = status.UAT_TAF_total;
            $scope.UAT_NEXRAD_total = status.UAT_NEXRAD_total;
            $scope.UAT_SIGMET_total = status.UAT_SIGMET_total;
            $scope.UAT_PIREP_total = status.UAT_PIREP_total;
            $scope.UAT_NOTAM_total = status.UAT_NOTAM_total;
            $scope.UAT_OTHER_total = status.UAT_OTHER_total;
            $scope.Logfile_Size = humanFileSize(status.Logfile_Size);
            $scope.AHRS_LogFiles_Size = humanFileSize(status.AHRS_LogFiles_Size);
			// Errors array.
			if (status.Errors.length > 0) {
				$scope.visible_errors = true;
				$scope.Errors = status.Errors;
			}

			var uptime = status.Uptime;
			if (uptime !== undefined) {
				var up_d = parseInt((uptime/1000) / 86400),
				    up_h = parseInt((uptime/1000 - 86400*up_d) / 3600),
				    up_m = parseInt((uptime/1000 - 86400*up_d - 3600*up_h) / 60),
				    up_s = parseInt((uptime/1000 - 86400*up_d - 3600*up_h - 60*up_m));
				$scope.Uptime = String(up_d + "/" + ((up_h < 10) ? "0" + up_h : up_h) + ":" + ((up_m < 10) ? "0" + up_m : up_m) + ":" + ((up_s < 10) ? "0" + up_s : up_s));
			} else {
				// $('#Uptime').text('unavailable');
			}
			var boardtemp = status.CPUTemp;
			if (boardtemp !== undefined) {
				/* boardtemp is celcius to tenths */
				$scope.CPUTemp = String(boardtemp.toFixed(1) + 'C / ' + ((boardtemp * 9 / 5) + 32.0).toFixed(1) + 'F');
			} else {
				// $('#CPUTemp').text('unavailable');
			}

			$scope.$apply(); // trigger any needed refreshing of data
		};
	}

	$scope.postRestart = function () {
		$http.post(URL_RESTARTAPP).
		then(function (response) {
			// do nothing
			// $scope.$apply();
		}, function (response) {
			// do nothing
		});
	};

	$scope.webUIRefresh = function() {
		location.reload(true);
	};

	$scope.postDeleteLog = function () {
		$http.post(URL_DELETELOGFILE).
		then(function (response) {
			// do nothing
			// $scope.$apply();
		}, function (response) {
			// do nothing
		});
	};

	$scope.postDownloadLog = function () {
		$http.post(URL_DOWNLOADLOGFILE).
		then(function (response) {
			// do nothing
			// $scope.$apply();
		}, function (response) {
			// do nothing
		});
	};

    $scope.postDeleteAHRSLogs = function () {
        $http.post(URL_DELETEAHRSLOGFILES).
        then(function (response) {
            // do nothing
            // $scope.$apply();
        }, function (response) {
            // do nothing
        });
    };

    $scope.postDownloadAHRSLogs = function () {
        $http.post(URL_DOWNLOADAHRSLOGFILES).
        then(function (response) {
            // do nothing
            // $scope.$apply();
        }, function (response) {
            // do nothing
        });
    };

	$scope.postDownloadDB = function () {
		$http.post(URL_DOWNLOADDB).
		then(function (response) {
			// do nothing
			// $scope.$apply();
		}, function (response) {
			// do nothing
		});
	};

	connect($scope); // connect - opens a socket and listens for messages

}

function humanFileSize(size) {
    if (size === 0) {
        return '0 B'
	} else {
        var i = Math.floor(Math.log(size) / Math.log(1024));
        return ( size / Math.pow(1024, i) ).toFixed(2) * 1 + ' ' + ['B', 'kB', 'MB', 'GB', 'TB'][i];
    }
}
