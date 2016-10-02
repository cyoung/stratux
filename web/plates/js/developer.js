angular.module('appControllers').controller('DeveloperCtrl', DeveloperCtrl); // get the main module contollers set
DeveloperCtrl.$inject = ['$rootScope', '$scope', '$state', '$http', '$interval']; // Inject my dependencies

// create our controller function with all necessary logic
function DeveloperCtrl($rootScope, $scope, $state, $http, $interval) {
	$scope.$parent.helppage = 'plates/developer-help.html';
	$scope.data_list = [];
};
    
