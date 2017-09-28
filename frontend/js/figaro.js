function send_ok(channel_id, value){
    console.log("button clicked");
    console.log("channel: " + channel_id)
    console.log("ok: " + value)
    $.post( "backend/change_status/",{ Ok: value, ID: channel_id },function(json) {
         console.log("ok is sent");
    })
}

Handlebars.registerHelper('list', function(items, options) {
  var out = "";
  if (items === null) {
    return ""
  }
  for(var i=0, l=items.length; i<l; i++) {
    out += options.fn(items[i]);
  }

  return out;
});

Handlebars.registerHelper('parseTime', function(t) {
  d = new Date(Date.parse(t))
  return moment(d, "minute").fromNow();
});

$(function () {
// Grab the template script
var theTemplateScript = $("#channel-template").html();

// Compile the template
var theTemplate = Handlebars.compile(theTemplateScript);

var socket = new WebSocket("ws://localhost:8080");
console.log("Connected")
socket.onmessage = function (event) {
  var data = JSON.parse(event.data)
  console.log(data)
  var bad = {
    "channels": data.Bad
  }
  var ok = {
    "channels": data.Ok
  }
  // Pass our data to the template
  var compiledHtmlOk = theTemplate(bad);
  var compiledHtmlBad = theTemplate(ok);

  // Add the compiled html to the page
  $('.channels-up').html(compiledHtmlOk);
  $('.channels-down').html(compiledHtmlBad);
}
});
