angular.module('appControllers').controller('LogsCtrl', LogsCtrl);      // get the main module contollers set
LogsCtrl.$inject = ['$scope', '$http'];                                   // Inject my dependencies

// create our controller function with all necessary logic
function LogsCtrl($scope, $http) {
    $scope.userAgent = navigator.userAgent;
    $scope.deviceViewport = 'screen = ' + window.screen.width + ' x ' + window.screen.height;
}
