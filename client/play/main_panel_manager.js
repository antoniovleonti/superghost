"use strict";

class MainPanelManager {
  mainPanel_;
  affixForm_;
  rebutForm_;
  onlyEnabledOnMyTurn_;
  activeStemSpans_;
  shortStatusSpan_;

  // Use an opts struct to make this less error-prone
  constructor(opts) {
    this.mainPanel_ = opts.mainPanel;
    this.affixForm_ = opts.affixForm;
    this.rebutForm_ = opts.rebutForm;
    this.challengeContinuationButton_ = opts.challengeContinuationButton;
    this.challengeIsWordButton_ = opts.challengeIsWordButton;
    this.onlyEnabledOnMyTurn_ = opts.onlyEnabledOnMyTurn;
    this.activeStemSpans_ = opts.activeStemSpans;
    this.shortStatusSpan_ = opts.shortStatusSpan;

    this.affixForm_.addEventListener('submit', this.handleAffix.bind(this));
    this.rebutForm_.addEventListener('submit', this.handleRebut.bind(this));
    opts.challengeContinuationButton.addEventListener(
        'click', this.handleChallengeContinuation.bind(this));
    opts.challengeIsWordButton.addEventListener(
        'click', this.handleChallengeIsWord.bind(this));
    opts.concedeButton.addEventListener('click',
                                         this.handleConcede.bind(this));
  }

  update(room, myUsername) {
    this.resetGameForms();
    this.updateShortStatus(room.CurrentPlayerUsername, room.LastPlayerUsername,
                           room.State, myUsername);
    this.updateButtons(room.State, room.CurrentPlayerUsername, myUsername);
    this.updateActiveStemSpans(room.Stem);
  }

  updateShortStatus(nextPlayer, lastPlayer, state, myUsername) {
    Client.clearElement(this.shortStatusSpan_);

    const nextOrYour = MainPanelManager.createPlayersOrYourSpan(
        nextPlayer, nextPlayer == myUsername);
    const lastOrYour = MainPanelManager.createPlayersOrYourSpan(
        lastPlayer, lastPlayer == myUsername);

    switch (state) {
      case "edit":
        this.shortStatusSpan_.appendChild(document.createTextNode("It is "));
        this.shortStatusSpan_.appendChild(nextOrYour);
        this.shortStatusSpan_.appendChild(
            document.createTextNode(" turn to add a letter."));
        break;
      case "rebut":
        this.shortStatusSpan_.appendChild(document.createTextNode("It is "));
        this.shortStatusSpan_.appendChild(nextOrYour);
        this.shortStatusSpan_.appendChild(
            document.createTextNode(" turn to rebut "));
        this.shortStatusSpan_.appendChild(lastOrYour);
        this.shortStatusSpan_.appendChild(
            document.createTextNode(" challenge."));
        break;
      case "waiting to start":
        this.shortStatusSpan_.appendChild(
            document.createTextNode("Waiting for 2+ players."));
        break;
      default:  // (Indicative of a bug)
        this.shortStatusSpan_.appendChild(document.createTextNode("? (Not implemented)"));
    }
  }

  updateButtons(state, currentPlayerUsername, myUsername) {
    // Hide one, show the other
    const isMyTurn = (currentPlayerUsername == myUsername);
    this.mainPanel_.dataset.isMyTurn = isMyTurn;
    for (const el of this.onlyEnabledOnMyTurn_) {
      el.disabled = !isMyTurn
    }
    this.mainPanel_.dataset.state = state;
  }

  updateActiveStemSpans(stem) {
    for (const s of this.activeStemSpans_) {
      s.innerHTML = "";
      s.appendChild(document.createTextNode(stem));
    }
  }

  resetGameForms() {
    this.affixForm_.reset();
    this.rebutForm_.reset();
  }

  handleAffix(e) {
    e.preventDefault();
    const data = new URLSearchParams(new FormData(e.target));
    Client.postDataResetTargetOnSuccess(
        e, window.location.pathname + '/affix', data)
  }

  handleRebut(e) {
    e.preventDefault();
    const data = new URLSearchParams(new FormData(e.target));
    Client.postDataResetTargetOnSuccess(
        e, window.location.pathname + '/rebuttal', data)
  }

  handleChallengeContinuation(e) {
    Client.postDataResetTargetOnSuccess(
        e, window.location.pathname + '/challenge-continuation', null)
  }

  handleChallengeIsWord(e) {
    Client.postDataResetTargetOnSuccess(
        e, window.location.pathname + '/challenge-is-word', null)
  }

  handleConcede(e) {
    Client.postDataResetTargetOnSuccess(
        e, window.location.pathname + '/concession', null)
  }

  static createPlayersOrYourSpan(player, isMe) {
    const el = document.createElement("span");
    if (isMe) {
      el.appendChild(document.createTextNode("your"));
    } else {
      el.appendChild(Client.createUsernameSpan(player));
      el.appendChild(document.createTextNode("'s"));
    }
    return el;
  }
}

