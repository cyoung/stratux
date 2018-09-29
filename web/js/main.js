// application constants
var URL_HOST_BASE           = window.location.hostname + (window.location.port ? ':' + window.location.port : '');
var URL_HOST_PROTOCOL       = window.location.protocol + "//";

var URL_AHRS_CAGE           = URL_HOST_PROTOCOL + URL_HOST_BASE + "/cageAHRS";
var URL_AHRS_CAL            = URL_HOST_PROTOCOL + URL_HOST_BASE + "/calibrateAHRS";
var URL_AHRS_ORIENT         = URL_HOST_PROTOCOL + URL_HOST_BASE + "/orientAHRS";
var URL_DELETEAHRSLOGFILES  = URL_HOST_PROTOCOL + URL_HOST_BASE + "/deleteahrslogfiles";
var URL_DELETELOGFILE       = URL_HOST_PROTOCOL + URL_HOST_BASE + "/deletelogfile";
var URL_DEV_TOGGLE_GET      = URL_HOST_PROTOCOL + URL_HOST_BASE + "/develmodetoggle";
var URL_DOWNLOADAHRSLOGFILES = URL_HOST_PROTOCOL + URL_HOST_BASE + "/downloadahrslogs";
var URL_DOWNLOADDB          = URL_HOST_PROTOCOL + URL_HOST_BASE + "/downloaddb";
var URL_DOWNLOADLOGFILE     = URL_HOST_PROTOCOL + URL_HOST_BASE + "/downloadlog";
var URL_GMETER_RESET        = URL_HOST_PROTOCOL + URL_HOST_BASE + "/resetGMeter";
var URL_REBOOT              = URL_HOST_PROTOCOL + URL_HOST_BASE + "/reboot";
var URL_RESTARTAPP          = URL_HOST_PROTOCOL + URL_HOST_BASE + "/restart";
var URL_SATELLITES_GET      = URL_HOST_PROTOCOL + URL_HOST_BASE + "/getSatellites";
var URL_SETTINGS_GET        = URL_HOST_PROTOCOL + URL_HOST_BASE + "/getSettings";
var URL_SETTINGS_SET        = URL_HOST_PROTOCOL + URL_HOST_BASE + "/setSettings";
var URL_SHUTDOWN            = URL_HOST_PROTOCOL + URL_HOST_BASE + "/shutdown";
var URL_STATUS_GET          = URL_HOST_PROTOCOL + URL_HOST_BASE + "/getStatus";
var URL_TOWERS_GET          = URL_HOST_PROTOCOL + URL_HOST_BASE + "/getTowers";
var URL_UPDATE_UPLOAD       = URL_HOST_PROTOCOL + URL_HOST_BASE + "/updateUpload";

var URL_DEVELOPER_WS        = "ws://" + URL_HOST_BASE + "/developer";
var URL_GPS_WS              = "ws://" + URL_HOST_BASE + "/situation";
var URL_STATUS_WS           = "ws://" + URL_HOST_BASE + "/status";
var URL_TRAFFIC_WS          = "ws://" + URL_HOST_BASE + "/traffic";
var URL_WEATHER_WS          = "ws://" + URL_HOST_BASE + "/weather";

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
			var settings = angular.fromJson(response.data);
            $scope.DeveloperMode = settings.DeveloperMode;
    }, function(response) {
        //Second function handles error
    });	
});
