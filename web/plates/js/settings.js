angular.module('appControllers').controller('SettingsCtrl', SettingsCtrl); // get the main module contollers set
SettingsCtrl.$inject = ['$rootScope', '$scope', '$state', '$http']; // Inject my dependencies

// create our controller function with all necessary logic
function SettingsCtrl($rootScope, $scope, $state, $http) {

    $scope.$parent.helppage = 'plates/settings-help.html';
    
    var toggles = ['UAT_Enabled', 'ES_Enabled', 'GPS_Enabled', 'AHRS_Enabled', 'DEBUG', 'ReplayLog']; // DEBUG is 'DspTrafficSrc'
    var settings = {};
    for (i = 0; i < toggles.length; i++) {
        settings[toggles[i]] = undefined;
    }

    function getSettings() {
        // Simple GET request example (note: responce is asynchronous)
        $http.get(URL_SETTINGS_GET).
        then(function (response) {
            //process
            $scope.rawSettings = response.data; // angular.toJson(response.data, true);
            settings = angular.fromJson(response.data);
            $scope.UAT_Enabled = settings.UAT_Enabled;
            $scope.ES_Enabled = settings.ES_Enabled;
            $scope.GPS_Enabled = settings.GPS_Enabled;
            $scope.AHRS_Enabled = settings.AHRS_Enabled;
            $scope.DEBUG = settings.DEBUG;
            $scope.ReplayLog = settings.ReplayLog;
            $scope.PPM = settings.PPM;
            // $scope.$apply();
        }, function (response) {
            $scope.rawSettings = "error getting settings";
            for (i = 0; i < toggles.length; i++) {
                settings[toggles[i]] = false;
            }

        });
    };

    function setSettings(msg) {
        // Simple POST request example (note: responce is asynchronous)
        $http.post(URL_SETTINGS_SET, msg).
        then(function (response) {
            //process
            $scope.rawSettings = response.data; // angular.toJson(response.data, true);
            settings = angular.fromJson(response.data);
            $scope.UAT_Enabled = settings.UAT_Enabled;
            $scope.ES_Enabled = settings.ES_Enabled;
            $scope.GPS_Enabled = settings.GPS_Enabled;
            $scope.AHRS_Enabled = settings.AHRS_Enabled;
            $scope.DEBUG = settings.DEBUG;
            $scope.ReplayLog = settings.ReplayLog;
            $scope.PPM = parseInt(settings.PPM);
            // $scope.$apply();
        }, function (response) {
            $scope.rawSettings = "error setting settings";
            for (i = 0; i < toggles.length; i++) {
                settings[toggles[i]] = false;
            }

        });
    }

    getSettings();

    $scope.$watchGroup(toggles, function (newValues, oldValues, scope) {
        var newsettings = {}
        var dirty = false;
        for (i = 0; i < newValues.length; i++) {
            if ((newValues[i] !== undefined) && (settings[toggles[i]] !== undefined)) {
                if (newValues[i] !== settings[toggles[i]]) {
                    settings[toggles[i]] = newValues[i];
                    newsettings[toggles[i]] = newValues[i];
                    dirty = true;
                };
            }
        }
        if (dirty) {
            console.log(angular.toJson(newsettings));
            setSettings(angular.toJson(newsettings));
        }
    });

    $scope.updateppm = function() {
        if (($scope.PPM !== undefined) && ($scope.PPM !== null) && $scope.PPM !== settings["PPM"]) {
            settings["PPM"] = $scope.PPM;
            newsettings = {
                "PPM": $scope.PPM
            };
            console.log(angular.toJson(newsettings));
            setSettings(angular.toJson(newsettings));
        }
    };
};