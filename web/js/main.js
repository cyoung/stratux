// application constants
var URL_HOST_BASE = window.location.hostname;
var URL_SETTINGS_GET = "http://" + URL_HOST_BASE + "/getSettings";
var URL_SETTINGS_SET = "http://" + URL_HOST_BASE + "/setSettings";



// define the module with dependency on mobile-angular-ui
//var app = angular.module('stratux', ['ngRoute', 'mobile-angular-ui', 'mobile-angular-ui.gestures', 'appControllers']);
var app = angular.module('stratux', ['ui.router', 'mobile-angular-ui', 'mobile-angular-ui.gestures', 'appControllers']);
var appControllers = angular.module('appControllers', []);


app.config(function ($stateProvider, $urlRouterProvider) {
	$stateProvider
		.state('home', 		{ url: '/',			templateUrl: 'plates/status.html',	controller: 'StatusCtrl',	reloadOnSearch: false })
		.state('weather', 	{ url: '/weather',	templateUrl: 'plates/weather.html',	controller: 'WeatherCtrl',	reloadOnSearch: false })
		.state('traffic', 	{ url: '/traffic',	templateUrl: 'plates/traffic.html',	controller: 'TrafficCtrl',	reloadOnSearch: false })
		.state('gps', 		{ url: '/gps',		templateUrl: 'plates/gps.html',		controller: 'GPSCtrl',		reloadOnSearch: false })
		.state('logs',		{ url: '/logs',		templateUrl: 'plates/logs.html',	controller: 'LogsCtrl',		reloadOnSearch: false })
		.state('settings', 	{ url: '/settings',	templateUrl: 'plates/settings.html',controller: 'SettingsCtrl',	reloadOnSearch: false });
	$urlRouterProvider.otherwise('/');
});


app.run(function ($transform) {
	window.$transform = $transform;
});

// For this app we have a MainController for whatever and individual controllers for each page
app.controller('MainCtrl', function ($rootScope, $scope) {
	// any logic global logic
});