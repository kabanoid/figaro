var socket = io.connect(websocketHost);
socket.on('connect', function (msg) {
    console.log('Connection established.');
});
socket.on('message', function (channels) {
    var content = '', rows = Math.ceil(channels.length / 3);
    for (row = 0; row < rows; row++) {
        content += '<div class="row">'
        for (col = 0; col < 3; col++) {
            if (channels.length > 0) {
                var channel = channels.pop();
                content += '<div class="col-sm-4"><div class="panel panel-default">';
                content += '<div class="panel-heading">' + channel.name + '</div>';
                content += '<div class="panel-body"><div class="list-group">';
                channel.messages.forEach(function (msg) {
                    content += '<a href="#" class="list-group-item">' + msg.author + ' ' + msg.timestamp + '</a>';
                    content += '<a href="#" class="list-group-item">' + msg.text + '</a>';
                });
                content += '</div></div></div></div>';
            }
        }
        content += '</div>'
    }
    $('#body-content').html(content);
});
