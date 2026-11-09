// Общий код для index.html и admin.html
window.BLOCKED = /(nude|naked|nsfw|sex|porn|explicit|topless|lingerie|erotic|loli|child|kid|teen|minor|school|голы|обнажен|секс|порно|эрот|интим|белье|бельё|лоли|ребен|ребён|дет[иь]|школь|подрост|несовершеннолет)/i;

window.configured = function () {
  return !!(window.SUPABASE_URL && window.SUPABASE_ANON_KEY);
};

window.sb = window.configured()
  ? window.supabase.createClient(window.SUPABASE_URL, window.SUPABASE_ANON_KEY)
  : null;

window.$ = function (id) { return document.getElementById(id); };

window.lsGet = function (k, d) { try { return localStorage.getItem(k) || d; } catch (e) { return d; } };
window.lsSet = function (k, v) { try { localStorage.setItem(k, v); } catch (e) {} };

// Безопасно создаёт элемент с текстом (без innerHTML, защита от XSS)
window.el = function (tag, props, children) {
  var n = document.createElement(tag);
  Object.keys(props || {}).forEach(function (k) {
    if (k === 'text') n.textContent = props[k];
    else if (k === 'class') n.className = props[k];
    else if (k.slice(0, 2) === 'on') n[k] = props[k];
    else n.setAttribute(k, props[k]);
  });
  (children || []).forEach(function (c) { if (c) n.appendChild(c); });
  return n;
};

window.fmtDate = function (s) {
  try { return new Date(s).toLocaleString('ru-RU'); } catch (e) { return s; }
};
