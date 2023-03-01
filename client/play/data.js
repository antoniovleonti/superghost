"use strict";

function getUsernameFromCookie() {
  console.log(document.cookie);
  const cookieKVPairs = document.cookie.split(";");
  const usernameToID = cookieKVPairs[0].split("=");
  return usernameToID[0];
}

// Will ALWAYS return a config. Never fails.
async function getConfig() {
  let config;
  while (!config) {
    config = await fetch(window.location.pathname + '/config')
        .then(response => response.json())
        .catch(error => {
          console.error("Error getting config: " + error); });
  }
  return config;
}

async function getCurrentGameState() {
	return fetch(window.location.pathname + '/current-state')
      .then(response => response.json())
      .catch(error => console.error(error));
}

async function getNextGameState() {
	return fetch(window.location.pathname + '/next-state')
      .then(response => {
        if (response.ok) {
          return response.json();
        }
        response.text().then(txt => {
            throw new Error(`(${response.status}) ${txt}`); });
      });
}

async function getNextChat() {
	return fetch(window.location.pathname + '/next-chat')
      .then(response => {
        if (response.ok) {
          return response.json();
        }
        response.text().then(txt => {
            throw new Error(`(${response.status}) ${txt}`); });
      })
}

async function cancelLeave() {
	return fetch(window.location.pathname + '/cancel-leave', { method: 'POST' })
      .catch(error => console.error(error));
}

function leaveMaybe() {
  navigator.sendBeacon(window.location.pathname + "/cancellable-leave");
}
window.addEventListener("pagehide", (e) => leaveMaybe())

affixForm.addEventListener('submit', e => {
  e.preventDefault();
  const data = new URLSearchParams(new FormData(affixForm));

  fetch(window.location.pathname + '/affix', { method: 'POST', body: data, })
      .then(response => {
        if (response.ok) {
          return response;
        } else {
          console.error(response.text());
        }
      })
      .catch(error => console.error(error));
});

rebutForm.addEventListener('submit', e => {
  e.preventDefault();
  const data = new URLSearchParams(new FormData(rebutForm));

  fetch(window.location.pathname + '/rebuttal', { method: 'POST', body: data, })
      .then(response => {
        if (response.ok) {
          return response;
        } else {
          console.error(response.text());
        }
      })
      .catch(error => console.error(error));
});

joinForm.addEventListener('submit', e => {
  e.preventDefault();
  const data = new URLSearchParams(new FormData(joinForm));

  fetch(window.location.pathname + '/join', {
          method: 'POST',
          body: data,
          redirect: 'follow',
        })
			.then(response => {
				if (response.ok) {
					window.location.reload();
				} else {
          response.text().then( txt => {
            renderJoinErr(`(${response.status}) ${txt}`);
          })
        }
      })
      .catch(error => renderJoinErr(error));
});

challengeContButton.addEventListener('click', e => {
  fetch(window.location.pathname + '/challenge-continuation',
        { method: 'POST' })
      .then(response => {
        if (response.ok) {
          return response;
        } else {
          console.error(response.text());
        }
      })
      .catch(error => console.error(error));
});

challengeIsWordButton.addEventListener('click', e => {
  fetch(window.location.pathname + '/challenge-is-word',
        { method: 'POST' })
      .then(response => {
        if (response.ok) {
          return response;
        } else {
          console.error(response.text());
        }
      })
      .catch(error => console.error(error));
});


concedeButton.addEventListener('click', e => {
  fetch(window.location.pathname + '/concession',
        { method: 'POST' })
      .then(response => {
        if (response.ok) {
          return response;
        } else {
          console.error(response.text());
        }
      })
      .catch(error => console.error(error));
});

chatForm.addEventListener('submit', e => {
  e.preventDefault();
  const data = new URLSearchParams(new FormData(chatForm));
  console.log(data.toString());

  fetch(window.location.pathname + '/chat', { method: 'POST', body: data, })
      .then(response => {
        if (response.ok) {
          chatForm.reset();
          return response;
        } else {
          console.error(response.text());
        }
      })
      .catch(error => console.error(error));
});

chatText.addEventListener("keydown", e => {
	if (event.which === 13 && !event.shiftKey) {
		if (!event.repeat) {
			const newEvent = new Event("submit", {cancelable: true});
			event.target.form.dispatchEvent(newEvent);
		}

		e.preventDefault(); // Prevents the addition of a new line in the text field
	}
});
