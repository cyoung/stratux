angular.module('appControllers').controller('RadarCtrl', RadarCtrl); // get the main module contollers set
RadarCtrl.$inject = ['$rootScope', '$scope', '$state', '$http', '$interval']; // Inject my dependencies

var Lat;
var Long;
var GPSCourse = 0;
var BaroAltitude;   // Barometric Altitude if availabe, else set to GPS Altitude, invalid is -100.000
var DisplayRadius = 10;    // Radius in NM, below this radius targets are displayed

var MaxAlarms = 5;         // number of times an alarm sound is played, if airplane enters AlarmRadius
var MaxSpeechAlarms = 1;         // number of times the aircraft is announced, MaxSpeedAlarms needs to be less than MaxAlarms

var minimalCircle = 25;    //minimal circle in pixel around center ist distance is very close
var radar;    // global RadarRenderer
var posangle = Math.PI;   //global var for angle position of text 

var zoom = [2,5,10,20,40];     // different zooms in nm
var zoomfactor = 2;   // start with 10 nm
var speechOn = false;    // speech output
var synth;   // global speechSynthesis variable

var altDiff = [5,10,20,50,100,500];   // Threshold to display other planes within altitude difference in 100 ft
var altindex = 2;  // start with 2000 ft
var AltDiffThreshold;    // in 100 feet display value
var storageDiff = 20;   // altitude difference in 100 ft below airplane is stored in list (otherwise do not even consider, performance optimization

var situation = {};


var sound_alert = new Audio('alert.wav');

function RadarCtrl($rootScope, $scope, $state, $http, $interval) {

        //  basics();
	$scope.$parent.helppage = 'plates/radar-help.html';
	$scope.data_list = [];
	$scope.data_list_invalid = [];

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

	function radiansRel (angle) {     //adopted from equations.go
		if (angle>180) angle = angle -360;
		if (angle<=-180) angle = angle +360;
		return angle * Math.PI / 180;
        }

	// get situation data and turn radar
	function ownSituation(data)
	{
	      situation = angular.fromJson(data);
	      // consider using angular.extend()
	      $scope.raw_data = angular.toJson(data, true); // makes it pretty
	      Lat = situation.GPSLatitude;
	      Long = situation.GPSLongitude;
	      GPSCourse = situation.GPSTrueCourse;
	      var press_time = Date.parse(situation.BaroLastMeasurementTime);
	      var gps_time = Date.parse(situation.GPSLastGPSTimeStratuxTime);
	      if (gps_time - press_time < 1000) {    //pressure is ok
		    BaroAltitude = Math.round(situation.BaroPressureAltitude.toFixed(0));
		    $scope.BaroAltValid = "Baro";
	      } else {
		   var gps_horizontal_accuracy = situation.GPSHorizontalAccuracy.toFixed(1);
		   if ( gps_horizontal_accuracy > 19999) {   //no valid gps signal
		       $scope.BaroAltValid = "Invalid";
		       BaroAltitude = -100000;   // marks invalid value
		   } else {
		       $scope.BaroAltValid = "GPS";
		       BaroAltitude= situation.GPSAltitudeMSL.toFixed(1);
		   }
	      }
	       var gps_horizontal_accuracy = situation.GPSHorizontalAccuracy.toFixed(1);
	       if ( gps_horizontal_accuracy > 19999) {   //no valid gps signal
		   $scope.GPSValid = "Invalid";
	       } else {
		   $scope.GPSValid = "Valid";
	       }
	       $scope.$apply();
	}
	
	function speaktraffic(altitudeDiff, direction) {
		if ( speechOn ) {
			var feet = altitudeDiff * 100;
			var sign = "plus";
			if (altitudeDiff <= 0 ) sign = "minus";
			var txt = "Traffic ";
			if ( direction) txt += direction +" o'clock ";
			txt += sign + " " + feet + " feet";
			var utterOn = new SpeechSynthesisUtterance(txt);
			utterOn.lang="en-US";
			utterOn.rate=1.1;
			synth.speak(utterOn);
		}
	}
			
	function checkCollisionVector(traffic) {
		var doUpdate = 0;    //1 if update has to be done;
		var altDiff;   //difference of altitude to my altitude
		var altDiffValid = 3;  // 1 = valid difference 2 = absolute height 3 = u/s
		var distcirc = (-traffic.ema - 6) * (-traffic.ema -6) / 30;    //distance approx. in nm, 6dB for double distance
		var distx = Math.round(200/DisplayRadius * distcirc);   // x position for circle
		var ctxt;

		if (traffic.circ) {   //delete circle + Text
			traffic.circ.remove().forget();   // undisplay elements and clear children
			doUpdate = 1;
		}
		//console.log("Mode S %d traffic. Distance %f nm Distx %d \n", traffic.icao_int, distcirc, distx);
		if ( distx < minimalCircle ) distx = minimalCircle;
		if ( BaroAltitude > -100000 ) {  // valid BaroAlt or valid GPSAlt
		   altDiff = Math.round((traffic.altitude - BaroAltitude) / 100);;
		   altDiffValid = 1;
		} else {
		   altDiffValid = 2;   //altDiff is absolute height
		}
		if ( traffic.altitude == 0 ) altDiffValid = 3;   //set height to unspecified, nothing displayed for now
		if ( distx <= 200 ) {
		   if ( ((altDiffValid == 1) && (Math.abs(altDiff) <= AltDiffThreshold)) || (altDiffValid ==2) ) {
			doUpdate=1;
			if ( distcirc<=(DisplayRadius/2) ) {
				if (!traffic.alarms) traffic.alarms = 0;
				if ( speechOn && (traffic.alarms <MaxSpeechAlarms) && (altDiffValid == 1)) speaktraffic(altDiff, null);
				if (traffic.alarms <MaxAlarms ) sound_alert.play();  // play alarmtone max times
				traffic.alarms = traffic.alarms + 1;
			} else {
			        if ( distcirc >= (DisplayRadius*0.75) ) {   // implement hysteresis, play tone again only if 3/4 of DisplayRadius outside 
					traffic.alarms = 0;   // reset alarm counter, to play again
				}
                        }
			traffic.circ = radar.allScreen.group();  //not rotated
    			var circle = radar.allScreen.circle(distx*2).cx(0).cy(0).addClass('greenCirc');
			traffic.circ.add(circle);
			if (!traffic.posangle)  {   //not yet with position for text
				traffic.posangle = posangle;
				posangle = posangle + 3*Math.PI/16;
				if (posangle > (2*Math.PI)) posangle = Math.PI;   // make sure only upper half is used 
			}
			if (altDiffValid == 1) {
				var vorzeichen = "+";  
				if (altDiff < 0) vorzeichen = "-";
				var pfeil = "";
				if ( traffic.vspeed > 0 ) pfeil = '\u2191';
				if ( traffic.vspeed < 0 ) pfeil = '\u2193';
			        var ctxt = vorzeichen+Math.abs(altDiff)+pfeil;
			} else if (altDiffValid == 2 ) {
				ctxt = traffic.altitude;
			} else {
				ctxt = "u/s";
			}
			var dx = Math.round(distx*Math.cos(traffic.posangle));
			var dy = Math.round(distx*Math.sin(traffic.posangle));
		        var outtext = radar.allScreen.text(ctxt).center(dx,dy).addClass('textCOut'); //Outline in black 
			traffic.circ.add(outtext);
		        var tratext = radar.allScreen.text(ctxt).center(dx,dy).addClass('textCirc'); //not rotated
			traffic.circ.add(tratext);
		        var tailout = radar.allScreen.text(traffic.tail).center(dx,dy-16).addClass('textRegOut');  
			traffic.circ.add(tailout);
		        var tailtext = radar.allScreen.text(traffic.tail).center(dx,dy-16).addClass('textCircReg');
			traffic.circ.add(tailtext);
                    } 
		} 
		if ( doUpdate == 1) radar.update();
        }

	function checkCollisionVectorValid(traffic) {
             	var radius_earth = 6371008.8;  // in meters
		//simplified from equations.go 
		var avgLat,distanceLat,distanceLng;
		var doUpdate = 0;

		if (traffic.planeimg) {   //delete Images + Text
			traffic.planeimg.remove().forget();
			traffic.planetext.remove().forget();  
			traffic.planespeed.remove().forget();  
			traffic.planetail.remove().forget();  
			// do not remove radar-trace
			doUpdate = 1;
                }
		var altDiff;   //difference of altitude to my altitude
		if ( BaroAltitude > -100000 ) {  // valid BaroAlt or valid GPSAlt
		   altDiff = Math.round((traffic.altitude - BaroAltitude) / 100);;
		} else {
		   altDiff = traffic.altitude;   //take absolute altitude
                }
		if ( Math.abs(altDiff) > AltDiffThreshold )  {
		   if ( doUpdate == 1) radar.update();
                   return;    //finished
                }

		avgLat = radiansRel((Lat+traffic.lat)/2);
                distanceLat = (radiansRel(traffic.lat-Lat) * radius_earth) / 1852;
                distanceLng = ((radiansRel(traffic.lon-Long) * radius_earth) / 1852) * Math.abs(Math.cos(avgLat));

		var distx = Math.round(200 / DisplayRadius*distanceLng);
		var disty = -Math.round(200 / DisplayRadius*distanceLat);
		var distradius = Math.sqrt((distanceLat*distanceLat) + (distanceLng*distanceLng));   // pythagoras
		//console.log("Alt %f Long %f Lat %f DistanceLong %f DistLat %f Heading %f dx %d dy %d\n", traffic.altitude, traffic.lon, traffic.lat, distanceLat, distanceLng, traffic.heading, distx, disty);
		if ( distradius<=DisplayRadius ) {
		        doUpdate = 1;	
			if ( distradius <=(DisplayRadius/2) ) {
				if (!traffic.alarms) traffic.alarms = 0;
				if ( speechOn && (traffic.alarms <MaxSpeechAlarms) ) {
				  var alpha = 0;
				  if ( disty >=0 ) { 
					alpha = Math.PI - Math.atan(distx/disty);
 				  } else {
					alpha = -Math.atan(distx/disty);
			          }
				  alpha = alpha * 360 / (2*Math.PI );   // convert to angle
				  alpha = alpha - GPSCourse;  // substract own GPSCourse
				  if ( alpha<0)  alpha +=360;
				  var oclock = Math.round(alpha/30);
				  if (oclock <= 0 ) oclock += 12;
				  console.log("Distx %d Disty %d GPSCourse %f alpha-Course %f oclock %f\n", distx, disty, GPSCourse, alpha, oclock);
				  speaktraffic(altDiff, oclock);
				}
				if (traffic.alarms <=MaxAlarms ) sound_alert.play();  // play alarmtone max 5 times
				traffic.alarms = traffic.alarms + 1;
			} else {
				traffic.alarms = 0;   // reset counter ones outside alarm circle
			}
			var heading = 0;
			if (traffic.heading != "---" ) heading = traffic.heading;  //sometimes heading is missing, then set to zero

			traffic.planeimg = radar.rScreen.group();
			traffic.planeimg.path("m 32,6.5 0.5,0.9 0.4,1.4 5.3,0.1 -5.3,0.1 0.1,0.5 0.3,0.1 0.6,0.4 0.4,0.4 0.4,0.8 1.1,7.1 0.1,0.8 3.7,1.7 22.2,1.3 0.5,0.1 0.3,0.3 0.3,0.7 0.2,6 -0.1,0.1 -26.5,2.8 -0.3,0.1 -0.4,0.3 -0.3,0.5 -0.1,0.3 -0.9,6.3 -1.7,10.3 9.5,0 0.2,0.1 0.2,0.2 -0.1,4.6 -0.2,0.2 -8.8,0 -1.1,-2.4 -0.2,2.5 -0.3,2.5 -0.3,-2.5 -0.2,-2.5 -1.1,2.4 -8.8,0 -0.2,-0.2 -0.1,-4.6 0.2,-0.2 0.2,-0.1 9.5,0 -1.7,-10.3 -0.9,-6.3 -0.1,-0.3 -0.3,-0.5 -0.4,-0.3 -0.3,-0.1 -26.5,-2.8 -0.1,-0.1 0.2,-6 0.3,-0.7 0.3,-0.3 0.5,-0.1 22.2,-1.3 3.7,-1.7 0,-0.8 1.2,-7.1 0.4,-0.8 0.4,-0.4 0.6,-0.4 0.3,-0.1 0.1,-0.5 -5.3,-0.1 5.3,-0.1 0.4,-1.4 z")
				.addClass('plane').size(30,30).center(distx,disty+3);
			traffic.planeimg.circle(2).center(distx,disty).addClass('planeRotationPoint');
			traffic.planeimg.rotate(heading, distx, disty);

			var vorzeichen = "+";   
			if (altDiff < 0) vorzeichen = "-";
			var pfeil = "";
			if ( traffic.vspeed > 0 ) pfeil = '\u2191';
			if ( traffic.vspeed < 0 ) pfeil = '\u2193';
			traffic.planetext = radar.rScreen.text(vorzeichen + Math.abs(altDiff)+pfeil).move(distx+17,disty-10)
				.rotate(GPSCourse, distx, disty).addClass('textPlane');
			traffic.planespeed = radar.rScreen.text(traffic.nspeed + 'kts').move(distx+17,disty)
				.rotate(GPSCourse, distx, disty).addClass('textPlaneSmall');
			traffic.planetail = radar.rScreen.text(traffic.tail).move(distx+17,disty+10)
				.rotate(GPSCourse, distx, disty).addClass('textPlaneReg');
			if (!traffic.trace) {
			     traffic.trace = radar.rScreen.polyline([[distx,disty]]).addClass('trace');
			} else {
				var points = traffic.trace.attr('points');
				points += ' '+[distx,disty];
				traffic.trace.attr('points', points);
			}
		} else {   // if airplane is outside of radarscreen
			if ( traffic.trace ) {   //remove trace when aircraft gets out of range
			     traffic.trace.remove().forget(); 
			     traffic.trace = '';;
			     doUpdate = 1;
			}
			traffic.alarms = 0;   //reset alarm counter
	        }	
		if ( doUpdate == 1) radar.update();   // only necessary if changes were done
	}

	function expMovingAverage (oldema, newsignal, timelack) {
		var lambda = 0.5;
		if (!newsignal) {    //sometimes newsign=NaN
			return oldema;
		}
		if ( timelack < 0) {
			return newsignal;
		}
		var expon = Math.exp(-timelack/100*lambda);
	   	//console.log("Signal %f oldema %f timelack %f new_ema %f\n", newsignal, oldema, timelack, oldema*expon + newsignal*(1-expon));
		return oldema*expon + newsignal*(1-expon);
	}

	function setAircraft(obj, new_traffic) {
		new_traffic.icao_int = obj.Icao_addr;
		new_traffic.targettype = obj.TargetType;
		var timestamp = Date.parse(obj.Timestamp);
		var timeLack = -1;
		if (new_traffic.timeVal >0 ) {
			timeLack = timestamp - new_traffic.timeVal;
		} 
		new_traffic.timeVal = timestamp;
		new_traffic.time = utcTimeString(timestamp);
		new_traffic.signal = obj.SignalLevel;
		new_traffic.ema = expMovingAverage(new_traffic.ema, new_traffic.signal, timeLack);

		new_traffic.lat = obj.Lat;
		new_traffic.lon = obj.Lng;
		var n = Math.round(obj.Alt / 25) * 25;
		new_traffic.altitude = n;

		if (obj.Speed_valid) {
			new_traffic.nspeed = Math.round(obj.Speed / 5) * 5;
                        new_traffic.heading = Math.round(obj.Track / 5) * 5;
                } else {
			new_traffic.nspeed = "-";
                        new_traffic.heading = "---";
                }
                new_traffic.vspeed = Math.round(obj.Vvel / 100) * 100


		new_traffic.age = obj.Age;
		new_traffic.ageLastAlt = obj.AgeLastAlt;
                new_traffic.dist = (obj.Distance/1852); 
		new_traffic.tail = obj.Tail;   //registration No
	}

	function onMessageNew (msg) {
			
		var message = JSON.parse(msg.data);
		//$scope.raw_data = angular.toJson(msg.data, true);
			
		// we need to use an array so AngularJS can perform sorting; it also means we need to loop to find an aircraft in the traffic set
		// only aircraft in possible display position are stored
		var validIdx = -1;
		var invalidIdx = -1;
		var altDiffValid = false; 
		var altDiff;
		if ( (BaroAltitude > -100000) && (message.Alt > 0)) {  // valid BaroAlt or valid GPSAlt and valid altitude
			altDiff = Math.round((message.Alt - BaroAltitude) / 100);
			altDiffValid = true;
		}
		for (var i = 0, len = $scope.data_list.length; i < len; i++) {
			if ($scope.data_list[i].icao_int === message.Icao_addr) {
				setAircraft(message, $scope.data_list[i]);
				if (message.Position_valid) checkCollisionVectorValid($scope.data_list[i]);
				validIdx = i;
				break;  // break in anycase, if once found
			}
		}
				
		if ( validIdx < 0 ) {   // not yet found
		   for (var i = 0, len = $scope.data_list_invalid.length; i < len; i++) {
			if ($scope.data_list_invalid[i].icao_int === message.Icao_addr) {
				setAircraft(message, $scope.data_list_invalid[i]);
				if (!message.Position_valid) checkCollisionVector($scope.data_list_invalid[i]);
				//console.log($scope.data_list_invalid[i]);
				invalidIdx = i;
				break;
			}
                   }
		}
		var new_traffic = {};
				
		if ((validIdx < 0) && (message.Position_valid)) {    //new aircraft with valid position
			if ( altDiffValid && (Math.abs(altDiff) <= AltDiffThreshold) )  { 
				// optimization: only store ADSB aircraft if inside altDiff
				setAircraft(message, new_traffic);
				checkCollisionVectorValid(new_traffic);
				$scope.data_list.unshift(new_traffic); // add to start of valid array.
			}    // else not added in list, since not relevant 
		}

		if ((invalidIdx < 0) && (!message.Position_valid)) {     // new aircraft without position
			if ( altDiffValid && (Math.abs(altDiff) <= (AltDiffThreshold + storageDiff)) )  { 
				setAircraft(message, new_traffic);  //store in any case, since EMA needs history of dB
				checkCollisionVector(new_traffic);
				$scope.data_list_invalid.unshift(new_traffic); // add to start of invalid array.
			}    // else not added in list, since not relevant 
		}

		// Handle the negative cases of those above - where an aircraft moves from "valid" to "invalid" or vice-versa.
		if ((validIdx >= 0) && (!message.Position_valid)) {    //known valid aircraft now with invalid position
			// Position is not valid any more. Remove from "valid" table.
			if ( $scope.data_list[validIdx].planeimg ) { 
				$scope.data_list[validIdx].planeimg.remove().forget();  // remove plane image
				$scope.data_list[validIdx].planetext.remove().forget();  // remove plane image
				$scope.data_list[validIdx].planespeed.remove().forget();  // remove plane image
				$scope.data_list[validIdx].planetail.remove().forget();  // remove plane image
				if ( $scope.data_list[validIdx].trace ) { 
				    $scope.data_list[validIdx].trace.remove().forget();  // remove plane image
				    $scope.data_list[validIdx].trace = '';
				}
			}
			$scope.data_list.splice(validIdx, 1);
		}

		if ((invalidIdx >= 0) && message.Position_valid) {   //known invalid aircraft now with valid position
			// Position is now valid. Remove from "invalid" table.
			if ($scope.data_list_invalid[invalidIdx].circ) {   //delete circle + Text
				$scope.data_list_invalid[invalidIdx].circ.remove().forget();
				delete $scope.data_list_invalid[invalidIdx].posangle;  //make sure angle is not used again
			
			}
			$scope.data_list_invalid.splice(invalidIdx, 1);
		}

		$scope.$apply();
	}

	function connect($scope) {
		if (($scope === undefined) || ($scope === null))
			return; // we are getting called once after clicking away from the status page

		if (($scope.socket === undefined) || ($scope.socket === null)) {
			socket = new WebSocket(URL_TRAFFIC_WS);
			$scope.socket = socket; // store socket in scope for enter/exit usage
                        sit_socket = new WebSocket(URL_GPS_WS);  // socket for situation
			$scope.sit_socket = sit_socket;
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
			//ownSituation($scope);   move to getclock
			onMessageNew(msg);
		        //radar.update();   moved to changes
		};

		sit_socket.onopen = function (msg) {
			//nothing, status is set with traffic port
		};

		sit_socket.onclose = function (msg) {
			setTimeout(connect, 1000);
		};

		sit_socket.onerror = function (msg) {
			//nothing, status is set with traffic port
		};

		sit_socket.onmessage = function (msg) {
			ownSituation(msg.data); 
		        radar.update(); 
		};
	}

	var getClock = $interval(function () {
		$http.get(URL_STATUS_GET).
		then(function (response) {
			globalStatus = angular.fromJson(response.data);
				
			var tempClock = new Date(Date.parse(globalStatus.Clock));
			var clockString = tempClock.toUTCString();
			$scope.Clock = clockString;

			var tempUptimeClock = new Date(Date.parse(globalStatus.UptimeClock));
			var uptimeClockString = tempUptimeClock.toUTCString();
			$scope.UptimeClock = uptimeClockString;

			var tempLocalClock = new Date;
			$scope.LocalClock = tempLocalClock.toUTCString();
			$scope.SecondsFast = (tempClock-tempLocalClock)/1000;
			
			$scope.GPS_connected = globalStatus.GPS_connected;
		        var boardtemp = globalStatus.CPUTemp;
		        if (boardtemp != undefined) {
			     /* boardtemp is celcius to tenths */
			     $scope.CPUTemp = boardtemp.toFixed(1);
		        }
			radar.update();
						
		}, function (response) {
			radar.update();  // just update, if status gets error
		});
	}, 500, 0, false);
		


	// perform cleanup every 10 seconds
	var clearStaleTraffic = $interval(function () {
		// remove stale aircraft = anything more than 20 seconds without a position update
		var cutoff = 59;

		// Clean up "valid position" table.
		for (var i = $scope.data_list.length; i > 0; i--) {
			if ($scope.data_list[i - 1].age >= cutoff) {
				if ( $scope.data_list[i-1].planeimg ) { 
					$scope.data_list[i-1].planeimg.remove().forget();  // remove plane image
					$scope.data_list[i-1].planetext.remove().forget();  // remove plane image
					$scope.data_list[i-1].planespeed.remove().forget();  // remove plane image
					$scope.data_list[i-1].planetail.remove().forget();  // remove plane image
					if ( $scope.data_list[i-1].trace ) { 
					    $scope.data_list[i-1].trace.remove().forget();  // remove plane image
					    $scope.data_list[i-1].trace = '';
					}
				}
				$scope.data_list.splice(i - 1, 1);
			}
		}

		// Clean up "invalid position" table.
		for (var i = $scope.data_list_invalid.length; i > 0; i--) {
			if (($scope.data_list_invalid[i - 1].age >= cutoff) || ($scope.data_list_invalid[i - 1].ageLastAlt >= cutoff)) {
				if ( $scope.data_list_invalid[i-1].circ ) {    // is displayed
				    $scope.data_list_invalid[i-1].circ.remove().forget(); 
				}
				$scope.data_list_invalid.splice(i - 1, 1);
			}
		}
	}, (1000 * 10), 0, false);


	$state.get('radar').onEnter = function () {
		// everything gets handled correctly by the controller
	};

	$state.get('radar').onExit = function () {
		// disconnect from the socket
		if (($scope.socket !== undefined) && ($scope.socket !== null)) {
			$scope.socket.close();
			$scope.socket = null;
		}
		// stop stale traffic cleanup
		$interval.cancel(clearStaleTraffic);
	};

	radar = new RadarRenderer ("radar_display",$scope);

	// Traffic Controller tasks
	connect($scope); // connect - opens a socket and listens for messages
}

function clearRadarTraces ($scope) {
	for (var i = $scope.data_list.length; i > 0; i--) {
	     if ( $scope.data_list[i-1].planeimg ) { 
		$scope.data_list[i-1].planeimg.remove().forget();  // remove plane image
		$scope.data_list[i-1].planetext.remove().forget();  // remove plane image
		$scope.data_list[i-1].planespeed.remove().forget();  // remove plane image
		$scope.data_list[i-1].planetail.remove().forget();  // remove plane image
		$scope.data_list[i-1].alarms = 0; //reset alarm counter
                if ( $scope.data_list[i-1].trace ) {
			$scope.data_list[i-1].trace.remove().forget();  // remove plane image
			$scope.data_list[i-1].trace = '';
		}
	     }
	}
}

function requestFullScreen(el) {
   // Supports most browsers and their versions.
   var requestMethod = el.requestFullscreen || el.webkitRequestFullScreen || el.mozRequestFullScreen || el.msRequestFullscreen;
   if (requestMethod) requestMethod.call(el);
}


function cancelFullScreen(el) {
   var requestMethod = el.cancelFullScreen||el.webkitCancelFullScreen||el.mozCancelFullScreen||el.exitFullscreen;
   if (requestMethod) { // cancel full screen.
        requestMethod.call(el);
   } else if (typeof window.ActiveXObject !== "undefined") { // Older IE.
       var wscript = new ActiveXObject("WScript.Shell");
       if (wscript !== null) {
            wscript.SendKeys("{F11}");
       }
   }
}

function RadarRenderer(locationId,$scope) {
    this.width = -1;
    this.height = -1;

    this.locationId = locationId;
    this.canvas = document.getElementById(this.locationId);
    this.resize();

    AltDiffThreshold = altDiff[altindex];	
    DisplayRadius = zoom[zoomfactor];	

    // Draw the radar using the svg.js library
    var radarAll = SVG(this.locationId).viewbox(-201, -201, 402, 302).group().addClass('radar');
    var background = radarAll.rect(402,402).radius(5).x(-201).y(-201).addClass('blackRect');
    var card = radarAll.group().addClass('card');
    card.circle(400).cx(0).cy(0);  
    card.circle(200).cx(0).cy(0);  
    this.displayText = radarAll.text(DisplayRadius+' nm').addClass('textOutside').x(-200).cy(-158);  //not rotated
    this.altText = radarAll.text('\xB1'+AltDiffThreshold+'00ft').addClass('textOutsideRight').x(200).cy(-158);  //not rotated
    card.text("N").addClass('textDir').center(0,-190);
    card.text("S").addClass('textDir').center(0,190);
    card.text("W").addClass('textDir').center(-190,0);
    card.text("E").addClass('textDir').center(190,0);

    var middle=radarAll.path("m 32,6.5 0.5,0.9 0.4,1.4 5.3,0.1 -5.3,0.1 0.1,0.5 0.3,0.1 0.6,0.4 0.4,0.4 0.4,0.8 1.1,7.1 0.1,0.8 3.7,1.7 22.2,1.3 0.5,0.1 0.3,0.3 0.3,0.7 0.2,6 -0.1,0.1 -26.5,2.8 -0.3,0.1 -0.4,0.3 -0.3,0.5 -0.1,0.3 -0.9,6.3 -1.7,10.3 9.5,0 0.2,0.1 0.2,0.2 -0.1,4.6 -0.2,0.2 -8.8,0 -1.1,-2.4 -0.2,2.5 -0.3,2.5 -0.3,-2.5 -0.2,-2.5 -1.1,2.4 -8.8,0 -0.2,-0.2 -0.1,-4.6 0.2,-0.2 0.2,-0.1 9.5,0 -1.7,-10.3 -0.9,-6.3 -0.1,-0.3 -0.3,-0.5 -0.4,-0.3 -0.3,-0.1 -26.5,-2.8 -0.1,-0.1 0.2,-6 0.3,-0.7 0.3,-0.3 0.5,-0.1 22.2,-1.3 3.7,-1.7 0,-0.8 1.2,-7.1 0.4,-0.8 0.4,-0.4 0.6,-0.4 0.3,-0.1 0.1,-0.5 -5.3,-0.1 5.3,-0.1 0.4,-1.4 z");
    middle.size(25,25).center(0,3).addClass('centerplane');
    radarAll.circle(2).center(0,0).addClass('planeRotationPoint');

    var zoomin = radarAll.group().cx(-120).cy(-190).addClass('zoom');
    zoomin.circle(45).cx(0).cy(0).addClass('zoom');
    zoomin.text('Ra-').cx(12).cy(2).addClass('textZoom');
    zoomin.on('click', function () {
	var animateTime= 200;
        if (zoomfactor > 0 ) { 
		zoomfactor--;     
        } else {  
		animateTime = 20;
	}
        DisplayRadius = zoom[zoomfactor];	
	zoomin.animate(animateTime).rotate(90, 0, 0);
        this.displayText.text(DisplayRadius+' nm');
	//update();
        zoomin.animate(animateTime).rotate(0, 0, 0);
	clearRadarTraces($scope);
    }, this);

    var zoomout = radarAll.group().cx(-177).cy(-190).addClass('zoom');
    zoomout.circle(45).cx(0).cy(0).addClass('zoom');
    zoomout.text('Ra+').cx(12).cy(2).addClass('textZoom');
    zoomout.on('click', function () {
	var animateTime= 200;
        if (zoomfactor < (zoom.length-1) ) { 
		zoomfactor++;     
        } else {  
		animateTime = 20;
	}
        DisplayRadius = zoom[zoomfactor];	
	zoomout.animate(animateTime).rotate(90, 0, 0);
        this.displayText.text(DisplayRadius+' nm');
        zoomout.animate(animateTime).rotate(0, 0, 0);
	clearRadarTraces($scope);
    }, this);

    var altmore = radarAll.group().cx(120).cy(-190).addClass('zoom');
    altmore.circle(45).cx(0).cy(0).addClass('zoom');
    altmore.text('Alt+').cx(12).cy(2).addClass('textZoom');
    altmore.on('click', function () {
	var animateTime= 200;
        if (altindex < (altDiff.length-1) ) { 
		altindex++;     
        } else {  
		animateTime = 20;
	}
        AltDiffThreshold = altDiff[altindex];	
	altmore.animate(animateTime).rotate(90, 0, 0);
        this.altText.text('\xB1'+AltDiffThreshold+'00ft');
	//update();
        altmore.animate(animateTime).rotate(0, 0, 0);
	clearRadarTraces($scope);
    }, this);

    var altless = radarAll.group().cx(177).cy(-190).addClass('zoom');
    altless.circle(45).cx(0).cy(0).addClass('zoom');
    altless.text('Alt-').cx(12).cy(2).addClass('textZoom');
    altless.on('click', function () {
	var animateTime= 200;
        if (altindex > 0 ) { 
		altindex--;     
        } else {  
		animateTime = 20;
	}
        AltDiffThreshold = altDiff[altindex];	
	altless.animate(animateTime).rotate(90, 0, 0);
        //update();
        this.altText.text('\xB1'+AltDiffThreshold+'00ft');
        altless.animate(animateTime).rotate(0, 0, 0);
	clearRadarTraces($scope);
    }, this);

    var fullscreen = radarAll.group().cx(185).cy(-125).addClass('zoom');
    fullscreen.rect(40,35).radius(10).cx(0).cy(0).addClass('zoom');
    fullscreen.text('F/S').cx(10).cy(2).addClass('textZoom');
    fullscreen.on('click', function () {
	   var elem = this.canvas;
           var isInFullScreen = (document.fullScreenElement && document.fullScreenElement !== null) ||  (document.mozFullScreen || document.webkitIsFullScreen);

            if (isInFullScreen) {
		fullscreen.get(0).removeClass('zoomInvert').addClass('zoom');
		fullscreen.get(1).removeClass('textZoomInvert').addClass('textZoom');
                cancelFullScreen(document);
            } else {
		fullscreen.get(0).removeClass('zoom').addClass('zoomInvert');
		fullscreen.get(1).removeClass('textZoom').addClass('textZoomInvert');
                requestFullScreen(elem);
            }
    }, this);


    var speech = radarAll.group().cx(-185).cy(-125).addClass('zoom');
    speech.rect(40,35).radius(10).cx(0).cy(0).addClass('zoom');
    speech.text('Spk').cx(12).cy(2).addClass('textZoom');
    synth = window.speechSynthesis;

    speech.on('click', function () {
	if (!synth) return;    // speech function not working
	if ( ! speechOn ) {
    		var utterOn = new SpeechSynthesisUtterance("Speech on");
		utterOn.lang="en-US";
		utterOn.rate=1.1;
		speech.get(0).removeClass('zoom').addClass('zoomInvert');
		speech.get(1).removeClass('textZoom').addClass('textZoomInvert');
		synth.speak(utterOn);
		speechOn = true;
	} else {
    		var utterOff = new SpeechSynthesisUtterance("Speech off");
		utterOff.lang="en-US";
    		utterOff.rate=1.1;
		speech.get(0).removeClass('zoomInvert').addClass('zoom');
		speech.get(1).removeClass('textZoomInvert').addClass('textZoom');
		synth.speak(utterOff);
		speechOn = false;
        }
    }, this);



    this.allScreen = radarAll;
    this.rScreen = card;
}

RadarRenderer.prototype = {
    constructor: RadarRenderer,

    resize: function () {
        var canvasWidth = this.canvas.parentElement.offsetWidth - 12;

        if (canvasWidth !== this.width) {
            this.width = canvasWidth;
            this.height = canvasWidth * 0.5;

            this.canvas.width = this.width;
            this.canvas.height = this.height;
        }
    },

    update: function () {
	 if (this.fl) this.fl.remove();
	 this.rScreen.rotate(-GPSCourse,0,0);    // rotate conforming to GPSCourse
	 this.fl = this.allScreen.text("FL"+Math.round(BaroAltitude/100)).addClass('textSmall').move(7,5); 
    }
};
