"use strict";

class ListManager {
  ol_;

  constructor(ol) {
    this.ol_ = ol;
  }

  checkScrolledToBottom() {
    return this.ol_.scrollHeight - this.ol_.clientHeight <=
           this.ol_.scrollTop + 1;
  }

  scrollToBottom() {
    this.ol_.scrollTop = this.ol_.scrollHeight - this.ol_.clientHeight;
  }
}
