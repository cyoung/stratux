angular.module('appControllers').controller('GPSCtrl', GPSCtrl); // get the main module contollers set
GPSCtrl.$inject = ['$rootScope', '$scope', '$state', '$http', '$interval']; // Inject my dependencies

// create our controller function with all necessary logic
function GPSCtrl($rootScope, $scope, $state, $http, $interval) {
	$scope.$parent.helppage = 'plates/gps-help.html';

	var status = {};
	var display_area_size = -1;

	function sizeMap() {
		var width = 0;
		var el = document.getElementById("map_display").parentElement;
		width = el.offsetWidth; // was  (- (2 * el.offsetLeft))
		if (width !== display_area_size) {
			display_area_size = width;
			$scope.map_width = width;
			$scope.map_height = width *0.5;
		}
		return width;
	}

	function setGeoReferenceMap(la, lo) {
		// Mercator projection
		// var map = "img/world.png";
		var map_width = 2530;
		var map_height = 1603;
		var map_zero_x = 1192;
		var map_zero_y = 1124;
		var font_size = 18; // size of font used for marker

		sizeMap();
		var div_width = $scope.map_width;
		var div_height = $scope.map_height;

		
		// longitude: just scale and shift
		var x = (map_width * (180 + lo) / 360) - (map_width/2 - map_zero_x); // longitude_shift;

		// latitude: using the Mercator projection
		la_rad = la * Math.PI / 180; // convert from degrees to radians
		merc_n = Math.log(Math.tan((la_rad / 2) + (Math.PI / 4))); // do the Mercator projection (w/ equator of 2pi units)
		var y = (map_height / 2) - (map_width * merc_n / (2 * Math.PI)) - (map_height/2 - map_zero_y); // fit it to our map

		// dot = '<div style="position:absolute; width:' + dot_size + 'px; height:' + dot_size + 'px; top:' + y + 'px; left:' + x + 'px; background:#ff7f00;"></div>';
		// <img src="map-world-medium.png" style="position:absolute;top:0px;left:0px">
		$scope.map_pos_x = map_width - Math.round(x - (div_width / 2));
		$scope.map_pos_y = map_height - Math.round(y - (div_height / 2));

		$scope.map_mark_x = Math.round((div_width - (font_size * 0.85)) / 2);
		$scope.map_mark_y = Math.round((div_height - font_size) / 2);
	};


	function loadStatus(data) {
		status = angular.fromJson(data);
		// consider using angular.extend()
		$scope.raw_data = angular.toJson(data, true); // makes it pretty

		/*	not currently used
		$scope.gps_satellites = status.Satellites;
		*/
		$scope.gps_accuracy = status.Accuracy.toFixed(1);
                $scope.gps_vert_accuracy = (status.AccuracyVert*3.2808).toFixed(1); // accuracy is in meters, need to display in ft


		// NACp should be an integer value in the range of 0 .. 11
		// var accuracies = ["≥ 10 NM", "< 10 NM", "< 4 NM", "< 2 NM", "< 1 NM", "< 0.5 NM", "< 0.3 NM", "< 0.1 NM", "< 100 m", "< 30 m", "< 10 m", "< 3 m"];
		// $scope.gps_accuracy = accuracies[status.NACp];
		// "LastFixLocalTime":"2015-10-11T16:47:03.523085162Z"

		$scope.gps_lat = status.Lat.toFixed(5); // result is string
		$scope.gps_lon = status.Lng.toFixed(5); // result is string
		$scope.gps_alt = Math.round(status.Alt);
		$scope.gps_track = status.TrueCourse;
		$scope.gps_speed = status.GroundSpeed;
                $scope.gps_vert_speed = status.GPSVertVel.toFixed(1);

		// "LastGroundTrackTime":"0001-01-01T00:00:00Z"

		/* not currently used 
		$scope.ahrs_temp = status.Temp;
		*/
		$scope.ahrs_alt = Math.round(status.Pressure_alt);

		$scope.ahrs_heading = Math.round(status.Gyro_heading*100)/100;
		// pitch and roll are in degrees
		$scope.ahrs_pitch = Math.round(status.Pitch*100)/100;
		$scope.ahrs_roll = Math.round(status.Roll*100)/100;
		// "LastAttitudeTime":"2015-10-11T16:47:03.534615187Z"

		setGeoReferenceMap(status.Lat, status.Lng);

		// $scope.$apply();
	};

	function getStatus() {
		// Simple GET request example (note: responce is asynchronous)
		$http.get(URL_GPS_GET).
		then(function (response) {
			loadStatus(response.data);
			ahrs.animate(1, $scope.ahrs_pitch, $scope.ahrs_roll, $scope.ahrs_heading);
			// $scope.$apply();
		}, function (response) {
			$scope.raw_data = "error getting gps / ahrs status";
		});
	};

	var updateStatus = $interval(function () {
		// refresh GPS/AHRS status once each half second (aka polling)
		getStatus();
	}, (1 * 500), 0, false);

	$state.get('gps').onEnter = function () {
		// everything gets handled correctly by the controller
	};

	$state.get('gps').onExit = function () {
		// stop polling for gps/ahrs status
		$interval.cancel(updateStatus);
	};


	// GPS/AHRS Controller tasks go here
	var ahrs = new ahrsRenderer("ahrs_display");
	ahrs.init();
	ahrs.orientation(0, 0, 0);

};
