/* Router hash: #/rute-layar */
(function () {
  var styleEl = document.getElementById('screen-css');
  function go() {
    var route = location.hash.replace(/^#\/?/, '').replace(/\/$/, '');
    var sc = window.SCREENS[route] || window.SCREENS[''];
    if (!sc._tpl) {
      var t = document.createElement('template');
      t.innerHTML = sc.tpl;
      sc._tpl = t;
      sc._C = sc.make();
    }
    styleEl.textContent = sc.css;
    document.title = sc.title + ' · OPERA';
    window.DCMount(sc._C, sc._tpl, document.getElementById('app'));
    window.scrollTo(0, 0);
  }
  window.addEventListener('hashchange', go);
  go();
})();
