class StatusManager {
  span_;

  constructor(span) {
    this.span_ = span;
  }


  clear() {
    while (this.span_.lastChild) {
      this.span_.removeChild(this.span_.lastChild);
    }
  }
}
