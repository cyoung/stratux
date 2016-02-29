angular.module('appControllers').controller('LogsCtrl', LogsCtrl);      // get the main module contollers set
LogsCtrl.$inject = ['$scope', '$state', '$http'];                                   // Inject my dependencies

// create our controller function with all necessary logic
function LogsCtrl($scope, $state, $http) {
	$scope.$parent.helppage = 'plates/logs-help.html';

	// just a couple environment variables that may bve useful for dev/debugging but otherwise not significant
	$scope.userAgent = navigator.userAgent;
    $scope.deviceViewport = 'screen = ' + window.screen.width + ' x ' + window.screen.height;
}
