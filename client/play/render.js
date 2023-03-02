"use strict";

showConfigButton.addEventListener("click", function(e) {
  configDialog.showModal();
});

function scoreToString(score) {
  let str = "WORDY".slice(0, Math.min(score, 5));
  str += "-----".slice(Math.min(5, score), 5);
  if (score > 5) {  // Add overflow
    str += " +" + (score - 5).toString();
  }
  return str;
}

function autoUpdateTimer(timer, deadline) {
  setInterval(function () {
    const duration = new Date(Math.max(0, deadline - new Date()));
    const duration_str = formatDuration(duration, /*countdown = */true);
    timer.innerHTML = "";
    timer.appendChild(document.createTextNode(duration_str));
  }, 1);
}

function formatDuration(duration, countdown = false) {
  const msPerSec = 1e3;
  const msPerMin = 6e4;
  const msPerCsec = 10;

  if (duration.getTime() == 0) {
    return "**:**:**";
  }

  const minutes = duration.getMinutes();
  const seconds = duration.getSeconds();
  const centiseconds = Math.floor(duration.getMilliseconds() / msPerCsec);

  const minutesPadded = minutes.toString().padStart(2, '0');
  const secondsPadded = seconds.toString().padStart(2, '0');
  const centisecondsPadded = centiseconds.toString().padStart(2, '0');

  if (countdown && centiseconds > 50) {
    return minutesPadded + "." + secondsPadded + "." + centisecondsPadded;
  } else {
    return minutesPadded + ":" + secondsPadded + ":" + centisecondsPadded;
  }
}

function createStandaloneButton(innerText) {
  const button = document.createElement("button");
  button.classList.add("standalone-button");
  button.appendChild(document.createTextNode(innerText));
  return button;
}

// Functions to update UI
function populatePlayerList(players,
                            state,
                            currentPlayerUsername,
                            currentPlayerDeadline,
                            myUsername,
                            maxPlayers) {
  const nsPerMs = 1e6;

  playersOL.innerHTML = ""; // Clear the list
  for (const p of players) {
    const newLI = document.createElement("li");
    newLI.classList.add("players-list-item");
    playersOL.appendChild(newLI);

    // Player menu dialog
    const menu = document.createElement("dialog");
    const menuHeader = document.createElement("h3");
    menuHeader.appendChild(createUsernameSpan(p.Username));
    if (p.Username == myUsername) {
      menuHeader.appendChild(document.createTextNode(" (you)"));
    }
    menu.appendChild(menuHeader);
    if (p.Username == myUsername) {
      const leaveButton = createStandaloneButton("Leave");
      leaveButton.addEventListener('click', leaveButtonHandler);
      menu.appendChild(leaveButton);
    } else if (players[0].Username == myUsername) {
      const kickButton = createStandaloneButton("Kick");
      kickButton.addEventListener(
          'click', createKickButtonEventListener(p.Username))
      menu.appendChild(kickButton);
    }
    const closeMenu = createStandaloneButton("Close");
    closeMenu.addEventListener("click", function(e) {
      menu.close();
    });
    menu.appendChild(closeMenu);
    newLI.appendChild(menu);

    // Click to show player options
    const newButton = document.createElement("button");
    newButton.classList.add("show-player-options");
    newButton.classList.add("content-container");
    newButton.addEventListener("click", function(e) {
      menu.showModal();
    });
    newLI.appendChild(newButton);

    // Layout divs
    const leftCol = document.createElement("div");
    leftCol.classList.add("player-left-column");
    newButton.appendChild(leftCol);

    const rightCol = document.createElement("div");
    rightCol.classList.add("player-right-column");
    newButton.appendChild(rightCol);

    // Content
    const username = document.createElement("div");
    username.appendChild(createUsernameSpan(p.Username));
    if (p.Username == myUsername) {
      username.appendChild(document.createTextNode(" (you)"));
    }
    leftCol.appendChild(username);

    const score = document.createElement("div");
    score.classList.add("player-score");
    score.appendChild(document.createTextNode(scoreToString(p.Score)));
    leftCol.appendChild(score);

    const duration = new Date(0);
    duration.setMilliseconds(p.TimeRemaining / nsPerMs);
    const timer = document.createElement("div");
    timer.classList.add("player-timer");
    timer.appendChild(document.createTextNode(
        formatDuration(duration)));
    if (currentPlayerUsername == p.Username) {
      autoUpdateTimer(timer, currentPlayerDeadline);
      newLI.dataset.activePlayer = state;
    }
    rightCol.appendChild(timer);
  }
  // If the player has not joined and there is space, display a button
  // prompting them to join the game
  if (myUsername == null && players.length < maxPlayers) {
    const joinLI = document.createElement("li");
    joinLI .classList.add("players-list-item");
    playersOL.appendChild(joinLI);

    const showJoinFormButton = document.createElement("button");
    showJoinFormButton.classList.add("content-container");
    showJoinFormButton.classList.add("show-join-form-button");
    showJoinFormButton.appendChild(
        document.createTextNode("You are spectating. Click here to join!"));
    showJoinFormButton.addEventListener("click", function(e) {
      joinDialog.showModal();
    });
    joinLI.appendChild(showJoinFormButton);
  }
}

function appendToGameLog(logMsgs) {
  let isScrolledToBottom = checkAlreadyScrolledToBottom(logOL);

  for (const msg of logMsgs) {
    let newLI = document.createElement("li");
    logOL.appendChild(logMsgToLI(msg));
  }

  if (isScrolledToBottom) {
    scrollToBottom(logOL);
  }
}

function createUsernameSpan(username) {
  let span = document.createElement("span");
  span.classList.add("username");
  span.appendChild(document.createTextNode(username));
  return span;
}

function createStemSpan(child) {
  let span = document.createElement("span");
  span.classList.add("stem");
  if (typeof child === "string") {
    span.appendChild(document.createTextNode(child));
  }
  else {
    span.appendChild(child);
  }
  return span;
}

function logMsgToLI(msg) {
  // Creates a link to a definition for a particular word.
  function define(word) {
    let link = document.createElement("a");
    link.href = "https://en.wiktionary.org/wiki/" + word.toLowerCase();
    link.setAttribute("target", "_blank");
    link.setAttribute("rel", "noopener noreferrer");
    link.appendChild(document.createTextNode(word));
    return link;
  }
  function valOrEmpty(val) {
    return (typeof val === "undefined") ? "" : val;
  }
  function bold(text) {
    let b = document.createElement("b");
    b.appendChild(document.createTextNode(text));
    return b;
  }

  let txt = document.createElement("li");
  switch (msg.Type) {
    case "Join":
      txt.appendChild(createUsernameSpan(msg.From));
      txt.appendChild(document.createTextNode(" joined the game!"));
      return txt;

    case "Leave":
      txt.appendChild(createUsernameSpan(msg.From));
      txt.appendChild(document.createTextNode(" left the game."));
      return txt;

    case "ChallengeIsWord":
      txt.appendChild(createUsernameSpan(msg.From));
      txt.appendChild(document.createTextNode(" claimed "));
      txt.appendChild(createUsernameSpan(msg.To));
      txt.appendChild(document.createTextNode(" spelled a valid word."));
      return txt;

    case "ChallengeContinuation":
      txt.appendChild(createUsernameSpan(msg.From));
      txt.appendChild(document.createTextNode(" challenged "));
      txt.appendChild(createUsernameSpan(msg.To));
      txt.appendChild(document.createTextNode(" for a continuation."));
      return txt;

// {"Type":"ChallengeResult","To":"Antonio","Stem":"EXHIBIT","Success":true}
    case "ChallengeResult":
      if (msg.Success) {
        txt.appendChild(createStemSpan(define(msg.Stem)));
        txt.appendChild(bold(" is"));
      }
      else {
        txt.appendChild(createStemSpan(document.createTextNode(msg.Stem)));
        txt.appendChild(document.createTextNode(" is"));
        txt.appendChild(bold(" not"));
      }
      txt.appendChild(document.createTextNode(" a word! +1 "));
      txt.appendChild(createUsernameSpan(msg.To));
      txt.appendChild(document.createTextNode("."));
      return txt;

    case "ChallengedPlayerLeft":
      txt.appendChild(createUsernameSpan(msg.From));
      txt.appendChild(document.createTextNode(" challenged "));
      txt.appendChild(createUsernameSpan(msg.To));
      txt.appendChild(document.createTextNode(", who left the game."));
      return txt;

    case "Rebut":
      txt.appendChild(createUsernameSpan(msg.From));
      txt.appendChild(document.createTextNode(" rebutted with "));
      txt.appendChild(createStemSpan(bold(valOrEmpty(msg.Prefix))));
      txt.appendChild(createStemSpan(msg.Stem));
      txt.appendChild(createStemSpan(bold(valOrEmpty(msg.Suffix))));
      txt.appendChild(document.createTextNode("."));
      return txt;

    case "Affix":
      txt.appendChild(createUsernameSpan(msg.From));
      txt.appendChild(document.createTextNode(": "));
      txt.appendChild(createStemSpan(bold(valOrEmpty(msg.Prefix))));
      txt.appendChild(createStemSpan(valOrEmpty(msg.Stem)));
      txt.appendChild(createStemSpan(bold(valOrEmpty(msg.Suffix))));
      return txt;

    case "Concede":
      txt.appendChild(createUsernameSpan(msg.From));
      txt.appendChild(document.createTextNode(" conceded the round. +1 "));
      txt.appendChild(createUsernameSpan(msg.From));
      txt.appendChild(document.createTextNode("."));
      return txt;

    case "Eliminated":
      txt.appendChild(createUsernameSpan(msg.From));
      txt.appendChild(
          document.createTextNode(" was eliminated from the game!"));
      return txt;

    case "Kick":
      txt.appendChild(createUsernameSpan(msg.From));
      txt.appendChild(document.createTextNode(" kicked "));
      txt.appendChild(createUsernameSpan(msg.To));
      txt.appendChild(document.createTextNode(" from the game."));
      return txt;

    case "GameOver":
      txt.appendChild(document.createTextNode("Game over! "));
      txt.appendChild(createUsernameSpan(msg.To));
      txt.appendChild(
          document.createTextNode(" won the game! Ready up to play another."));
      return txt;

    case "GameStart":
      txt.appendChild(
          document.createTextNode("All players ready! Starting game!"));
      return txt;

    case "Timeout":
      txt.appendChild(createUsernameSpan(msg.From));
      txt.appendChild(document.createTextNode(" ran out of time. +1 "));
      txt.appendChild(createUsernameSpan(msg.From));
      txt.appendChild(document.createTextNode("."));
      return txt;

    case "InsufficientPlayers":
      txt.appendChild(document.createTextNode(
          "There aren't enough players continue play."));
      return txt;

    case "ReadyUp":
      txt.appendChild(createUsernameSpan(msg.From));
      txt.appendChild(document.createTextNode(" is ready."));
      return txt;
  }
}

function appendToChat(chatMsg) {
  let isScrolledToBottom = checkAlreadyScrolledToBottom(chatOL);

  let newLI = document.createElement("li");
  chatOL.appendChild(newLI);

  newLI.appendChild(createUsernameSpan(chatMsg.Sender));

  let content = document.createTextNode(": " + chatMsg.Content);
  newLI.appendChild(content);

  if (isScrolledToBottom) {
    scrollToBottom(chatOL);
  }
}

function updateButtons(room, myUsername) {
  // Hide one, show the other
  const isMyTurn = (room.CurrentPlayerUsername == myUsername);
  mainPanel.dataset.isMyTurn = isMyTurn;
  for (const el of onlyEnabledOnMyTurn) {
    el.disabled = !isMyTurn
  }
  mainPanel.dataset.state = room.State;
}

function createPlayersOrYourSpan(player, isMe) {
  const el = document.createElement("span");
  if (isMe) {
    el.appendChild(document.createTextNode("your"));
  } else {
    el.appendChild(createUsernameSpan(player));
    el.appendChild(document.createTextNode("'s"));
  }
  return el;
}

function writeShortStatus(nextPlayer, lastPlayer, state, myUsername) {
  const nextOrYour = createPlayersOrYourSpan(nextPlayer,
                                             nextPlayer == myUsername);
  const lastOrYour = createPlayersOrYourSpan(lastPlayer,
                                             lastPlayer == myUsername);

  shortStatus.innerHTML = "";  // Clear
  switch (state) {
    case "edit":
      shortStatus.appendChild(document.createTextNode("It is "));
      shortStatus.appendChild(nextOrYour);
      shortStatus.appendChild(
          document.createTextNode(" turn to add a letter."));
      break;
    case "rebut":
      shortStatus.appendChild(document.createTextNode("It is "));
      shortStatus.appendChild(nextOrYour);
      shortStatus.appendChild(
          document.createTextNode(" turn to rebut "));
      shortStatus.appendChild(lastOrYour);
      shortStatus.appendChild(document.createTextNode(" challenge."));
      break;
    case "waiting to start":
      shortStatus.appendChild(
          document.createTextNode("Waiting for 2+ players."));
      break;
    default:  // (Indicative of a bug)
      shortStatus.appendChild(document.createTextNode("? (Not implemented)"));
  }
}

function writeStem(stem) {
  for (const s of stemSpans) {
    s.innerHTML = "";
    s.appendChild(document.createTextNode(stem));
  }
}

function renderGameState(room, config, myUsername) {
  const deadline = Date.parse(room.CurrentPlayerDeadline)
  resetGameForms();
  updateButtons(room, myUsername);
  writeStem(room.Stem);
  populatePlayerList(room.Players, room.State, room.CurrentPlayerUsername,
                     deadline, myUsername, config.MaxPlayers);
  writeShortStatus(room.CurrentPlayerUsername, room.LastPlayerUsername,
                   room.State, myUsername);
  appendToGameLog(room.LogPush);
}

function checkAlreadyScrolledToBottom(el) {
  return el.scrollHeight - el.clientHeight <= el.scrollTop + 1;
}

function scrollToBottom(el) {
  el.scrollTop = el.scrollHeight - el.clientHeight;
}

function createConfigValue(str) {
  const span = document.createElement("span");
  span.classList.add("config-value");
  span.appendChild(document.createTextNode(str));
  return span;
}
function populateConfig(config) {
  configSpan.innerHTML = "";
  for (const key in config) {
    const keyWords = key.match(/([A-Z]?[^A-Z]*)/g).slice(0,-1);
    const keyFormatted = keyWords.join(" ");
    configSpan.appendChild(
        document.createTextNode(keyFormatted + ": "));
    switch (key) {
      // Any specialized formatting goes here
      default:
        configSpan.appendChild(createConfigValue(config[key]));
        configSpan.appendChild(document.createElement("br"));
    }
  }
}

function resetGameForms() {
  affixForm.reset();
  rebutForm.reset();
}

function renderJoinErr(err) {
  joinErr.innerHTML = "";
  joinErr.appendChild(document.createTextNode(err));
}

