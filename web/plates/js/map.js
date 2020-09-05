angular.module('appControllers').controller('MapCtrl', MapCtrl);           // get the main module contollers set
MapCtrl.$inject = ['$rootScope', '$scope', '$state', '$http', '$interval'];  // Inject my dependencies


function MapCtrl($rootScope, $scope, $state, $http, $interval) {
	let TRAFFIC_MAX_AGE_SECONDS = 15;


	$scope.$parent.helppage = 'plates/radar-help.html';
	$scope.map = L.map('map_display').setView([52.0, 10.0], 4);
	$scope.aircraft = [];
	let osm = L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    	attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
	});
	let openaip = L.tileLayer('http://{s}.tile.maps.openaip.net/geowebcache/service/tms/1.0.0/openaip_basemap@EPSG%3A900913@png/{z}/{x}/{y}.png', {
		tms: true,
		subdomains: ['1', '2']
	});

	osm.addTo($scope.map);
	openaip.addTo($scope.map);

	let base = {
		'OpenStreeMap [online]': osm
	};
	let overlay = {
		'Open AIP [online]': openaip
	};
	L.control.layers(base, overlay).addTo($scope.map);

	function connect($scope) {
		if (($scope === undefined) || ($scope === null))
			return;  // we are getting called once after clicking away from the status page

		if (($scope.socket === undefined) || ($scope.socket === null)) {
			socket = new WebSocket(URL_TRAFFIC_WS);
			$scope.socket = socket;                  // store socket in scope for enter/exit usage
		}
		
		$scope.ConnectState = 'Disconnected';

		socket.onopen = function(msg) {
			$scope.ConnectState = 'Connected';
			$scope.$apply();
		};

		socket.onclose = function(msg) {
			$scope.ConnectState = 'Disconnected';
			$scope.$apply();
			if ($scope.socket !== null ) {
				setTimeout(connect, 1000);   // do not set timeout after exit
			}
		};

		socket.onerror = function(msg) {
			// $scope.ConnectStyle = "label-danger";
			$scope.ConnectState = 'Problem';
			$scope.$apply();
		};

		socket.onmessage = function(msg) {
			$scope.onMessage(msg);
		};
	}

	$scope.createPlaneSvg = function(aircraft) {
		let html = `
			<svg height="30" width="30" viewBox="0 0 250 250" transform="rotate({Track})" class="plane-map">
				<path id="acpath" d="M 247.51404,152.40266 139.05781,71.800946 c 0.80268,-12.451845 1.32473,-40.256266 0.85468,-45.417599 -3.94034,-43.266462 -31.23018,-24.6301193 -31.48335,-5.320367 -0.0693,5.281361 -1.01502,32.598388 -1.10471,50.836622 L 0.2842717,154.37562 0,180.19575 l 110.50058,-50.48239 3.99332,80.29163 -32.042567,22.93816 -0.203845,16.89693 42.271772,-11.59566 0.008,0.1395 42.71311,10.91879 -0.50929,-16.88213 -32.45374,-22.39903 2.61132,-80.35205 111.35995,48.50611 -0.73494,-25.77295 z" fill-rule="evenodd"/>
			</svg>
			`;

		return L.Util.template(html, aircraft);

	}

	$scope.onMessage = function(msg) {
		let aircraft = JSON.parse(msg.data);
		if (!aircraft.Position_valid || aircraft.Age > TRAFFIC_MAX_AGE_SECONDS)
			return;

		aircraft.receivedTs = Date.now();

		// It is only a 'real' update, if the traffic's Age actually changes.
		// If it doesn't, don't restart animation (only interpolated position).
		let isActualUpdate = true;
		let updateIndex = -1;
		for (let i in $scope.aircraft) {
			if ($scope.aircraft[i].Icao_addr == aircraft.Icao_addr) {
				aircraft.marker = $scope.aircraft[i].marker;
				isActualUpdate = (aircraft.Age < $scope.aircraft[i].Age);
				$scope.aircraft[i] = aircraft;
				updateIndex = i;
			}
		}
		if (updateIndex < 0) {
			$scope.aircraft.push(aircraft);
		}

		let acPosition = [aircraft.Lat, aircraft.Lng];

		if (!aircraft.marker) {
			let planeIcon = L.divIcon({
				className: '',
				html: $scope.createPlaneSvg(aircraft) + L.Util.template(`<span class="plane-map-label">{Tail}<br/>{Alt}ft<br/>{Speed}kt</span>`, aircraft),
				iconAnchor: [15, 15]
			});
			let marker = L.marker(acPosition, {
				icon: planeIcon,
				title: aircraft.Tail
			});

			aircraft.marker = marker;
			marker.addTo($scope.map);
		} else {
			aircraft.marker.setLatLng(acPosition);

			let svgElem = aircraft.marker._icon.getElementsByTagName('svg')[0]
			svgElem.transform.baseVal.getItem(0).setRotate(aircraft.Track, 0, 0);
			// Restart animation if age changed..
			if (isActualUpdate) {
				svgElem.style.animation = 'none';
				setTimeout(function() {
					svgElem.style.animation = null;
				}, 100);
			}
		}
	
	}

	$scope.updateAges = function() {
		let now = Date.now();
		for (let ac of $scope.aircraft) {
			// Remember the "Age" value when we received the traffic
			if (!ac.ageReceived)
				ac.ageReceived = ac.Age;
			ac.Age = ac.ageReceived + (now - ac.receivedTs) / 1000.0;
		}
	}

	$scope.removeStaleTraffic = function() {
		let now = Date.now();
		for (let i = 0; i < $scope.aircraft.length; i++) {
			let aircraft = $scope.aircraft[i];
			if (aircraft.Age > TRAFFIC_MAX_AGE_SECONDS) {
				if (aircraft.marker)
					$scope.map.removeLayer(aircraft.marker);
				$scope.aircraft.splice(i, 1);
				i--;
			}
		}
	}

	$scope.update = function() {
		$scope.updateAges();
		$scope.removeStaleTraffic();
	}


	function getLocationForInitialPosition() {
		$http.get(URL_GET_SITUATION).then(function(response) {
			situation = angular.fromJson(response.data);
			if (situation.GPSFixQuality > 0) {
				$scope.map.setView([situation.GPSLatitude, situation.GPSLongitude], 10);
			}

		});
	};

	$state.get('map').onExit = function () {
		// disconnect from the socket
		if (($scope.socket !== undefined) && ($scope.socket !== null)) {
			$scope.socket.close();
			$scope.socket = null;
		}
		// stop stale traffic cleanup
		$interval.cancel($scope.update);
	}


	connect($scope);
	getLocationForInitialPosition();

	$interval($scope.update, 1000);

}
