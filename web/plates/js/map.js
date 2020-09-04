angular.module('appControllers').controller('MapCtrl', MapCtrl);           // get the main module contollers set
MapCtrl.$inject = ['$rootScope', '$scope', '$state', '$http', '$interval'];  // Inject my dependencies


function MapCtrl($rootScope, $scope, $state, $http, $interval) {
	let TRAFFIC_MAX_AGE_SECONDS = 15;


	$scope.$parent.helppage = 'plates/radar-help.html';
	$scope.map = L.map('map_display').setView([52.0, 10.0], 4);
	$scope.aircraft = [];
	let osm = L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    	attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
	});//.addTo($scope.map);
	let openaip = L.tileLayer('http://{s}.tile.maps.openaip.net/geowebcache/service/tms/1.0.0/openaip_basemap@EPSG%3A900913@png/{z}/{x}/{y}.png', {
		tms: true,
		subdomains: ['1', '2']
	});//.addTo($scope.map);

	osm.addTo($scope.map);
	openaip.addTo($scope.map);

	let base = {
		'OpenStreeMap': osm
	};
	let overlay = {
		'Open AIP': openaip
	};
	L.control.layers(base, overlay).addTo($scope.map);

	function connect($scope) {
		if (($scope === undefined) || ($scope === null))
			return;  // we are getting called once after clicking away from the status page

		if (($scope.rsocket === undefined) || ($scope.rsocket === null)) {
			rsocket = new WebSocket(URL_RADAR_WS);
			$scope.rsocket = rsocket;                  // store socket in scope for enter/exit usage
		}
		
		$scope.ConnectState = 'Disconnected';

		rsocket.onopen = function(msg) {
			$scope.ConnectState = 'Connected';
			$scope.$apply();
		};

		rsocket.onclose = function(msg) {
			$scope.ConnectState = 'Disconnected';
			$scope.$apply();
			if ($scope.rsocket !== null ) {
				setTimeout(connect, 1000);   // do not set timeout after exit
			}
		};

		rsocket.onerror = function(msg) {
			// $scope.ConnectStyle = "label-danger";
			$scope.ConnectState = 'Problem';
			$scope.$apply();
		};

		rsocket.onmessage = function(msg) {
			$scope.onMessage(msg);
		};
	}

	$scope.onMessage = function(msg) {
		let aircraft = JSON.parse(msg.data);
		if (!aircraft.Position_valid || aircraft.Age > TRAFFIC_MAX_AGE_SECONDS)
			return;

		let updateIndex = -1;
		for (let i in $scope.aircraft) {
			if ($scope.aircraft[i].Icao_addr == aircraft.Icao_addr) {
				aircraft.marker = $scope.aircraft[i].marker;
				$scope.aircraft[i] = aircraft;
				updateIndex = i;
			}
		}
		if (updateIndex < 0) {
			$scope.aircraft.push(aircraft);
			updateIndex = $scope.aircraft.length - 1;
		}

		let planeIcon = L.divIcon({
			className: '',
			html: '<img src="img/plane.svg" style="width:40px; height:40px; transform: rotate(' + 
					aircraft.Track + 'deg)" /><span class="plane-map-label">' + aircraft.Tail + '<br/>' + aircraft.Alt + 'ft<br/>' + aircraft.Speed + 'kt</span>',
			iconAnchor: [20, 20]
		});

		if (!aircraft.marker) {

			let marker = L.marker([aircraft.Lat, aircraft.Lng], {
				icon: planeIcon,
				title: aircraft.Tail
			});
			$scope.aircraft[updateIndex].marker = marker;
			marker.addTo($scope.map);
		} else {
			aircraft.marker.setIcon(planeIcon); // to update rotation
			aircraft.marker.setLatLng([aircraft.Lat, aircraft.Lng]);
		}
	
	}

	$scope.removeStaleTraffic = function() {
		for (let i = 0; i < $scope.aircraft.length; i++) {
			let aircraft = $scope.aircraft[i];
			if (aircraft.Age > TRAFFIC_MAX_AGE_SECONDS) {
				$scope.map.removeLayer(aircraft.marker);
				$scope.aircraft.splice(i, 1);
				i--;
			}
		}
	}

	$scope.update = function() {
		$scope.removeStaleTraffic();
	}

	function getLocationForInitialPosition() {
		$http.get(URL_GET_SITUATION).then(function(response) {
			situation = angular.fromJson(response.data);
			$scope.map.setView([situation.GPSLatitude, situation.GPSLongitude], 10);

		});
	};


	connect($scope);
	getLocationForInitialPosition();

	$interval($scope.update, 1000);

}
