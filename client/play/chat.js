"use strict";

class ChatManager extends ListManager {
  constructor(ol, form, textarea) {
    super(ol);

    form.addEventListener('submit', ChatManager.handleSendChat);
    textarea.addEventListener("keydown", ChatManager.handleChatKeydown);
  }

  append(msg) {
    const isScrolledToBottom = this.checkScrolledToBottom();

    const newLI = document.createElement("li");
    this.ol_.appendChild(newLI);

    newLI.appendChild(Client.createUsernameSpan(msg.Sender));

    let content = document.createTextNode(": " + msg.Content);
    newLI.appendChild(content);

    if (isScrolledToBottom) {
      this.scrollToBottom();
    }
  }

  async subscribeToChat() {
    while (true) {
      try {
        const message = await ChatManager.getNextChat();
        this.append(message);
      } catch (err) {
        console.error(err);
      }
    }
  }

  // Returns chat message object or error
  static async getNextChat() {
     return fetch(window.location.pathname + '/next-chat')
         .then(response => {
           if (!response.ok) {
             response.text().then(t => {
               throw new Error(t);
             });
           }
           return response.json();
         });
  }

  static handleChatKeydown(e) {
    if (event.which === 13 && !event.shiftKey) {
      if (!event.repeat) {
        const newEvent = new Event("submit", {cancelable: true});
        event.target.form.dispatchEvent(newEvent);
      }

      e.preventDefault(); // Prevents the addition of a new line in the text field
    }
  }

  static handleSendChat(e) {
    e.preventDefault();
    const data = new URLSearchParams(new FormData(e.target));
    Client.postDataResetTargetOnSuccess(
        e, window.location.pathname + '/chat', data)
  }
}

