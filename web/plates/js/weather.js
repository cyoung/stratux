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

	function parseShortDatetime(sdt) {
		var d = new Date();
		var s = String(sdt);
		if (s.length < 7)
			return 0;
		d.setUTCDate(parseInt(s.substring(0, 2)));
		d.setUTCHours(parseInt(s.substring(2, 4)));
		d.setUTCMinutes(parseInt(s.substring(4, 6)));
		d.setUTCSeconds(0);
		d.setUTCMilliseconds(0);
		return d.getTime();
	}

	function setDataItem(obj, data_item) {
		if (obj.Type === "TAF.AMD") {
			data_item.type = "TAF";
			data_item.update = true;
		} else {
			data_item.type = obj.Type;
			data_item.update = false;
		}
		data_item.location = obj.Location;
		data_item.age = parseShortDatetime(obj.Time);
		data_item.time = utcTimeString(data_item.age);
		data_item.received = utcTimeString(obj.LocaltimeReceived);
		data_item.data = obj.Data;
	}

	function connect($scope) {
		if (($scope === undefined) || ($scope === null))
			return; // we are getting called once after clicking away from the status page

		if (($scope.socket === undefined) || ($scope.socket === null)) {
			socket = new WebSocket('ws://' + URL_HOST_BASE + '/weather');
			$scope.socket = socket; // store socket in scope for enter/exit usage
		}

		$scope.ConnectState = "Not Receiving";

		socket.onopen = function (msg) {
			// $scope.ConnectStyle = "label-success";
			$scope.ConnectState = "Receiving";
		};

		socket.onclose = function (msg) {
			// $scope.ConnectStyle = "label-danger";
			$scope.ConnectState = "Not Receiving";
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
		// remove stail data = anything more than 30 minutes old
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