var socket;

function setConnectedClass (cssClass) {
  $('#connectedLabel').removeClass('label-success');
  $('#connectedLabel').removeClass('label-warning');
  $('#connectedLabel').removeClass('label-danger');
  $('#connectedLabel').addClass( cssClass );

  if(cssClass == 'label-success')
    $('#connectedLabel').text('Connected');
  else
    $('#connectedLabel').text('Disconnected');
}

function connect() {
  socket = new WebSocket('ws://' + window.location.hostname + ':9110/');

  socket.onopen = function(msg) {
    setConnectedClass('label-success');
  };

  socket.onclose = function(msg)  {
    setConnectedClass('label-danger');
    setTimeout(connect,1000);
  };

  socket.onerror = function(msg)  {
    console.log('System', e ? e : 'A unknown error occurred');
    setConnectedClass('label-danger');
    setTimeout(connect,1000);
  };

  socket.onmessage = function(msg) {
    console.log('Received status update.')

    var status = JSON.parse(msg.data)

    $('#Version').text(status.Version);
    $('#Devices').text(status.Devices);
    $('#Connected_Users').text(status.Connected_Users);
    $('#UAT_messages_last_minute').text(status.UAT_messages_last_minute);
    $('#UAT_messages_max').text(status.UAT_messages_max);
    $('#ES_messages_last_minute').text(status.UAT_messages_last_minute);
    $('#ES_messages_max').text(status.UAT_messages_max);
    $('#GPS_satellites_locked').text(status.GPS_satellites_locked);
    $('#RY835AI_connected').text(status.RY835AI_connected);
    $('#Uptime').text(status.Uptime);
  };
}

$(document).ready(function() {
  connect();

  $('input[name=UAT_Enabled]').click(function () {
    console.log('UAT_Enabled clicked');

    msg = {setting: 'UAT_Enabled', state: $('input[name=UAT_Enabled]').checked };
    socket.send(JSON.stringify(msg));
  });

  $('input[name=ES_Enabled]').click(function () {
    console.log('ES_Enabled clicked');

    msg = {setting: 'ES_Enabled', state: $('input[name=ES_Enabled]').checked };
    socket.send(JSON.stringify(msg));
  });

  $('input[name=GPS_Enabled]').click(function () {
    console.log('GPS_Enabled clicked');

    msg = {setting: 'GPS_Enabled', state: $('input[name=GPS_Enabled]').checked };
    socket.send(JSON.stringify(msg));
  });

  $('input[name=AHRS_Enabled]').click(function () {
    console.log('AHRS_Enabled clicked');

    msg = {setting: 'AHRS_Enabled', state: $('input[name=AHRS_Enabled]').checked };
    socket.send(JSON.stringify(msg));
  });

});
