var socket;

function setLEDstatus(ledElement, status) {
    if (status) {
        ledElement.removeClass('led-red');
        ledElement.addClass('led-green');
    } else {
        ledElement.removeClass('led-green');
        ledElement.addClass('led-red');
    }
}

function setConnectedClass(cssClass) {
    $('#connectedLabel').removeClass('label-success');
    $('#connectedLabel').removeClass('label-warning');
    $('#connectedLabel').removeClass('label-danger');
    $('#connectedLabel').addClass(cssClass);

    if (cssClass == 'label-success')
        $('#connectedLabel').text('Connected');
    else
        $('#connectedLabel').text('Disconnected');
}

function connect() {
    socket = new WebSocket('ws://' + window.location.hostname + '/control');

    socket.onopen = function (msg) {
        setConnectedClass('label-success');
    };

    socket.onclose = function (msg) {
        setConnectedClass('label-danger');
        setTimeout(connect, 1000);
    };

    socket.onerror = function (msg) {
        setConnectedClass('label-danger');
    };

    socket.onmessage = function (msg) {
        console.log('Received status update.')

        var status = JSON.parse(msg.data)

        // Update Status
        $('#Version').text(status.Version);
        $('#Devices').text(status.Devices);
        $('#Connected_Users').text(status.Connected_Users);
        $('#UAT_messages_last_minute').text(status.UAT_messages_last_minute);
        $('#UAT_messages_max').text(status.UAT_messages_max);
        $('#ES_messages_last_minute').text(status.ES_messages_last_minute);
        $('#ES_messages_max').text(status.ES_messages_max);
        $('#GPS_satellites_locked').text(status.GPS_satellites_locked);
        setLEDstatus($('#RY835AI_connected'), status.RY835AI_connected);
        
        /* the formatting code could move to the other end of the socket */
        var uptime = status.Uptime;
        if (uptime != undefined) {
            var time_parts = uptime.match(/([0-9]*)h([0-9]*)m([0-9\.]*)s/);
            $('#Uptime').text(time_parts[1] + "h" + time_parts[2] + "m" + Math.round(parseFloat(time_parts[3])) + "s");
        }
        // not yet implemented - showing the raspberry pi board temperature will be helpful when Stratux is contained in a case
        /* $('#PI_Temperature').text(status.Pi_Temperature); */

        // Update Settings
        $('input[name=UAT_Enabled]').prop('checked', status.UAT_Enabled);
        $('input[name=ES_Enabled]').prop('checked', status.ES_Enabled);
        $('input[name=GPS_Enabled]').prop('checked', status.GPS_Enabled);
        $('input[name=AHRS_Enabled]').prop('checked', status.AHRS_Enabled);
    };
}

$(document).ready(function () {
    connect();

    $('input[name=UAT_Enabled]').click(function () {
        console.log('UAT_Enabled clicked');

        msg = {
            setting: 'UAT_Enabled',
            state: $('input[name=UAT_Enabled]').prop('checked')
        };
        socket.send(JSON.stringify(msg));
    });

    $('input[name=ES_Enabled]').click(function () {
        console.log('ES_Enabled clicked');

        msg = {
            setting: 'ES_Enabled',
            state: $('input[name=ES_Enabled]').prop('checked')
        };
        socket.send(JSON.stringify(msg));
    });

    $('input[name=GPS_Enabled]').click(function () {
        console.log('GPS_Enabled clicked');

        msg = {
            setting: 'GPS_Enabled',
            state: $('input[name=GPS_Enabled]').prop('checked')
        };
        socket.send(JSON.stringify(msg));
    });

    $('input[name=AHRS_Enabled]').click(function () {
        console.log('AHRS_Enabled clicked');

        msg = {
            setting: 'AHRS_Enabled',
            state: $('input[name=AHRS_Enabled]').prop('checked')
        };
        socket.send(JSON.stringify(msg));
    });

});