{{define "script"}}

const kMyUsername = "{{.Username}}"

const affixForm = document.getElementById("affixForm")
const suffixText = document.getElementById("suffixText")
const prefixText = document.getElementById("prefixText")
const affixButton = document.getElementById("affixButton")

const rebutForm = document.getElementById("rebutForm")

const affixFieldSet = document.getElementById("affixFieldSet")
const challengeFieldSet = document.getElementById("challengeFieldSet")
const rebutFieldSet = document.getElementById("rebutFieldSet")

const wordSpan = document.getElementById("wordSpan")
const statusSpan = document.getElementById("statusSpan")
const lastRoundResultSpan = document.getElementById("lastRoundResultSpan")

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

function renderStatus(mode, lastPlayerUsername, nextPlayerUsername) {
  let n, l
  switch (mode) {
    case "edit":
      n = nextPlayerUsername == kMyUsername ? "your" : nextPlayerUsername + "'s"
      statusSpan.innerHTML = "It is " + n + " turn."
      break
    case "rebut":
      n = nextPlayerUsername == kMyUsername ? "you" : nextPlayerUsername
      l = lastPlayerUsername == kMyUsername ? "You" : lastPlayerUsername
      statusSpan.innerHTML = l + " challenged " + n + "."
      break
    case "insufficient players":
      statusSpan.innerHTML = "Waiting for players."
      break
    default:
      break
  }
}

function renderLastRoundResult(lastRoundResult) {
  lastRoundResultSpan.innerHTML = lastRoundResult
}

// TODO: clean this up
function renderEverything(gameState) {
  renderPlayers(gameState.players, gameState.nextPlayer)
  renderWord(gameState.mode, gameState.word,
             gameState.players[gameState.nextPlayer].username)
  renderStatus(gameState.mode, gameState.lastPlayer,
               gameState.players[gameState.nextPlayer].username)
  renderLastRoundResult(gameState.lastRoundResult)

  if (gameState.players[gameState.nextPlayer].username == kMyUsername) {
    switch (gameState.mode) {
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
    node.classList.add("playerListItem")
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

function renderWord(mode, word, nextPlayerUsername) {
  wordSpan.innerHTML = word
  switch (mode) {
    case "edit":
      if (nextPlayerUsername != kMyUsername) {
        prefixText.style.visibility = "hidden"
        suffixText.style.visibility = "hidden"
        suffixText.style.display = "inline"
        affixButton.style.visibility = "hidden"
      } else {
        prefixText.style.visibility = "visible"
        suffixText.style.visibility = "visible"
        suffixText.style.display = word.length > 0 ? "inline" : "none"
        affixButton.style.visibility = "visible"
      }
      break
    default:
      prefixText.style.visibility = "hidden"
      suffixText.style.visibility = "hidden"
      suffixText.style.display = "inline"
      affixButton.style.visibility = "hidden"
  }
}

getCurrentGameState()
longPollNextGameState()

{{end}}
