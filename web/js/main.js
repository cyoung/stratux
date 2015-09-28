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
        .state('ahrs', {
            url: '/ahrs',
            templateUrl: 'plates/ahrs.html',
            controller: 'AHRSCtrl',
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
        });

    $urlRouterProvider.otherwise('/');
});


app.run(function ($transform) {
    window.$transform = $transform;
});
/*
app.config(function ($httpProvider) {
    // We need to setup some parameters for http requests
    // These three lines are all you need for CORS support
    $httpProvider.defaults.useXDomain = true;
    // $httpProvider.defaults.withCredentials = true;
    delete $httpProvider.defaults.headers.common['X-Requested-With'];
});
*/
// For this app we have a MainController for whatever and the nindividual controllers for each page
app.controller('MainCtrl', function ($rootScope, $scope) {
    // any logic global logic
});