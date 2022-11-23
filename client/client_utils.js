{{define "ClientUtils"}}

function sendHTTPRequest(method, pathFn, getDataFn=null, onloadFn=null) {
  let xhr = new XMLHttpRequest();
  xhr.onload = function() {
    if (onloadFn != null) {
      onloadFn(xhr);
    } else {
      getDefaultOnload(null)(xhr);
    }
  }
  let path = pathFn();
  console.log(path);
  xhr.open(method, path);
  data = getDataFn == null ? null : getDataFn();
  console.log([method, path, data].join(" "));
  xhr.send(data);
}

function getHTTPRequester(method, path, form, onloadFn) {
  return function(e) {
    e.preventDefault() // do not redirect
    sendHTTPRequest(method, path, form, onloadFn);
  }
}

function getDefaultOnload(form = null, errorFn = (err)=>console.error(err)) {
  return function(xhr) {
    switch (xhr.status) {
      case 200:
        if (form != null) form.reset();
        break;

      // Redirections
      case 302:
      case 303:
        location.href = xhr.responseText;  // Redirect
        break;

      default:
        errorFn(xhr.responseText);
        break;
    }
  }
}

{{end}}
