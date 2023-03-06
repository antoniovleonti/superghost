"use strict";

class Client {
  playersManager_;
  gameLogManager_;
  chatManager_;
  configManager_;
  mainPanelManager_;
  joinManager_;

  myUsername_;

  constructor() {
    this.playersManager_ = new PlayersManager(
        document.getElementById("players-list"),
        document.getElementById("join-dialog"));
    this.gameLogManager_ =
        new GameLogManager(document.getElementById("game-log-list"));
    this.chatManager_ = new ChatManager(
        document.getElementById("chat-list"),
        document.getElementById("chat-form"),
        document.getElementById("chat-textarea"));
    this.configManager_ = new ConfigManager(
        document.getElementById("config-span"),
        document.getElementById("show-config-button"),
        document.getElementById("config-dialog"));
    this.mainPanelManager_ = new MainPanelManager({
      mainPanel: document.getElementById("main-panel"),
      affixForm: document.getElementById("affix-form"),
      rebutForm: document.getElementById("rebut-form"),
      concedeButton: document.getElementById("concede-button"),
      challengeContinuationButton: document.getElementById("ch-cont-button"),
      challengeIsWordButton: document.getElementById("ch-word-button"),
      shortStatusSpan: document.getElementById("short-status"),
      activeStemSpans: document.getElementsByClassName("active-stem"),
      onlyEnabledOnMyTurn:
          document.getElementsByClassName("only-enabled-on-my-turn")
    });
    this.joinManager_ = new JoinManager(document.getElementById("join-form"),
                                        document.getElementById("join-err"));

    // Misc stuff that needs to happen
    window.addEventListener('pagehide', Client.handlePageHide);
  }

  renderGameState(room) {
    const deadline = Date.parse(room.CurrentPlayerDeadline)
    this.mainPanelManager_.update(room, this.myUsername_);
    this.playersManager_.update(
        room.Players, room.State, room.CurrentPlayerUsername, deadline,
        this.myUsername_, this.configManager_.config().MaxPlayers);
    this.gameLogManager_.push(room.LogPush);
  }

  async subscribeToGameState() {
    while (true) {
      try {
        const gameState = await fetch(window.location.pathname + '/next-state')
            .then(response => {
              if (response.ok) {
                return response.json();
              }
              response.text().then(txt => {throw new Error(txt);});
            });

        console.log(gameState);
        this.renderGameState(gameState);
      } catch (err) {
        console.error(err);
      }
    }
  }

  async enterGameLoop() {
    await this.configManager_.forceGetConfig();
    this.configManager_.populateDisplay();
    // The only case where cancel-leave response is not ok is when the server
    // doesn't recognize the player
    const cancelLeaveResponse = await Client.cancelLeave();
    const hasJoined = cancelLeaveResponse.ok;
    this.myUsername_ = hasJoined ? Client.getUsernameFromCookie() : null;

    // Sync with the server
     await fetch(window.location.pathname + '/current-state')
        .then(response => {
          if (response.ok) {
            return response.json();
          }
          response.text().then(txt => {throw new Error(txt);});
        })
        .then(room => this.renderGameState(room))
        .catch(err => console.error(err));

    // Subscribe to changes
    this.chatManager_.subscribeToChat();
    this.subscribeToGameState();
  }

  /*** Utility functions ***/

  static createStandaloneButton(innerText) {
    const button = document.createElement("button");
    button.classList.add("standalone-button");
    button.appendChild(document.createTextNode(innerText));
    return button;
  }

  static createUsernameSpan(username) {
    const span = document.createElement("span");
    span.classList.add("username");
    span.appendChild(document.createTextNode(username));
    return span;
  }

  static createStemSpan(child) {
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

  static clearElement(el) {
    while (el.lastChild) {
      el.removeChild(el.lastChild);
    }
  }

  static postDataResetTargetOnSuccess(e, path, data) {
    fetch(path, { method: 'POST', body: data })
        .then(response => {
          if (!response.ok) {
            console.error(response.text());
          }
          try {
            e.target.reset();
          } catch (err) {
            // This is ok; this function is used for buttons too.
          }
        })
        .catch(err => console.error(err));
  }

  static async cancelLeave() {
    return fetch(window.location.pathname + '/cancel-leave', { method: 'POST' })
        .catch(err => console.error(err));
  }

  static handlePageHide(e) {
    navigator.sendBeacon(window.location.pathname + "/cancellable-leave");
  }

  static getUsernameFromCookie() {
    const cookieKVPairs = document.cookie.split(";");
    // Always take the last cookie
    const usernameToID = cookieKVPairs[cookieKVPairs.length - 1].split("=");
    const username = usernameToID[0].trim();
    console.log({ rawCookie: document.cookie, derivedUsername: username });
  }
}

