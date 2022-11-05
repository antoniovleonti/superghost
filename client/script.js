{{define "script"}}

const kMyUsername = "{{.Username}}"

const affixForm = document.getElementById("affix-form")
const affixFieldSet = document.getElementById("affixFieldSet")

const rebutForm = document.getElementById("rebut-form")
const concedeButton = document.getElementById("concedeButton")

const challengeFieldSet = document.getElementById("challengeFieldSet")
const rebutFieldSet = document.getElementById("rebutFieldSet")

const wordSpan = document.getElementById("wordSpan")
const statusSpan = document.getElementById("statusSpan")
const lastRoundResultSpan = document.getElementById("lastRoundResultSpan")

const playerList = document.getElementById("playerList")

let leaveButton = document.createElement("button");
leaveButton.innerHTML = "leave"

leaveButton.addEventListener("click", function(e){
  e.preventDefault() // do not write response to screen
  var xhr = new XMLHttpRequest()
	xhr.onload = function() {
    if (xhr.status == 303) { // "see other"
      location.href = xhr.responseText
    }
	}
  xhr.open("POST", "{{.RoomID}}/leave")
  xhr.send()
})

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
  xhr.open("POST", "{{.RoomID}}/affix")
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
  xhr.open("POST", "{{.RoomID}}/rebuttal")
  xhr.send(new URLSearchParams(new FormData(rebutForm)))
})

concedeButton.addEventListener("click", function(e){
  e.preventDefault() // do not redirect
  var xhr = new XMLHttpRequest()
	xhr.onload = function() {
		if (xhr.status != 200) {
			console.log(xhr.responseText)
			return // I should probably do something useful here
		}
	}
  xhr.open("POST", "{{.RoomID}}/concession")
  xhr.send()
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
  xhr.open("POST", "{{.RoomID}}/challenge-is-word")
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
  xhr.open("POST", "{{.RoomID}}/challenge-continuation")
  xhr.send()
})

// setInterval(function() {
  // var xhr = new XMLHttpRequest()
	// xhr.onload = function() {
		// if (xhr.status != 200) {
      // window.location.href = "join"
		// }
	// }
  // xhr.open("POST", "heartbeat?" + new Date().getTime())
  // xhr.send()
// }, 100) // every 100 ms

function longPollNextGameState () {
  var xhr = new XMLHttpRequest()
  xhr.onload = function () {
    var state = JSON.parse(xhr.responseText)
    console.log(state)
    renderEverything(state)
    longPollNextGameState()
  }
  xhr.open("GET", "{{.RoomID}}/next-state")
  xhr.send()
}

function getCurrentGameState () {
  var xhr = new XMLHttpRequest()
  xhr.onload = function () {
    var state = JSON.parse(xhr.responseText)
    console.log(state)
    renderEverything(state)
  }
  xhr.open("GET", "{{.RoomID}}/current-state")
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
      l = lastPlayerUsername == kMyUsername ? "You" : lastPlayerUsername
      n = nextPlayerUsername == kMyUsername ? "you" : nextPlayerUsername
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
function renderEverything(gs) {
  renderPlayers(gs.players, gs.nextPlayer)
  renderForms(gs.awaiting, gs.word, gs.players[gs.nextPlayer].username)
  renderStatus(gs.awaiting, gs.lastPlayer, gs.players[gs.nextPlayer].username)
  renderLastRoundResult(gs.lastRoundResult)
}

function renderPlayers(players, nextPlayer) {
  playerList.innerHTML = "" // clear
  for (const p of players) {
    let isMe = p.username == kMyUsername
    let li = document.createElement("li")
    li.classList.add("playerli")

    let pstr = p.username + " " + p.score
    li.appendChild(document.createTextNode(pstr))

    if (isMe) {
      li.appendChild(leaveButton)
    }
    playerList.appendChild(li)
  }
}

function renderForms(mode, word, nextPlayerUsername) {
  wordSpan.innerHTML = word

  nextPlayerIsMe = nextPlayerUsername == kMyUsername
  affixFieldSet.disabled = !(nextPlayerIsMe && mode == "edit")
  challengeFieldSet.disabled = !(nextPlayerIsMe && mode == "edit")
  rebutFieldSet.disabled = !(nextPlayerIsMe && mode == "rebut")
}

getCurrentGameState()
longPollNextGameState()

{{end}}
