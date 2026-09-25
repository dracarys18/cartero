(function () {
  'use strict';

  var params = new URLSearchParams(location.search);
  var url = params.get('url');
  var fallbackTitle = params.get('title') || '';

  var statusEl = document.getElementById('status');
  var articleEl = document.getElementById('article');
  var titleEl = document.getElementById('title');
  var kickerEl = document.getElementById('kicker');
  var bylineEl = document.getElementById('byline');
  var heroEl = document.getElementById('hero');
  var heroImg = document.getElementById('hero-img');
  var bodyEl = document.getElementById('reader-body');
  var progressEl = document.getElementById('progress');
  var origLink = document.getElementById('orig-link');
  var saveBtn = document.getElementById('save-btn');

  var current = null;

  var backLink = document.getElementById('back-link');
  try {
    if (history.length > 1 && new URL(document.referrer).origin === location.origin) {
      backLink.addEventListener('click', function (e) {
        e.preventDefault();
        history.back();
      });
    }
  } catch (e) {}

  function safeUrl(u) {
    try {
      var p = new URL(u, location.origin);
      return (p.protocol === 'http:' || p.protocol === 'https:') ? p.href : '#';
    } catch (e) { return '#'; }
  }

  function escapeHtml(s) {
    var d = document.createElement('div');
    d.textContent = s == null ? '' : String(s);
    return d.innerHTML;
  }

  function words(s) {
    return (s || '').trim().split(/\s+/).filter(Boolean).length;
  }

  function readingTime(n) {
    return Math.max(1, Math.round(n / 225)) + ' min read';
  }

  function fmtDate(iso) {
    if (!iso) return '';
    var d = new Date(iso);
    if (isNaN(d.getTime())) return '';
    return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
  }

  function setProgress() {
    var doc = document.documentElement;
    var max = doc.scrollHeight - window.innerHeight;
    var pct = max > 0 ? (window.scrollY / max) * 100 : 0;
    progressEl.style.width = Math.min(100, Math.max(0, pct)) + '%';
  }

  function fail(msg) {
    statusEl.innerHTML =
      '<div class="err">' + escapeHtml(msg || 'Could not load this article.') + '</div>' +
      '<div style="margin-top:12px"><a href="' + safeUrl(url) + '" target="_blank" rel="noopener">Open original ↗</a></div>';
  }

  function refreshSave() {
    if (!current || !current.url) return;
    if (window.CarteroSaved && window.CarteroSaved.isSaved) {
      var saved = window.CarteroSaved.isSaved(current.url);
      saveBtn.classList.toggle('saved', saved);
      saveBtn.setAttribute('aria-label', saved ? 'Remove from saved' : 'Save');
      saveBtn.title = saved ? 'Remove from saved' : 'Save';
    }
  }

  saveBtn.addEventListener('click', function () {
    if (!current || !current.url) return;
    if (window.CarteroSaved && window.CarteroSaved.toggleSaved) {
      window.CarteroSaved.toggleSaved({
        url: current.url,
        title: current.title || fallbackTitle,
        siteName: current.siteName || '',
        byline: current.byline || '',
        image_url: current.image || '',
        excerpt: current.excerpt || ''
      });
      refreshSave();
    }
  });

  window.addEventListener('scroll', setProgress, { passive: true });
  window.addEventListener('resize', setProgress, { passive: true });

  function leadImage(doc, contentHtml) {
    var og = doc.querySelector('meta[property="og:image"], meta[name="twitter:image"]');
    if (og) {
      var c = og.getAttribute('content');
      if (c) return c;
    }
    var tmp = document.createElement('div');
    tmp.innerHTML = window.DOMPurify.sanitize(contentHtml || '', { ADD_TAGS: [] });
    var img = tmp.querySelector('img');
    return img && img.getAttribute('src') ? img.getAttribute('src') : '';
  }

  function render(article, doc) {
    var title = article.title || fallbackTitle || url;
    var html = window.DOMPurify.sanitize(article.content || '');

    titleEl.textContent = title;
    document.title = title + ' · Reader';

    var parts = [];
    if (article.byline) parts.push('<span>' + escapeHtml(article.byline) + '</span>');
    if (article.siteName) parts.push('<span>' + escapeHtml(article.siteName) + '</span>');
    if (article.publishedTime) parts.push('<span>' + escapeHtml(fmtDate(article.publishedTime)) + '</span>');
    parts.push('<span>' + readingTime(words(article.textContent)) + '</span>');
    bylineEl.innerHTML = parts.join('<span class="dot"> &nbsp;·&nbsp; </span>');

    var img = leadImage(doc, article.content);
    if (img) {
      heroImg.src = safeUrl(img);
      heroEl.hidden = false;
    } else {
      heroEl.hidden = true;
    }

    if (article.siteName) {
      kickerEl.textContent = article.siteName;
      kickerEl.hidden = false;
    }

    bodyEl.innerHTML = html;

    current = {
      url: url,
      title: title,
      siteName: article.siteName || '',
      byline: article.byline || '',
      image: img || '',
      excerpt: article.excerpt || ''
    };

    origLink.href = safeUrl(url);
    saveBtn.hidden = false;
    refreshSave();

    statusEl.hidden = true;
    articleEl.hidden = false;
    setProgress();
  }

  if (!url) {
    fail('No article URL provided.');
    return;
  }

  origLink.href = safeUrl(url);

  fetch('/proxy?url=' + encodeURIComponent(url), { headers: { 'Accept': 'text/html' } })
    .then(function (res) {
      if (!res.ok) throw new Error('proxy status ' + res.status);
      return res.text();
    })
    .then(function (html) {
      var doc = new DOMParser().parseFromString(html, 'text/html');
      doc.querySelectorAll('script,style,noscript,iframe,object,embed,link[rel=stylesheet]')
        .forEach(function (el) { el.remove(); });
      var article = new Readability(doc).parse();
      if (!article || !article.content) throw new Error('no content');
      render(article, doc);
    })
    .catch(function (err) {
      console.error('reader:', err);
      fail("Couldn't extract this article. It may be paywalled or block fetching.");
    });
})();
