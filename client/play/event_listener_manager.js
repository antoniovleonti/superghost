"use strict"

class EventListenerManager {
  eventListeners_;

  constructor() {
    this.eventListeners_ = [];
  }

  push(target, type, eventListener) {
    this.eventListeners_.push({
      target: target,
      type: type,
      eventListener: eventListener,
    });
    target.addEventListener(type, eventListener);
  }

  clear() {
    for (const el of this.eventListeners_) {
      el.target.removeEventListener(el.type, el.eventListener);
    }
    this.eventListeners_ = [];
  }
}
