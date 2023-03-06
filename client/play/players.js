class PlayersManager extends ListManager {
  offerJoinLi_;

  constructor(ol, joinDialog) {
    super(ol);
    this.offerJoinLi_ = PlayersManager.createOfferJoinLi(joinDialog);
  }

  update(players, state, currentPlayerUsername,
         currentPlayerDeadline, myUsername, maxPlayers) {
    Client.clearElement(this.ol_);

    const hostIsMe = players.length > 0 && players[0].Username == myUsername;
    for (const playerObj of players) {
      const isCurrentPlayer = playerObj.Username == currentPlayerUsername;
      const isMe = playerObj.Username == myUsername;
      const deadline = isCurrentPlayer ? currentPlayerDeadline : null;
      // Make the display for this player
      const display = new PlayerDisplay(playerObj, state, isCurrentPlayer, isMe,
                                        hostIsMe, deadline);

      this.ol_.appendChild(display.li());
    }
    // If the player has not joined and there is space, display a button
    // prompting them to join the game
    if (myUsername == null && players.length < maxPlayers) {
      this.ol_.appendChild(this.offerJoinLi_);
    }
  }

  static createOfferJoinLi(joinDialog) {
    const offerJoinLi = document.createElement("li");
    offerJoinLi .classList.add("players-list-item");

    const showJoinFormButton = document.createElement("button");
    showJoinFormButton.classList.add("content-container");
    showJoinFormButton.classList.add("show-join-form-button");
    showJoinFormButton.appendChild(
        document.createTextNode("You are spectating. Click here to join!"));
    // This is only made once -- no event listener manager needed
    showJoinFormButton.addEventListener('click', () => joinDialog.showModal());
    offerJoinLi.appendChild(showJoinFormButton);
    return offerJoinLi;
  }
}

class PlayerDisplay {
  playerObj_;
  state_;
  isCurrentPlayer;
  isMe_;
  hostIsMe_;
  deadline_;
  li_;
  timer_;
  eventListenerManager_;

  constructor(playerObj, state, isCurrentPlayer, isMe, hostIsMe,
              deadline = null) {
    this.playerObj_ = playerObj;
    this.state_ = state;
    this.isCurrentPlayer = isCurrentPlayer;
    this.isMe_ = isMe;
    this.hostIsMe_ = hostIsMe;
    this.deadline_ = deadline;
    this.eventListenerManager_ = new EventListenerManager();

    this.li_ = document.createElement("li");
    this.li_.classList.add("players-list-item");

    // Player menu dialog
    const menu = this.createMenu();
    this.li_.appendChild(menu);

    // Click to show player options
    const playerWrapperButton = document.createElement("button");
    playerWrapperButton.classList.add("show-player-options");
    playerWrapperButton.classList.add("content-container");
    this.eventListenerManager_.push(playerWrapperButton, 'click',
                                    () => menu.showModal());
    this.li_.appendChild(playerWrapperButton);

    // Layout divs
    const leftCol = document.createElement("div");
    leftCol.classList.add("player-left-column");
    playerWrapperButton.appendChild(leftCol);

    const rightCol = document.createElement("div");
    rightCol.classList.add("player-right-column");
    playerWrapperButton.appendChild(rightCol);

    // Content
    const username = document.createElement("div");
    username.appendChild(Client.createUsernameSpan(playerObj.Username));
    if (isMe) {
      username.appendChild(document.createTextNode(" (you)"));
    }
    leftCol.appendChild(username);

    const score = document.createElement("div");
    score.classList.add("player-score");
    score.appendChild(document.createTextNode(
        PlayerDisplay.scoreToString(playerObj.Score)));
    leftCol.appendChild(score);

    if (isCurrentPlayer) {
      this.timer_ = new TickingPlayerTimer(deadline);
      this.li_.dataset.activePlayer = this.state;
    } else {
      this.timer_ = new FixedPlayerTimer(playerObj.TimeRemaining);
    }
    rightCol.appendChild(this.timer_.element());
  }

  createMenu() {
    const menu = document.createElement("dialog");
    const menuHeader = document.createElement("h3");
    menuHeader.appendChild(Client.createUsernameSpan(this.playerObj_.Username));
    if (this.isMe_) {
      menuHeader.appendChild(document.createTextNode(" (you)"));
    }
    menu.appendChild(menuHeader);
    if (this.isMe_) {
      const leaveButton = Client.createStandaloneButton("Leave");
      this.eventListenerManager_.push(leaveButton, 'click',
                                      PlayerDisplay.handleLeave);
      menu.appendChild(leaveButton);
    } else if (this.hostIsMe_) {
      const kickButton = Client.createStandaloneButton("Kick");
      const kickHandler =
          PlayerDisplay.createKickHandler(this.playerObj_.Username);
      this.eventListenerManager_.push(kickButton, 'click', kickHandler);
      menu.appendChild(kickButton);
    }

    const closeMenu = Client.createStandaloneButton("Close");
    this.eventListenerManager_.push(closeMenu, 'click', () => menu.close());
    menu.appendChild(closeMenu);
    return menu;
  }

  static scoreToString(score) {
    let str = "WORDY".slice(0, Math.min(score, 5));
    str += "-----".slice(Math.min(5, score), 5);
    if (score > 5) {  // Add overflow
      str += " +" + (score - 5).toString();
    }
    return str;
  }

  static createKickHandler(username) {
    return function(e) {
      const data = new URLSearchParams({Username: username});
      Client.postDataResetTargetOnSuccess(
          e, window.location.pathname + '/kick', data)
    };
  }

  li() {
    return this.li_;
  }

  static handleLeave(e) {
    fetch(window.location.pathname + '/leave',
          { method: 'POST', redirect: 'follow' })
        .then(response => {
          if (!response.ok) {
            response.text().then(txt => {
              console.error(`Error leaving: ${response.status} ${txt}`);
            });
          }
          if (response.redirected) {
            window.location.href = response.url;
          }
        })
        .catch(err => console.error(err));
  }

  teardown() {
    this.timer_.teardown();
    this.eventListenerManager_.clear();
  }
}

class PlayerTimer {
  element_;

  constructor() {
    this.element_ = document.createElement("div");
    this.element_.classList.add("player-timer");
  }

  element() {
    return this.element_;
  }

  static formatDuration(duration, countdown = false) {
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

  teardown() {}
}

class FixedPlayerTimer extends PlayerTimer {

  constructor(timeRemaining) {
    super();

    const nsPerMs = 1e6;
    const duration = new Date(0);
    duration.setMilliseconds(timeRemaining / nsPerMs);

    this.element_.appendChild(document.createTextNode(
        PlayerTimer.formatDuration(duration)));
  }
}

class TickingPlayerTimer extends PlayerTimer {

  updateInterval_;

  constructor(deadline) {
    super();

    this.setUpdateInterval(deadline);
  }

  setUpdateInterval(deadline) {
    this.updateInterval_ =  setInterval(this.update.bind(this), 10, deadline);
  }

  update(deadline) {
    const duration = new Date(Math.max(0, deadline - new Date()));
    const duration_str = PlayerTimer.formatDuration(duration, true);

    while (this.element_.lastChild) {
      this.element_.removeChild(this.element_.lastChild);
    }
    this.element_.appendChild(document.createTextNode(duration_str));
  }

  teardown() {
    super.teardown();

    clearInterval(this.updateInverval_);
  }
}
