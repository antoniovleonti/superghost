"use strict";

class ConfigManager {
  span_;
  config_;

  constructor(span, showConfigButton, configDialog) {
    this.span_ = span;
    showConfigButton.addEventListener('click', () => configDialog.showModal());
  }

  config() {
    return this.config_;
  }

  static createConfigValue(str) {
    const span = document.createElement("span");
    span.classList.add("config-value");
    span.appendChild(document.createTextNode(str));
    return span;
  }

  populateDisplay() {
    Client.clearElement(this.span_);

    for (const key in this.config_) {
      const keyWords = key.match(/([A-Z]?[^A-Z]*)/g).slice(0,-1);
      const keyFormatted = keyWords.join(" ");
      this.span_.appendChild(document.createTextNode(keyFormatted + ": "));
      switch (key) {
        // Any specialized formatting goes here
        default:
          this.span_.appendChild(
              ConfigManager.createConfigValue(this.config_[key]));
          this.span_.appendChild(document.createElement("br"));
      }
    }
  }

	async forceGetConfig() {
		while (!this.config_) {
			this.config_ = await fetch(window.location.pathname + '/config')
					.then(response => response.json())
					.catch(error => console.error("Error getting config: " + error));
    }
  }
}
