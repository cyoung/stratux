angular.module('appControllers').controller('TrafficCtrl', TrafficCtrl); // get the main module contollers set
TrafficCtrl.$inject = ['$rootScope', '$scope', '$state', '$http', '$interval']; // Inject my dependencies

// create our controller function with all necessary logic
function TrafficCtrl($rootScope, $scope, $state, $http, $interval) {

	$scope.$parent.helppage = 'plates/traffic-help.html';
	$scope.traffic = [];

	function utcTimeString(epoc) {
		var time = "";
		var val;
		var d = new Date(epoc);
		val = d.getUTCHours();
		time += (val < 10 ? "0" + val : "" + val);
		val = d.getUTCMinutes();
		time += ":" + (val < 10 ? "0" + val : "" + val);
		val = d.getUTCSeconds();
		time += ":" + (val < 10 ? "0" + val : "" + val);
		time += "Z";
		return time;
	}

	function dmsString(val) {
		return [0 | val,
				'd ',
				0 | (val < 0 ? val = -val : val) % 1 * 60,
				"' ",
				0 | val * 60 % 1 * 60,
				'"'].join('');
	}

	function setAircraft(obj, new_traffic) {
		new_traffic.icao_int = obj.Icao_addr;
		new_traffic.icao = obj.Icao_addr.toString(16).toUpperCase();
		new_traffic.tail = obj.Tail;
		new_traffic.lat = dmsString(obj.Lat);
		new_traffic.lon = dmsString(obj.Lng);
		var n = Math.round(obj.Alt / 100) * 100;
		new_traffic.alt = n.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
		new_traffic.heading = Math.round(obj.Track / 10) * 10;
		new_traffic.speed = Math.round(obj.Speed / 10) * 10;
		new_traffic.vspeed = Math.round(obj.Vvel / 100) * 100
		new_traffic.age = Date.parse(obj.Last_seen);
		new_traffic.time = utcTimeString(new_traffic.age);
		// return new_aircraft;
	}

	function connect($scope) {
		if (($scope === undefined) || ($scope === null))
			return; // we are getting called once after clicking away from the status page

		if (($scope.socket === undefined) || ($scope.socket === null)) {
			socket = new WebSocket('ws://' + URL_HOST_BASE + '/traffic');
			$scope.socket = socket; // store socket in scope for enter/exit usage
		}

		$scope.ConnectState = "Not Receiving";

		socket.onopen = function (msg) {
			$scope.ConnectStyle = "label-success";
			$scope.ConnectState = "Receiving";
		};

		socket.onclose = function (msg) {
			$scope.ConnectStyle = "label-danger";
			$scope.ConnectState = "Not Receiving";
			setTimeout(connect, 1000);
		};

		socket.onerror = function (msg) {
			$scope.ConnectStyle = "label-danger";
			$scope.ConnectState = "Problem";
		};

		socket.onmessage = function (msg) {
			console.log('Received traffic update.')

			var aircraft = JSON.parse(msg.data);
			if (aircraft.Position_valid) {
				$scope.rawTraffic = msg.data;
				// we need to use an array so AngularJS can perform sorting; it also means we need to loop to find an aircraft in the traffic set
				var found = false;
				for (var i = 0, len = $scope.traffic.length; i < len; i++) {
					if ($scope.traffic[i].icao_int === aircraft.Icao_addr) {
						setAircraft(aircraft, $scope.traffic[i]);
						found = true;
						break;
					}
				}
				if (!found) {
					var new_traffic = {};
					setAircraft(aircraft, new_traffic);
					$scope.traffic.unshift(new_traffic); // add to start of array
				}
				$scope.$apply();
			}
		};
	}

	// perform cleanup every 60 seconds
	var clearStaleTraffic = $interval(function () {
		// remove stail aircraft = anything more than 180 seconds without and update
		var dirty = false;
		var cutoff = Date.now() - (180 * 1000);

		for (var i = len = $scope.traffic.length; i > 0; i--) {
			if ($scope.traffic[i - 1].age < cutoff) {
				$scope.traffic.splice(i - 1, 1);
				dirty = true;
			}
		}
		if (dirty) {
			$scope.$apply();
		}
	}, (1000 * 60), 0, false);


	$state.get('traffic').onEnter = function () {
		// everything gets handled correctly by the controller
	};

	$state.get('traffic').onExit = function () {
		// disconnect from the socket
		if (($scope.socket !== undefined) && ($scope.socket !== null)) {
			$scope.socket.close();
			$scope.socket = null;
		}
		// stop stale traffic cleanup
		$interval.cancel(clearStaleTraffic);
	};

	// Traffic Controller tasks
	connect($scope); // connect - opens a socket and listens for messages
};