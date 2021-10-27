angular.module('appControllers').controller('MapCtrl', MapCtrl);           // get the main module contollers set
MapCtrl.$inject = ['$rootScope', '$scope', '$state', '$http', '$interval', 'craftService'];  // Inject my dependencies


function MapCtrl($rootScope, $scope, $state, $http, $interval, craftService) {
	let TRAFFIC_MAX_AGE_SECONDS = 15;

	$scope.$parent.helppage = 'plates/radar-help.html';

	$scope.aircraftSymbols = new ol.source.Vector();
	$scope.aircraftTrails = new ol.source.Vector();

	let osm = new ol.layer.Tile({
		title: '[online] OSM',
		type: 'base',
		source: new ol.source.OSM()
	});

	let openaip = new ol.layer.Tile({
		title: '[online] OpenAIP',
		type: 'overlay',
		visible: false,
		source: new ol.source.XYZ({
			url: 'http://{1-2}.tile.maps.openaip.net/geowebcache/service/tms/1.0.0/openaip_basemap@EPSG%3A900913@png/{z}/{x}/{-y}.png'
		})
	});

	// Dynamic MBTiles layers
	$http.get(URL_GET_TILESETS).then(function(response) {
		var tilesets = angular.fromJson(response.data);
		for (let file in tilesets) {
			let meta = tilesets[file];
			let name = (meta.name ? meta.name : file);
			let baselayer = meta.type && meta.type == 'baselayer';
			let format = meta.format ? meta.format : 'png';
			let minzoom = meta.minzoom ? parseInt(meta.minzoom) : 1;
			let maxzoom = meta.maxzoom ? parseInt(meta.maxzoom) : 18;
			
			let ext = [-180, -85, 180, 85];
			if (meta.bounds) {
				ext = meta.bounds.split(',').map(Number)
			}
			ext = ol.proj.transformExtent(ext, 'EPSG:4326', 'EPSG:3857')

			let layer = new ol.layer.Tile({
				title: '[offline] ' + name,
				type: baselayer ? 'base' : 'overlay',
				source: new ol.source.XYZ({
					url: URL_GET_TILE + '/' + file  + '/{z}/{x}/{-y}.' + format,
					maxZoom: maxzoom,
					minZoom: minzoom,
				}),
				extent: ext		
			});
			if (baselayer)
				$scope.map.getLayers().insertAt(0, layer);
			else
				$scope.map.addLayer(layer);
		}
		// Restore layer visibility
		$scope.map.getLayers().forEach((layer) => {
			const title = layer.get('title');
			if (!title) return;
			const key = 'stratux.map.layers.' + title + '.visible';
			const oldState = localStorage.getItem(key);
			if (oldState) {
				layer.setVisible((oldState === 'true'));
			}
		});

		// listener to remember enabled layers
		$scope.map.getLayers().forEach((layer) => {
			layer.on('change:visible', (ev) => {
				const title = ev.target.get('title');
				if (!title) return;
				const visible = ev.target.get('visible');
				const key = 'stratux.map.layers.' + title + '.visible';
				localStorage.setItem(key, visible);
			});
		});
	});

	let aircraftSymbolsLayer = new ol.layer.Vector({
		title: 'Aircraft symbols',
		source: $scope.aircraftSymbols,
		zIndex: 10
	});
	let aircraftTrailsLayer = new ol.layer.Vector({
		title: 'Aircraft trails 5NM',
		source: $scope.aircraftTrails,
		zIndex: 9
	});

	$scope.map = new ol.Map({
		target: 'map_display',
		layers: [
			osm,
			openaip,
			aircraftSymbolsLayer,
			aircraftTrailsLayer
		],
		view: new ol.View({
			center: ol.proj.fromLonLat([10.0, 52.0]),
			zoom: 4,
			enableRotation: false
		})
	});
	$scope.map.addControl(new ol.control.LayerSwitcher());
	
	$scope.aircraft = [];

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

	function createPlaneSvg(aircraft) {
		let html = ``;
		let color = craftService.getTransportColor(aircraft);	
		if (aircraft.TargetType === TARGET_TYPE_AIS) {
		    html = `
				<svg width="11" height="25" xmlns="http://www.w3.org/2000/svg" version="1.1">
					<g>
					<title>Layer 1</title>
					<path stroke="white" fill="${color}" d="m0.22305,11.928l0,12.99329l10.54714,0l0,-12.99329q0,-6.49665 -5.27357,-12.45192q-5.27357,6.49665 -5.27357,12.45192" fill-rule="evenodd" id="svg_1"/>
					</g>		   
				</svg>
			`;
		} else {
			html = `
				<svg height="30" width="30" viewBox="0 0 250 250" version="1.1" xmlns="http://www.w3.org/2000/svg" >
					<path fill="${color}" stroke="white" stroke-width="5" d="M 247.51404,152.40266 139.05781,71.800946 c 0.80268,-12.451845 1.32473,-40.256266 0.85468,-45.417599 -3.94034,-43.266462 -31.23018,-24.6301193 -31.48335,-5.320367 -0.0693,5.281361 -1.01502,32.598388 -1.10471,50.836622 L 0.2842717,154.37562 0,180.19575 l 110.50058,-50.48239 3.99332,80.29163 -32.042567,22.93816 -0.203845,16.89693 42.271772,-11.59566 0.008,0.1395 42.71311,10.91879 -0.50929,-16.88213 -32.45374,-22.39903 2.61132,-80.35205 111.35995,48.50611 -0.73494,-25.77295 z" fill-rule="evenodd"/>
				</svg>
				`;
		}

		return html;
	}

	// Converts from degrees to radians.
	function toRadians(degrees) {
		return degrees * Math.PI / 180;
	};
	
	// Converts from radians to degrees.
	function toDegrees(radians) {
		return radians * 180 / Math.PI;
	}

	function bearing(startLng, startLat, destLng, destLat) {
		startLat = toRadians(startLat);
		startLng = toRadians(startLng);
		destLat = toRadians(destLat);
		destLng = toRadians(destLng);

		y = Math.sin(destLng - startLng) * Math.cos(destLat);
		x = Math.cos(startLat) * Math.sin(destLat) - Math.sin(startLat) * Math.cos(destLat) * Math.cos(destLng - startLng);
		brng = Math.atan2(y, x);
		brng = toDegrees(brng);
		return (brng + 360) % 360;
	}

	function distance(lon1, lat1, lon2, lat2) {
		var R = 6371; // Radius of the earth in km
		var dLat = toRadians(lat2-lat1);  // deg2rad below
		var dLon = toRadians(lon2-lon1); 
		var a = 
			Math.sin(dLat/2) * Math.sin(dLat/2) +
			Math.cos(toRadians(lat1)) * Math.cos(toRadians(lat2)) * 
			Math.sin(dLon/2) * Math.sin(dLon/2)
			; 
		var c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1-a)); 
		var d = R * c; // Distance in km
		return d;
	}

	function computeTrackFromPositions(aircraft) {
		let dist = 0;
		let prev = [aircraft.Lng, aircraft.Lat]

		// Scan backwards until we have at least 500m of position data
		for (var i = aircraft.posHistory.length - 1; i >= 0; i--) {
			dist += distance(prev[0], prev[1], aircraft.posHistory[i][0], aircraft.posHistory[i][1]);
			prev = aircraft.posHistory[i];
			if (dist >= 0.5)
				break;
			
		}
		if (dist != 0 && i >= 0) {
			return bearing(aircraft.posHistory[i][0], aircraft.posHistory[i][1], aircraft.Lng, aircraft.Lat);
		}
		return 0;
	}

	function clipPosHistory(aircraft, maxLenKm) {
		let dist = 0;
		for (var i = aircraft.posHistory.length - 2; i >= 0; i--) {
			let prev = aircraft.posHistory[i+1];
			let curr = aircraft.posHistory[i];
			dist += distance(prev[0], prev[1], curr[0], curr[1]);
			if (dist > maxLenKm)
				break;
		}
		if (i > 0)
			aircraft.posHistory = aircraft.posHistory.slice(i);
	}

	function updateOpacity(aircraft) {
		// For AIS sources we set full opacity for 30 minutes
		let opacity
		if (craftService.isTrafficAged(aircraft)) {
			opacity = 0.0
		} else if (aircraft.TargetType === TARGET_TYPE_AIS) {
			opacity = 1.0;
		} else { // For other sources it's based on seconds
			opacity = 1.0 - (aircraft.Age / TRAFFIC_MAX_AGE_SECONDS);
		}		
		aircraft.marker.getStyle().getImage().setOpacity(opacity);
	}

	function updateVehicleText(aircraft) {
		let text = [];
		if (aircraft.Tail.length > 0)
			text.push(aircraft.Tail);
		if (aircraft.TargetType !== TARGET_TYPE_AIS) {
			text.push(aircraft.Alt + 'ft');
		}
		if (aircraft.Speed_valid && aircraft.Speed>0.1)
			text.push(aircraft.Speed + 'kt')
		aircraft.marker.getStyle().getText().setText(text.join('\n'));
	}

	function updateAircraftTrail(aircraft) {
		if (!aircraft.posHistory || aircraft.posHistory.length < 2)
			return;

		let coords = [];
		for (let c of aircraft.posHistory)
			coords.push(ol.proj.fromLonLat(c));
		coords.push(ol.proj.fromLonLat([aircraft.Lng, aircraft.Lat]));

		let trailFeature = aircraft.trail;
		if (!aircraft.trail) {
			trailFeature = new ol.Feature({
				geometry: new ol.geom.LineString(coords)
			});
			aircraft.trail = trailFeature;
			$scope.aircraftTrails.addFeature(trailFeature);
		} else {
			trailFeature.getGeometry().setCoordinates(coords);
		}
	}

	function isSameAircraft(addr1, addrType1, addr2, addrType2) {
		if (addr1 != addr2)
			return false;
		// Both aircraft have the same address and it is either an ICAO address for both,
		// or a non-icao address for both.
		// 1 = non-icao, everything else = icao
		if ((addrType1 == 1 && addrType2 == 1) || (addrType1 != 1 && addrType2 != 1))
			return true;
		
		return false;
	}

	$scope.onMessage = function(msg) {
		let aircraft = JSON.parse(msg.data);
		if (!aircraft.Position_valid || craftService.isTrafficAged(aircraft)) {
			return;
		}
		aircraft.receivedTs = Date.now();
		let prevColor = undefined;

		// It is only a 'real' update, if the traffic's Age actually changes.
		// If it doesn't, don't restart animation (only interpolated position).
		let updateIndex = -1;
		for (let i in $scope.aircraft) {
			if (isSameAircraft($scope.aircraft[i].Icao_addr, $scope.aircraft[i].Addr_type, aircraft.Icao_addr, aircraft.Addr_type)) {
				let oldAircraft = $scope.aircraft[i];
				prevColor = craftService.getTransportColor(oldAircraft);
				aircraft.marker = oldAircraft.marker;
				aircraft.trail = oldAircraft.trail;
				aircraft.posHistory = oldAircraft.posHistory;

				let prevRecordedPos = aircraft.posHistory[aircraft.posHistory.length - 1];
				 // remember one coord each 100m
				if (distance(prevRecordedPos[0], prevRecordedPos[1], aircraft.Lng, aircraft.Lat) > 0.1) {
					aircraft.posHistory.push([aircraft.Lng, aircraft.Lat]);
				}
				
				// At most 9.25km per aircraft
				aircraft.posHistroy = clipPosHistory(aircraft, 9.25);

				if (!aircraft.Speed_valid) {
					// Compute fake track from last to current position
					aircraft.Track = computeTrackFromPositions(aircraft);
				}
				$scope.aircraft[i] = aircraft;
				updateIndex = i;
			}
		}
		if (updateIndex < 0) {
			$scope.aircraft.push(aircraft);
			aircraft.posHistory = [[aircraft.Lng, aircraft.Lat]];
		}

		let acPosition = [aircraft.Lng, aircraft.Lat];

		if (!aircraft.marker) {
			let offsetY = 40;
			if (aircraft.TargetType === TARGET_TYPE_AIS) {
				offsetY = 20;
			}

			let planeStyle = new ol.style.Style({
				text: new ol.style.Text({
					text: '',
					offsetY: offsetY,
					font: 'bold 1em sans-serif',
					stroke: new ol.style.Stroke({color: 'white', width: 2}),
				})
			});
			let planeFeature = new ol.Feature({
				geometry: new ol.geom.Point(ol.proj.fromLonLat(acPosition))
			});
			planeFeature.setStyle(planeStyle);

			aircraft.marker = planeFeature;
			$scope.aircraftSymbols.addFeature(planeFeature);
		} else {
			aircraft.marker.getGeometry().setCoordinates(ol.proj.fromLonLat(acPosition));
			updateAircraftTrail(aircraft);
		}

		updateVehicleText(aircraft);
		if (!prevColor || prevColor != craftService.getTransportColor(aircraft)) {
			let imageStyle = new ol.style.Icon({
				opacity: 1.0,
				src: 'data:image/svg+xml;utf8,' + createPlaneSvg(aircraft),
				rotation: aircraft.Track,
				anchor: [0.5, 0.5],
				anchorXUnits: 'fraction',
				anchorYUnits: 'fraction'
			});
			aircraft.marker.getStyle().setImage(imageStyle); // to update the color if latest source changed
		}
		updateOpacity(aircraft);
		aircraft.marker.getStyle().getImage().setRotation(toRadians(aircraft.Track));
	}

	$scope.updateAges = function() {
		let now = Date.now();
		for (let ac of $scope.aircraft) {
			// Remember the "Age" value when we received the traffic
			if (!ac.ageReceived)
				ac.ageReceived = ac.Age;
			ac.Age = ac.ageReceived + (now - ac.receivedTs) / 1000.0;
			updateOpacity(ac);
		}
	}

	$scope.removeStaleTraffic = function() {
		let now = Date.now();
		for (let i = 0; i < $scope.aircraft.length; i++) {
			let aircraft = $scope.aircraft[i];
			if (craftService.isTrafficAged(aircraft)) {
				if (aircraft.marker)
					$scope.aircraftSymbols.removeFeature(aircraft.marker);
				if (aircraft.trail)
					$scope.aircraftTrails.removeFeature(aircraft.trail);
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
				pos = ol.proj.fromLonLat([situation.GPSLongitude, situation.GPSLatitude])
				$scope.map.setView(new ol.View({
					center: pos,
					zoom: 10,
					enableRotation: false
				}));


				let gpsLayer = new ol.layer.Vector({
					source: new ol.source.Vector({
						features: [
							new ol.Feature({
								geometry: new ol.geom.Point(pos),
								name: 'Your GPS position'
							})
						]
					}),
					style: new ol.style.Style({
						text: new ol.style.Text({
							text: '\uf041',
							font: 'normal 35px FontAwesome',
							textBaseline: 'bottom'
						})
					})
				});
				$scope.map.addLayer(gpsLayer);
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
