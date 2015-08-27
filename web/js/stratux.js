// socket.io specific code
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
  message('System', 'Reconnected to the server');
  setConnectedClass('label-success');
});

socket.on('reconnecting', function () {
  message('System', 'Attempting to re-connect to the server');
  setConnectedClass('label-danger');
});

socket.on('error', function (e) {
  message('System', e ? e : 'A unknown error occurred');
  setConnectedClass('label-danger');
});

function message (from, msg) {
  $('#lines').append($('<p>').append($('<b>').text(from), msg));
}


// dom manipulation

$(document).ready(function() { 
  $('input[name=UAT_Enabled]').change(function(){
    $('#settings').ajaxSubmit({url: 'control.php', type: 'post'})
  });
  $('input[name=ES_Enabled]').change(function(){
    $('#settings').ajaxSubmit({url: 'control.php', type: 'post'})
  });
  $('input[name=GPS_Enabled]').change(function(){
    $('#settings').ajaxSubmit({url: 'control.php', type: 'post'})
  });
  $('input[name=AHRS_Enabled]').change(function(){
    $('#settings').ajaxSubmit({url: 'control.php', type: 'post'})
  });
});
(function worker() {
  $.ajax({
    url: 'control.php', 
    success: function(data) {
      obj = $.parseJSON(data);
      $.each(obj, function(k, v) {
        // Radio values.
        if ((k == "UAT_Enabled") || (k == "ES_Enabled") || (k == "GPS_Enabled") || (k == "AHRS_Enabled")) {
          $('[name=' + k + ']').val([v.toString()]);
        }
        $('#' + k).text(v);
      });

    },
    complete: function() {
      // Schedule the next request when the current one is complete.
      setTimeout(worker, 1000);
    }
  });
})();