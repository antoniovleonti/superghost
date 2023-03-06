"use strict";

class JoinManager {
  joinForm_;
  joinErrorSpan_;

  constructor(joinForm, joinErrorSpan) {
    this.joinForm_ = joinForm;
    this.joinErrorSpan_ = joinErrorSpan;

    this.joinForm_.addEventListener('submit', this.handleJoin.bind(this));
  }

  renderJoinErr(err) {
    Client.clearElement(this.joinErr_);
    this.joinErr_.appendChild(document.createTextNode(err));
  }

  handleJoin(e) {
    e.preventDefault();
    const data = new URLSearchParams(new FormData(e.target));

    fetch(window.location.pathname + '/join', { method: 'POST', body: data, })
        .then(response => {
          if (response.ok) {
            window.location.reload();
          } else {
            response.text().then(txt => {
              this.renderJoinErr(txt);
            });
          }
        })
        .catch(error => this.renderJoinErr(error));

    return false;
  }
}
