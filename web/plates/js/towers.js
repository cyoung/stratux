angular.module('appControllers').controller('TowersCtrl', TowersCtrl); // get the main module contollers set
TowersCtrl.$inject = ['$rootScope', '$scope', '$state', '$http', '$interval']; // Inject my dependencies

// create our controller function with all necessary logic
function TowersCtrl($rootScope, $scope, $state, $http, $interval) {

	$scope.$parent.helppage = 'plates/towers-help.html';
	$scope.data_list = [];

	function dmsString(val) {
		return [0 | val,
				'° ',
				0 | (val < 0 ? val = -val : val) % 1 * 60,
				"' ",
				0 | val * 60 % 1 * 60,
				'"'].join('');
	}

	function setTower(obj, new_tower) {
		new_tower.lat = dmsString(obj.Lat);
		new_tower.lon = dmsString(obj.Lng);
		new_tower.power = obj.Signal_strength_now.toFixed(2);
		new_tower.power_last_min = obj.Signal_strength_last_minute.toFixed(2);
		new_tower.power_max = obj.Signal_strength_max.toFixed(2);
		// Messages_last_minute        uint64
		new_tower.messages = obj.Messages_last_minute;
	}

	function loadTowers(data) {
		if (($scope === undefined) || ($scope === null))
			return; // we are getting called once after clicking away from the status page

		var towers = data; // it seems the json was already converted to an object list by teh http request
		$scope.raw_data = angular.toJson(data, true);

		$scope.data_list.length = 0; // clear array
		// we need to use an array so AngularJS can perform sorting; it also means we need to loop to find a tower in the towers set
		for (var key in towers) {
			if (towers[key].Messages_last_minute > 0) {
				var new_tower = {};
				setTower(towers[key], new_tower);
				$scope.data_list.push(new_tower); // add to start of array
			}
		}
		// $scope.$apply();
	}

	function getTowers() {
		// Simple GET request example (note: responce is asynchronous)
		$http.get(URL_TOWERS_GET).
		then(function (response) {
			loadTowers(response.data);
		}, function (response) {
			$scope.raw_data = "error getting tower data";
		});
	};

	var updateTowers = $interval(function () {
		// refresh tower list once every 5 seconds (aka polling)
		getTowers();
	}, (2 * 1000), 0, false);

	$state.get('towers').onEnter = function () {
		// everything gets handled correctly by the controller
	};

	$state.get('towers').onExit = function () {
		// stop any interval functions
		$interval.cancel(updateTowers);
	};
};