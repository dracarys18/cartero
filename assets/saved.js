(function () {
  'use strict';

  var KEY = 'cartero:saved';
  var listeners = [];

  function load() {
    try {
      var v = JSON.parse(localStorage.getItem(KEY));
      return Array.isArray(v) ? v : [];
    } catch (e) { return []; }
  }

  function persist(list) {
    try { localStorage.setItem(KEY, JSON.stringify(list)); } catch (e) {}
  }

  function emit() {
    var list = load();
    listeners.forEach(function (fn) { try { fn(list); } catch (e) {} });
    document.dispatchEvent(new CustomEvent('cartero:saved-changed', { detail: list }));
  }

  function isSaved(url) {
    return load().some(function (s) { return s.url === url; });
  }

  function toggleSaved(entry) {
    if (!entry || !entry.url) return false;
    var list = load();
    var i = list.findIndex(function (s) { return s.url === entry.url; });
    var nowSaved;
    if (i >= 0) {
      list.splice(i, 1);
      nowSaved = false;
    } else {
      list.unshift({
        url: entry.url,
        title: entry.title || '',
        siteName: entry.siteName || '',
        byline: entry.byline || '',
        image_url: entry.image_url || '',
        excerpt: entry.excerpt || '',
        saved_at: Date.now()
      });
      nowSaved = true;
    }
    persist(list);
    emit();
    return nowSaved;
  }

  function removeSaved(url) {
    persist(load().filter(function (s) { return s.url !== url; }));
    emit();
  }

  function clearSaved() {
    persist([]);
    emit();
  }

  window.CarteroSaved = {
    load: load,
    isSaved: isSaved,
    toggleSaved: toggleSaved,
    removeSaved: removeSaved,
    clearSaved: clearSaved,
    onChange: function (fn) { listeners.push(fn); }
  };
})();
