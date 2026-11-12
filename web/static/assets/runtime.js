/* OPERA prototype runtime — renders template + logic class, no dependencies */
(function () {
  class DCLogic {
    constructor(props) { this.props = props || {}; this.state = {}; }
    setState(p) { Object.assign(this.state, typeof p === 'function' ? p(this.state) : p); if (this.__render) this.__render(); }
    forceUpdate() { if (this.__render) this.__render(); }
    renderVals() { return {}; }
  }
  window.DCLogic = DCLogic;

  var HOLE = /\{\{\s*([^}]+?)\s*\}\}/g;
  var WHOLE = /^\s*\{\{\s*([^}]+?)\s*\}\}\s*$/;
  var BOOL_PROPS = { checked: 1, disabled: 1, readonly: 1, selected: 1, hidden: 1, open: 1, required: 1 };

  function lookup(path, scope) {
    path = path.trim();
    if (path === 'true') return true;
    if (path === 'false') return false;
    if (path === 'null') return null;
    if (/^-?\d+(\.\d+)?$/.test(path)) return Number(path);
    if (/^'.*'$/.test(path) || /^".*"$/.test(path)) return path.slice(1, -1);
    var parts = path.split('.');
    var v;
    for (var i = scope.length - 1; i >= 0; i--) {
      if (scope[i] && Object.prototype.hasOwnProperty.call(scope[i], parts[0])) { v = scope[i]; break; }
    }
    if (v === undefined) return undefined;
    for (var j = 0; j < parts.length; j++) { if (v == null) return undefined; v = v[parts[j]]; }
    return v;
  }
  function interp(str, scope) {
    return str.replace(HOLE, function (_, p) { var v = lookup(p, scope); return v == null ? '' : String(v); });
  }
  function eventName(el, attr) {
    var n = attr.slice(2);
    if (n === 'change') {
      var t = (el.getAttribute('type') || '').toLowerCase();
      var tag = el.tagName.toLowerCase();
      if (tag === 'textarea' || (tag === 'input' && ['checkbox', 'radio', 'file'].indexOf(t) < 0)) return 'input';
    }
    return n;
  }
  function children(node) { return node.tagName === 'TEMPLATE' ? node.content.childNodes : node.childNodes; }

  function build(nodes, scope, parent) {
    for (var i = 0; i < nodes.length; i++) {
      var n = nodes[i];
      if (n.nodeType === 3) { parent.appendChild(document.createTextNode(interp(n.nodeValue, scope))); continue; }
      if (n.nodeType !== 1) continue;
      var sc = n.tagName === 'TEMPLATE' ? n.getAttribute('data-sc') : null;
      if (sc === 'for') {
        var m = WHOLE.exec(n.getAttribute('list') || '');
        var list = m ? lookup(m[1], scope) : [];
        var as = n.getAttribute('as') || 'item';
        (list || []).forEach(function (item, idx) {
          var s = {}; s[as] = item; s.$index = idx;
          build(children(n), scope.concat([s]), parent);
        });
        continue;
      }
      if (sc === 'if') {
        var mi = WHOLE.exec(n.getAttribute('value') || '');
        if (mi && lookup(mi[1], scope)) build(children(n), scope, parent);
        continue;
      }
      var el = n.cloneNode(false);
      var deferred = null;
      var attrs = Array.prototype.slice.call(el.attributes);
      for (var a = 0; a < attrs.length; a++) {
        var name = attrs[a].name, val = attrs[a].value;
        if (name.indexOf('hint-') === 0) { el.removeAttribute(name); continue; }
        var w = WHOLE.exec(val);
        if (name.indexOf('on') === 0 && name.length > 2) {
          el.removeAttribute(name);
          if (w) { var fn = lookup(w[1], scope); if (typeof fn === 'function') el.addEventListener(eventName(el, name), fn); }
          continue;
        }
        if (name === 'value') { el.removeAttribute('value'); deferred = w ? lookup(w[1], scope) : interp(val, scope); continue; }
        if (BOOL_PROPS[name] && w) {
          var b = !!lookup(w[1], scope);
          el.removeAttribute(name);
          if (b) el.setAttribute(name, '');
          if (name === 'checked') el.checked = b;
          if (name === 'readonly') el.readOnly = b;
          if (name === 'disabled') el.disabled = b;
          continue;
        }
        if (val.indexOf('{{') >= 0) {
          var v2 = w ? lookup(w[1], scope) : interp(val, scope);
          el.setAttribute(name, v2 == null ? '' : String(v2));
        }
      }
      build(children(n), scope, el);
      parent.appendChild(el);
      if (deferred !== null) { el.value = deferred == null ? '' : deferred; if (el.tagName === 'TEXTAREA') el.textContent = el.value; }
    }
  }

  function pathOf(node, root) {
    var p = [];
    while (node && node !== root) { var i = 0, s = node; while ((s = s.previousSibling)) i++; p.unshift(i); node = node.parentNode; }
    return node === root ? p : null;
  }
  function nodeAt(root, p) { var n = root; for (var i = 0; i < p.length && n; i++) n = n.childNodes[p[i]]; return n; }

  var current = null;
  window.DCMount = function (Component, tpl, root) {
    root = root || document.getElementById('app');
    tpl = tpl || document.getElementById('tpl');
    if (current && current.componentWillUnmount) current.componentWillUnmount();
    var comp = new Component({});
    current = comp;
    comp.__render = function () {
      if (current !== comp) return;
      var ae = document.activeElement, path = null, sel = null;
      if (ae && root.contains(ae)) {
        path = pathOf(ae, root);
        try { sel = [ae.selectionStart, ae.selectionEnd]; } catch (e) { sel = null; }
      }
      var scrolls = [];
      root.querySelectorAll('*').forEach(function (e) { if (e.scrollLeft || e.scrollTop) scrolls.push([pathOf(e, root), e.scrollLeft, e.scrollTop]); });
      var frag = document.createDocumentFragment();
      build(tpl.content.childNodes, [comp.renderVals()], frag);
      root.replaceChildren(frag);
      scrolls.forEach(function (s) { var e = nodeAt(root, s[0]); if (e) { e.scrollLeft = s[1]; e.scrollTop = s[2]; } });
      if (path) {
        var n = nodeAt(root, path);
        if (n && n.focus) { n.focus({ preventScroll: true }); if (sel && sel[0] != null) { try { n.setSelectionRange(sel[0], sel[1]); } catch (e) {} } }
      }
    };
    comp.__render();
    if (comp.componentDidMount) comp.componentDidMount();
    return comp;
  };
})();
