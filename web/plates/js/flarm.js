angular.module('appControllers').controller('FlarmCtrl', FlarmCtrl); // get the main module contollers set
FlarmCtrl.$inject = ['$rootScope', '$scope', '$state', '$http', '$interval', '$sce']; // Inject my dependencies

function FlarmCtrl($rootScope, $scope, $state, $http, $interval, $sce) {
    $scope.ogn_rf_url = $sce.trustAsResourceUrl('http://' + window.location.hostname + ':8082');
    $scope.ogn_decode_url = $sce.trustAsResourceUrl('http://' + window.location.hostname + ':8083');
}