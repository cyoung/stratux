var socket = io.connect('http://192.168.10.1:9110');

function setConnectedClass (cssClass) {
  $('#connectedLabel').removeClass('label-success');
  $('#connectedLabel').removeClass('label-warning');
  $('#connectedLabel').removeClass('label-danger');
  $('#connectedLabel').addClass( cssClass );

  if(cssClass == 'label-success;')
    $('#connectedLabel').text('Connected');
  else
    $('#connectedLabel').text('Disconnected');
}

socket.on('connect', function () {
  setConnectedClass('label-success');
});

socket.on('reconnect', function () {
  console.log('System', 'Reconnected to the server');
  setConnectedClass('label-success');
});

socket.on('reconnecting', function () {
  console.log('System', 'Attempting to re-connect to the server');
  setConnectedClass('label-danger');
});

socket.on('error', function (e) {
  console.log('System', e ? e : 'A unknown error occurred');
  setConnectedClass('label-danger');
});

socket.on('status', function (msg) {
  console.log('Received status update.')
});

socket.on('configuration', function (msg) {
  console.log('Received configuration update.')
});

$(document).ready(function() { 
  $('input[name=UAT_Enabled]').click(function () {
    console.log('UAT_Enabled clicked');
    socket.emit('UATconfigurationChange', $('input[name=UAT_Enabled]').val());
  });

  $('input[name=ES_Enabled]').click(function () {
    console.log('ES_Enabled clicked');
    socket.emit('ESconfigurationChange', $('input[name=ES_Enabled]').val());
  });

  $('input[name=GPS_Enabled]').click(function () {
    console.log('GPS_Enabled clicked');
    socket.emit('GPSconfigurationChange', $('input[name=GPS_Enabled]').val());
  });

  $('input[name=AHRS_Enabled]').click(function () {
    console.log('AHRS_Enabled clicked');
    socket.emit('AHRSconfigurationChange', $('input[name=AHRS_Enabled]').val());
  });

});
