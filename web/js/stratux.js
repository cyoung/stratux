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
        $('#UAT_products_last_minute').text(JSON.stringify(status.UAT_products_last_minute));
        $('#UAT_messages_max').text(status.UAT_messages_max);
        $('#ES_messages_last_minute').text(status.ES_messages_last_minute);
        $('#ES_messages_max').text(status.ES_messages_max);
        $('#GPS_satellites_locked').text(status.GPS_satellites_locked);
        setLEDstatus($('#RY835AI_connected'), status.RY835AI_connected);

        //process products
        var products = status.UAT_products_last_minute;
        for (var product_name in products) {
            if (products.hasOwnProperty(product_name)) {
                var row_for_product = $('#product_rows div.row[product="' + product_name + '"]');
                if (row_for_product.length == 0) {
                    $('#product_rows').append('<div class="row" product="' + product_name + '"><span class="col-xs-1"></span><label class="col-xs-5">' + product_name + '</label><span class="col-xs-3 text-right product-count"></span><span class="col-xs-3"></span></div>')
                    row_for_product = $('#product_rows div.row[product="' + product_name + '"]');
                } 
                $(row_for_product).find('.product-count').html(products[product_name]);
                $('#uat_products').show();
            }
        }
        
        /* the formatting code could move to the other end of the socket */
        var uptime = status.Uptime;
        if (uptime != undefined) {
            var up_s = parseInt((uptime/1000)%60), 
                up_m = parseInt((uptime/(1000*60))%60),
                up_h = parseInt((uptime/(1000*60*60))%24);
            $('#Uptime').text(((up_h<10)?"0"+up_h:up_h) + "h" + ((up_m<10)?"0"+up_m:up_m) + "m" + ((up_s<10)?"0"+up_s:up_s) + "s");
        } else {
            // $('#Uptime').text('unavailable');
        }
        var boardtemp = status.CPUTemp;
        if (boardtemp != undefined) {
            /* boardtemp is celcius to tenths */
            $('#CPUTemp').text(boardtemp.toFixed(1) + 'C / ' + ((boardtemp*9/5)+32.0).toFixed(1) + 'F');
        } else {
            // $('#CPUTemp').text('unavailable');
        }

        // Update Settings
        $('input[name=UAT_Enabled]').prop('checked', status.UAT_Enabled);
        $('input[name=ES_Enabled]').prop('checked', status.ES_Enabled);
        $('input[name=GPS_Enabled]').prop('checked', status.GPS_Enabled);
        $('input[name=AHRS_Enabled]').prop('checked', status.AHRS_Enabled);
        $('input[name=DspTrafficSrc]').prop('checked', status.DEBUG);
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

    $('input[name=DspTrafficSrc]').click(function () {
        console.log('DspTrafficSrc clicked');

        msg = {
            setting: 'DEBUG',
            state: $('input[name=DspTrafficSrc]').prop('checked')
        };
        socket.send(JSON.stringify(msg));
    });

});