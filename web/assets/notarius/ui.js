/* SaktiNotaris UI engine — dropdown, collapse, tabs (tanpa CDN, tanpa Bootstrap ganda).
 * Alasan: bundle vendor tabler.min.js tidak mengekspos window.bootstrap,
 * sehingga atribut data-bs-toggle tidak ter-bind ke apa pun (terbukti via E2E).
 * Markup tetap kompatibel-Bootstrap; engine ini yang menghidupkannya.
 * Idempoten: aman dieksekusi ulang (guard __snBound).
 */
(function () {
  if (window.__snBound) return;
  window.__snBound = true;

  function closeAllDrops(except) {
    document.querySelectorAll('.dropdown-menu.show').forEach(function (m) {
      if (m !== except) {
        m.classList.remove('show');
        var t = m.parentElement && m.parentElement.querySelector('[data-bs-toggle="dropdown"]');
        if (t) t.setAttribute('aria-expanded', 'false');
      }
    });
  }

  document.addEventListener('click', function (e) {
    var t = e.target.closest ? e.target.closest('[data-bs-toggle]') : null;
    if (!t) {
      closeAllDrops(null);
      return;
    }
    var kind = t.getAttribute('data-bs-toggle');
    if (kind === 'dropdown') {
      e.preventDefault();
      var wrap = t.closest('.dropdown');
      var menu = wrap ? wrap.querySelector('.dropdown-menu') : null;
      if (!menu) return;
      var wasOpen = menu.classList.contains('show');
      closeAllDrops(null);
      if (!wasOpen) {
        menu.classList.add('show');
        t.setAttribute('aria-expanded', 'true');
      }
    } else if (kind === 'collapse') {
      var sel = t.getAttribute('data-bs-target');
      if (!sel) return;
      var target = document.querySelector(sel);
      if (!target) return;
      e.preventDefault();
      target.classList.toggle('show');
      var exp = t.getAttribute('aria-expanded') === 'true';
      t.setAttribute('aria-expanded', String(!exp));
    } else if (kind === 'tab') {
      var href = t.getAttribute('href');
      if (!href || href.charAt(0) !== '#') return;
      var pane = document.querySelector(href);
      if (!pane) return;
      e.preventDefault();
      var nav = t.closest('.nav');
      if (nav) {
        nav.querySelectorAll('.nav-link').forEach(function (a) {
          a.classList.remove('active');
          a.removeAttribute('aria-selected');
        });
      }
      t.classList.add('active');
      t.setAttribute('aria-selected', 'true');
      var panes = pane.parentElement ? pane.parentElement.querySelectorAll('.tab-pane') : [pane];
      panes.forEach(function (x) { x.classList.remove('active', 'show'); });
      pane.classList.add('active', 'show');
      try { sessionStorage.setItem('sn-tab:' + location.pathname, href); } catch (e) {}
    }
  });

  // Ingat tab aktif per halaman (kembali ke tab yang sama setelah submit/redirect).
  function restoreTab() {
    var href = null;
    try { href = sessionStorage.getItem('sn-tab:' + location.pathname); } catch (e) {}
    if (!href || href.charAt(0) !== '#') return;
    var link = document.querySelector('.nav [data-bs-toggle="tab"][href="' + href + '"]');
    var pane = document.querySelector(href);
    if (!link || !pane || !pane.classList.contains('tab-pane')) return;
    var nav = link.closest('.nav');
    if (nav) {
      nav.querySelectorAll('.nav-link').forEach(function (a) {
        a.classList.remove('active');
        a.removeAttribute('aria-selected');
      });
    }
    link.classList.add('active');
    link.setAttribute('aria-selected', 'true');
    var panes = pane.parentElement ? pane.parentElement.querySelectorAll('.tab-pane') : [pane];
    panes.forEach(function (x) { x.classList.remove('active', 'show'); });
    pane.classList.add('active', 'show');
  }
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', restoreTab, { once: true });
  } else {
    restoreTab();
  }

  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') closeAllDrops(null);
  });
})();
