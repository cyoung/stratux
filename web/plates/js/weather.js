angular.module('appControllers').controller('WeatherCtrl', WeatherCtrl); // get the main module contollers set
WeatherCtrl.$inject = ['$rootScope', '$scope', '$state', '$http', '$interval']; // Inject my dependencies

// create our controller function with all necessary logic
function WeatherCtrl($rootScope, $scope, $state, $http, $interval) {

	var CONF_WATCHLIST = "KBOS KATL KORD KLAX"; // we default to 4 major airports
	var MAX_DATALIST = 10;

	$scope.$parent.helppage = 'plates/weather-help.html';
	$scope.data_list = [];
	$scope.watch_list = [];
	$scope.data_count = 0;
	$scope.watch_count = 0;

	function updateWatchList() {
		$scope.watching = CONF_WATCHLIST;
		// Simple GET request example (note: responce is asynchronous)
		$http.get(URL_SETTINGS_GET).
		then(function (response) {
			settings = angular.fromJson(response.data);
			$scope.watching = settings.WatchList.toUpperCase();
		}, function (response) {
			// nop
		});
	};

	function inList(word, sentence) {
		// since the watch list is just one long string, we cheat and see if the word in anywhere in the 'sentence'
		if ((sentence) && (word)) {
			return sentence.includes(word);
		}
		return false;
	}


	function parseFlightCondition(msg, body) {
		if ((msg !== "METAR") && (msg !== "SPECI"))
			return "";

		// check the visibility: a value preceeding 'SM' which is either a fraction or a whole number
		// we don't care what value of fraction since anything below 1SM is LIFR

		// BTW: now I know why no one wants to parse METARs - ther can be spaces in the numbers ARGH
		// test for special case of 'X X/X'
		var exp = new RegExp("([0-9]) ([0-9])/([0-9])SM");
		var match = exp.exec(body);
		if ((match !== null) && (match.length === 4)) {
			visability = parseInt(match[1]) + (parseInt(match[2]) / parseInt(match[3]));
		} else {
			exp = new RegExp("([0-9/]{1,5}?)SM");
			match = exp.exec(body);
			if (match === null)
				return "";
			// the only way we have 3 or more characters is if the '/' is present which means we need to do extra checking
			if (match[1].length === 3)
				return "LIFR";
			// do we have a usable visability distance
			var visability = parseInt(match[1]);
			if (visability === 0)
				return "";
		}

		// ceiling is at either the BKN or OVC layer
		exp = new RegExp("BKN([0-9]{3})");
		match = exp.exec(body);
		if (match === null) {
			exp = new RegExp("OVC([0-9]{3})");
			match = exp.exec(body);
		}
		var ceiling = 999;
		if (match !== null)
			ceiling = parseInt(match[1]);

		if ((visability > 5) && (ceiling > 30))
			return "VFR";
		if ((visability >= 3) && (ceiling >= 10))
			return "MVFR";
		if ((visability >= 1) && (ceiling >= 5))
			return "IFR";
		return "LIFR";
	}


	function deltaTimeString(epoc) {
		var time = "";
		var val;
		var d = new Date(epoc);
		val = d.getUTCDate() - 1; // we got here by subtrracting two dates so we have a delta, not a day of month
		if (val > 0)
			time += (val < 10 ? "0" + val : "" + val) + "d ";
		val = d.getUTCHours();
		if (val > 0) {
			time += (val < 10 ? "0" + val : "" + val) + "h ";
		} else {
			if (time.length > 0)
				time += "00h ";
		}
		val = d.getUTCMinutes();
		time += (val < 10 ? "0" + val : "" + val) + "m ";
		// ADS-B weather is only accurate to minutes
		// val = d.getUTCSeconds();
		// time += (val < 10 ? "0" + val : "" + val) + "s";

		return time;
	}

	function parseShortDatetime(sdt) {
		var d = new Date();
		var s = String(sdt);
		if (s.length < 7)
			return 0;
		d.setUTCDate(parseInt(s.substring(0, 2)));
		d.setUTCHours(parseInt(s.substring(2, 4)));
		if (s.length > 7) { // TAF datetime range
			d.setUTCMinutes(0);
		} else {
			d.setUTCMinutes(parseInt(s.substring(4, 6)));
		}
		d.setUTCSeconds(0);
		d.setUTCMilliseconds(0);
		return d;
	}

	function setDataItem(obj, data_item) {
		if (obj.Type === "TAF.AMD") {
			data_item.type = "TAF";
			data_item.update = true;
		} else {
			data_item.type = obj.Type;
			data_item.update = false;
		}

		data_item.flight_condition = parseFlightCondition(obj.Type, obj.Data);
		data_item.location = obj.Location;
		s = obj.Time;
		// data_item.time = s.substring(0, 2) + '-' + s.substring(2, 4) + ':' + s.substring(4, 6) + 'Z';
		// we may not get an accurate base time on the stratux device so we use the device time as our base
		// var dNow = new Date(obj.LocaltimeReceived);
		var dNow = new Date();
		var dThen = parseShortDatetime(obj.Time);
		data_item.age = dThen.getTime();
		var diff_ms = Math.abs(dThen - dNow);

		// If time is more than two days away, don't attempt to display data age.
		if (diff_ms > (1000*60*60*24*2)) {
			data_item.time = "?";
		} else if (dThen > dNow) {
			data_item.time = deltaTimeString(dThen - dNow) + " from now";
		} else {
			data_item.time = deltaTimeString(dNow - dThen) + " old";
		}

		// data_item.received = utcTimeString(obj.LocaltimeReceived);
		data_item.data = obj.Data;
	}

	function connect($scope) {
		if (($scope === undefined) || ($scope === null))
			return; // we are getting called once after clicking away from the status page

		if (($scope.socket === undefined) || ($scope.socket === null)) {
			socket = new WebSocket(URL_WEATHER_WS);
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
			setTimeout(connect, 1000);
		};

		socket.onerror = function (msg) {
			// $scope.ConnectStyle = "label-danger";
			$scope.ConnectState = "Problem";
			$scope.$apply();
		};

		socket.onmessage = function (msg) {
			console.log('Received data_list update.');

			$scope.raw_data = angular.toJson(msg.data, true);
			var message = JSON.parse(msg.data);
			// we need to use an array so AngularJS can perform sorting; it also means we need to loop to find an aircraft in the data_list set
			var found = false;
			if (inList(message.Location, $scope.watching)) {
				for (var i = 0, len = $scope.watch_list.length; i < len; i++) {
					if (($scope.watch_list[i].type === message.Type) && ($scope.watch_list[i].location === message.Location)) {
						setDataItem(message, $scope.watch_list[i]);
						found = true;
						break;
					}
				}
				if (!found) {
					var new_data_item = {};
					setDataItem(message, new_data_item);
					$scope.watch_list.unshift(new_data_item); // add to start of array
				}
			}
			// add to scrolling data_list
			{
				var new_data_item = {};
				setDataItem(message, new_data_item);
				$scope.data_list.unshift(new_data_item); // add to start of array
				if ($scope.data_list.length > MAX_DATALIST)
					$scope.data_list.pop(); // remove last from array
			}
			$scope.data_count = $scope.data_list.length;
			$scope.watch_count = $scope.watch_list.length;
			$scope.$apply();
		};
	}

	// perform cleanup every 5 minutes
	var clearStaleMessages = $interval(function () {
		// remove stale data = anything more than 30 minutes old
		var dirty = false;
		var cutoff = Date.now() - (30 * 60 * 1000);

		for (var i = len = $scope.watch_list.length; i > 0; i--) {
			if ($scope.watch_list[i - 1].age < cutoff) {
				$scope.watch_list.splice(i - 1, 1);
				dirty = true;
			}
		}
		if (dirty) {
			$scope.raw_data = "";
			$scope.$apply();
		}
	}, (5 * 60 * 1000), 0, false);


	$state.get('weather').onEnter = function () {
		// everything gets handled correctly by the controller
		updateWatchList();
	};

	$state.get('weather').onExit = function () {
		// disconnect from the socket
		if (($scope.socket !== undefined) && ($scope.socket !== null)) {
			$scope.socket.close();
			$scope.socket = null;
		}
		// stop stale message cleanup
		$interval.cancel(clearStaleMessages);
	};



	// Weather Controller tasks
	updateWatchList();
	connect($scope); // connect - opens a socket and listens for messages
};