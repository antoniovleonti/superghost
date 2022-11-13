{{define "ClientUtils"}}

function sendRequest(method, path, getDataFn=null, onloadFn=null) {
  let xhr = new XMLHttpRequest();
  xhr.onload = function() {
    if (onloadFn != null) {
      onloadFn(xhr);
    } else {
      getDefaultOnload(null)(xhr);
    }
  }
  xhr.open(method, path);
  data = getDataFn == null ? null : getDataFn();
  console.log([method, path, data].join(" "));
  xhr.send(data);
}

function getDefaultListener(method, path, form, onloadFn) {
  return function(e) {
    e.preventDefault() // do not redirect
    sendRequest(method, path, form, onloadFn);
  }
}

function getDefaultOnload(form) {
  return function(xhr) {
    if (xhr.status != 200) {
      console.log("Err: '" + xhr.responseText + "'.");
      return;
    }
    if (form != null) {
      form.reset();
    }
  }
}

{{end}}
