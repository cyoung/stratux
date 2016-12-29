// application constants
var URL_HOST_BASE 		= window.location.hostname;
var URL_SETTINGS_GET 	= "http://"	+ URL_HOST_BASE + "/getSettings";
var URL_SETTINGS_SET 	= "http://"	+ URL_HOST_BASE + "/setSettings";
var URL_GPS_GET 		= "http://"	+ URL_HOST_BASE + "/getSituation";
var URL_TOWERS_GET 		= "http://"	+ URL_HOST_BASE + "/getTowers"
var URL_STATUS_GET 		= "http://"	+ URL_HOST_BASE + "/getStatus"
var URL_SATELLITES_GET	= "http://"	+ URL_HOST_BASE + "/getSatellites"
var URL_STATUS_WS 		= "ws://"	+ URL_HOST_BASE + "/status"
var URL_TRAFFIC_WS 		= "ws://"	+ URL_HOST_BASE + "/traffic";
var URL_WEATHER_WS 		= "ws://"	+ URL_HOST_BASE + "/weather";
var URL_DEVELOPER_GET   = "ws://"   + URL_HOST_BASE + "/developer";
var URL_UPDATE_UPLOAD	= "http://" + URL_HOST_BASE + "/updateUpload";
var URL_REBOOT			= "http://" + URL_HOST_BASE + "/reboot";
var URL_SHUTDOWN		= "http://" + URL_HOST_BASE + "/shutdown";
var URL_RESTARTAPP      = "http://" + URL_HOST_BASE + "/restart";
var URL_DEV_TOGGLE_GET  = "http://" + URL_HOST_BASE + "/develmodetoggle";

// define the module with dependency on mobile-angular-ui
//var app = angular.module('stratux', ['ngRoute', 'mobile-angular-ui', 'mobile-angular-ui.gestures', 'appControllers']);
var app = angular.module('stratux', ['ui.router', 'mobile-angular-ui', 'mobile-angular-ui.gestures', 'appControllers']);
var appControllers = angular.module('appControllers', []);


app.config(function ($stateProvider, $urlRouterProvider) {
	$stateProvider
		.state('home', {
			url: '/',
			templateUrl: 'plates/status.html',
			controller: 'StatusCtrl',
			reloadOnSearch: false
		})
		.state('towers', {
			url: '/towers',
			templateUrl: 'plates/towers.html',
			controller: 'TowersCtrl',
			reloadOnSearch: false
		})
		.state('weather', {
			url: '/weather',
			templateUrl: 'plates/weather.html',
			controller: 'WeatherCtrl',
			reloadOnSearch: false
		})
		.state('traffic', {
			url: '/traffic',
			templateUrl: 'plates/traffic.html',
			controller: 'TrafficCtrl',
			reloadOnSearch: false
		})
		.state('gps', {
			url: '/gps',
			templateUrl: 'plates/gps.html',
			controller: 'GPSCtrl',
			reloadOnSearch: false
		})
		.state('logs', {
			url: '/logs',
			templateUrl: 'plates/logs.html',
			controller: 'LogsCtrl',
			reloadOnSearch: false
		})
		.state('settings', {
			url: '/settings',
			templateUrl: 'plates/settings.html',
			controller: 'SettingsCtrl',
			reloadOnSearch: false
		})
        .state('developer', {
			url: '/developer',
			templateUrl: 'plates/developer.html',
			controller: 'DeveloperCtrl',
			reloadOnSearch: false
		});
	$urlRouterProvider.otherwise('/');
});


app.run(function ($transform) {
	window.$transform = $transform;
});

// For this app we have a MainController for whatever and individual controllers for each page
app.controller('MainCtrl', function ($scope, $http) {
	// any logic global logic
    $http.get(URL_SETTINGS_GET)
    .then(function(response) {
			settings = angular.fromJson(response.data);
            $scope.DeveloperMode = settings.DeveloperMode;
    }, function(response) {
        //Second function handles error
    });	
});