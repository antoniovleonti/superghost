class GameLogManager extends ListManager {
  push(msgs) {
    let isScrolledToBottom = this.checkScrolledToBottom(this.logOL_);

    for (const msg of msgs) {
      let newLI = document.createElement("li");
      this.ol_.appendChild(this.logMsgToLI(msg));
    }

    if (isScrolledToBottom) {
      this.scrollToBottom(this.logOL_);
    }
  }

  logMsgToLI(msg) {
    // Creates a link to a definition for a particular word.
    function define(word) {
      const link = document.createElement("a");
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
      const b = document.createElement("b");
      b.appendChild(document.createTextNode(text));
      return b;
    }

    const txt = document.createElement("li");
    switch (msg.Type) {
      case "Join":
        txt.appendChild(Client.createUsernameSpan(msg.From));
        txt.appendChild(document.createTextNode(" joined the game!"));
        return txt;

      case "Leave":
        txt.appendChild(Client.createUsernameSpan(msg.From));
        txt.appendChild(document.createTextNode(" left the game."));
        return txt;

      case "ChallengeIsWord":
        txt.appendChild(Client.createUsernameSpan(msg.From));
        txt.appendChild(document.createTextNode(" claimed "));
        txt.appendChild(Client.createUsernameSpan(msg.To));
        txt.appendChild(document.createTextNode(" spelled a valid word."));
        return txt;

      case "ChallengeContinuation":
        txt.appendChild(Client.createUsernameSpan(msg.From));
        txt.appendChild(document.createTextNode(" challenged "));
        txt.appendChild(Client.createUsernameSpan(msg.To));
        txt.appendChild(document.createTextNode(" for a continuation."));
        return txt;

      case "ChallengeResult":
        if (msg.Success) {
          txt.appendChild(Client.createStemSpan(define(msg.Stem)));
          txt.appendChild(bold(" is"));
        }
        else {
          txt.appendChild(Client.createStemSpan(document.createTextNode(msg.Stem)));
          txt.appendChild(document.createTextNode(" is"));
          txt.appendChild(bold(" not"));
        }
        txt.appendChild(document.createTextNode(" a word! +1 "));
        txt.appendChild(Client.createUsernameSpan(msg.To));
        txt.appendChild(document.createTextNode("."));
        return txt;

      case "ChallengedPlayerLeft":
        txt.appendChild(Client.createUsernameSpan(msg.From));
        txt.appendChild(document.createTextNode(" challenged "));
        txt.appendChild(Client.createUsernameSpan(msg.To));
        txt.appendChild(document.createTextNode(", who left the game."));
        return txt;

      case "Rebut":
        txt.appendChild(Client.createUsernameSpan(msg.From));
        txt.appendChild(document.createTextNode(" rebutted with "));
        txt.appendChild(Client.createStemSpan(bold(valOrEmpty(msg.Prefix))));
        txt.appendChild(Client.createStemSpan(msg.Stem));
        txt.appendChild(Client.createStemSpan(bold(valOrEmpty(msg.Suffix))));
        txt.appendChild(document.createTextNode("."));
        return txt;

      case "Affix":
        txt.appendChild(Client.createUsernameSpan(msg.From));
        txt.appendChild(document.createTextNode(": "));
        txt.appendChild(Client.createStemSpan(bold(valOrEmpty(msg.Prefix))));
        txt.appendChild(Client.createStemSpan(valOrEmpty(msg.Stem)));
        txt.appendChild(Client.createStemSpan(bold(valOrEmpty(msg.Suffix))));
        return txt;

      case "Concede":
        txt.appendChild(Client.createUsernameSpan(msg.From));
        txt.appendChild(document.createTextNode(" conceded the round. +1 "));
        txt.appendChild(Client.createUsernameSpan(msg.From));
        txt.appendChild(document.createTextNode("."));
        return txt;

      case "Eliminated":
        txt.appendChild(Client.createUsernameSpan(msg.From));
        txt.appendChild(
            document.createTextNode(" was eliminated from the game!"));
        return txt;

      case "Kick":
        txt.appendChild(Client.createUsernameSpan(msg.From));
        txt.appendChild(document.createTextNode(" kicked "));
        txt.appendChild(Client.createUsernameSpan(msg.To));
        txt.appendChild(document.createTextNode(" from the game."));
        return txt;

      case "GameOver":
        txt.appendChild(document.createTextNode("Game over! "));
        txt.appendChild(Client.createUsernameSpan(msg.To));
        txt.appendChild(document.createTextNode(
            " won the game! Ready up to play another."));
        return txt;

      case "GameStart":
        txt.appendChild(
            document.createTextNode("All players ready! Starting game!"));
        return txt;

      case "Timeout":
        txt.appendChild(Client.createUsernameSpan(msg.From));
        txt.appendChild(document.createTextNode(" ran out of time. +1 "));
        txt.appendChild(Client.createUsernameSpan(msg.From));
        txt.appendChild(document.createTextNode("."));
        return txt;

      case "InsufficientPlayers":
        txt.appendChild(document.createTextNode(
            "There aren't enough players continue play."));
        return txt;

      case "ReadyUp":
        txt.appendChild(Client.createUsernameSpan(msg.From));
        txt.appendChild(document.createTextNode(" is ready."));
        return txt;
    }
  }
}
