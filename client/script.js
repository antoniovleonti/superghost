{{define "script"}}

const kMyUsername = "{{.Username}}"

const affixForm = document.getElementById("affixForm")
const rebutForm = document.getElementById("rebutForm")

const affixFieldSet = document.getElementById("affixFieldSet")
const challengeFieldSet = document.getElementById("challengeFieldSet")
const rebutFieldSet = document.getElementById("rebutFieldSet")

affixForm.addEventListener("submit", function(e){
  e.preventDefault() // do not redirect
  var xhr = new XMLHttpRequest()
	xhr.onload = function() {
		if (xhr.status != 200) {
			console.log(xhr.responseText)
			return // I should probably do something useful here
		}
		affixForm.reset()
	}
  xhr.open("POST", "/word")
  xhr.send(new URLSearchParams(new FormData(affixForm)))
})

rebutForm.addEventListener("submit", function(e){
  e.preventDefault() // do not redirect
  var xhr = new XMLHttpRequest()
	xhr.onload = function() {
		if (xhr.status != 200) {
			console.log(xhr.responseText)
			return // I should probably do something useful here
		}
		rebutForm.reset()
	}
  xhr.open("POST", "/rebuttal")
  xhr.send(new URLSearchParams(new FormData(rebutForm)))
})

isWordButton.addEventListener("click", function(e){
  e.preventDefault() // do not redirect
  var xhr = new XMLHttpRequest()
	xhr.onload = function() {
		if (xhr.status != 200) {
			console.log(xhr.responseText)
			return // I should probably do something useful here
		}
	}
  xhr.open("POST", "/challenge-is-word")
  xhr.send()
})

noContinuationButton.addEventListener("click", function(e){
  e.preventDefault() // do not redirect
  var xhr = new XMLHttpRequest()
	xhr.onload = function() {
		if (xhr.status != 200) {
			return // I should probably do something useful here
		}
	}
  xhr.open("POST", "/challenge-continuation")
  xhr.send()
})

setInterval(function() {
  var xhr = new XMLHttpRequest()
	xhr.onload = function() {
		if (xhr.status != 200) {
      window.location.href = "/join"
		}
	}
  xhr.open("POST", "/heartbeat?" + new Date().getTime())
  xhr.send()
}, 100) // every 100 ms

function longPollNextGameState () {
  var xhr = new XMLHttpRequest()
  xhr.onload = function () {
    var state = JSON.parse(xhr.responseText)
    console.log(state)
    renderEverything(state)
    longPollNextGameState()
  }
  xhr.open("GET", "/next-state")
  xhr.send()
}

function getCurrentGameState () {
  var xhr = new XMLHttpRequest()
  xhr.onload = function () {
    var state = JSON.parse(xhr.responseText)
    console.log(state)
    renderEverything(state)
  }
  xhr.open("GET", "/state")
  xhr.send()
}

// TODO: clean this up
function renderEverything(gameState) {
  renderPlayers(gameState.players, gameState.nextPlayer)
  renderWord(gameState.word)
  if (gameState.players[gameState.nextPlayer].username == kMyUsername) {
    switch (gameState.phase) {
      case "edit":
        enterEditMode()
        break;
      case "rebut":
        enterRebuttalMode()
        break
      case "insufficient players":
        enterReadonlyMode()
        break
    }
  } else {
    enterReadonlyMode()
  }
}

function renderPlayers(players, nextPlayer) {
  playerList = document.getElementById("playerList")
  playerList.innerHTML = "" // clear
  for (let i = 0; i < players.length; i++) {
    var node = document.createElement("li")
    var playerStr = players[i].username + " " + players[i].score

    node.appendChild(document.createTextNode(playerStr))
    playerList.appendChild(node)
  }
}

function enterEditMode() {
  // enable editor
  affixFieldSet.disabled=false
  challengeFieldSet.disabled=false
  rebutFieldSet.disabled=true

  // change visibilities
  rebutForm.style.display="none"
  challengeForm.style.display="block"
}

function enterReadonlyMode() {
  affixFieldSet.disabled=true
  challengeFieldSet.disabled=true
  rebutFieldSet.disabled=true
  rebutForm.style.display="none"
  challengeForm.style.display="none"
}

function enterRebuttalMode() {
  affixFieldSet.disabled=true
  challengeFieldSet.disabled=true
  rebutFieldSet.disabled=false
  rebutForm.style.display="block"
  challengeForm.style.display="none"
}

function renderWord(word) {
  document.getElementById("word").innerHTML = word
}

getCurrentGameState()
longPollNextGameState()

{{end}}
