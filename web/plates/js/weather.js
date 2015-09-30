angular.module('appControllers').controller('WeatherCtrl', WeatherCtrl); // get the main module contollers set
WeatherCtrl.$inject = ['$rootScope', '$scope', '$state', '$http']; // Inject my dependencies

// create our controller function with all necessary logic
function WeatherCtrl($rootScope, $scope, $state, $http) {

    /*
    $state.get('weather').onEnter = function () {
    };
    $state.get('weather').onExit = function () {
    };
    */

    // Weather Controller tasks go here
};