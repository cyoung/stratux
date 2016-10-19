angular.module('appControllers').controller('MessagingCtrl', MessagingCtrl); // get the main module contollers set
MessagingCtrl.$inject = ['$rootScope', '$scope', '$state', '$http', '$interval']; // Inject my dependencies

// create our controller function with all necessary logic
function MessagingCtrl($rootScope, $scope, $state, $http, $interval) {

	$scope.$parent.helppage = 'plates/messaging-help.html';
	$scope.message_list = [];

	function connect($scope) {
		if (($scope === undefined) || ($scope === null))
			return; // we are getting called once after clicking away from the status page

		if (($scope.socket === undefined) || ($scope.socket === null)) {
			socket = new WebSocket(URL_MESSAGING_WS);
			$scope.socket = socket; // store socket in scope for enter/exit usage
		}

		$scope.ConnectState = "Disconnected";

		socket.onopen = function (msg) {
			$scope.ConnectState = "Connected";
		};

		socket.onclose = function (msg) {
			$scope.ConnectState = "Disconnected";
			$scope.$apply();
			setTimeout(connect, 1000);
		};

		socket.onerror = function (msg) {
			$scope.ConnectState = "Problem";
			$scope.$apply();
		};

		socket.onmessage = function (msg) {
			console.log('Received message update.')

			var message = JSON.parse(msg.data);

			$scope.message_list.unshift(message);

			$scope.$apply();

		};
	}

	$scope.sendData = function() {
		var t = "";
		if ($scope.DataInput !== undefined)
			t = $scope.DataInput;

		sendMsg = {
			"Command": "send",
			"Data": t
		};
		j = angular.toJson(sendMsg);
		console.log("heyoo" + j + "\n");
		$scope.socket.send(j);
	};

	$state.get('messaging').onExit = function () {
		// disconnect from the socket
		if (($scope.socket !== undefined) && ($scope.socket !== null)) {
			$scope.socket.close();
			$scope.socket = null;
		}
	};

	// Messaging controller tasks.
	connect($scope); // connect - opens a socket and listens for messages
};