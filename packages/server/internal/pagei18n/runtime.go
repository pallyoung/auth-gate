// Package pagei18n provides localization for self-contained gateway HTML pages.
package pagei18n

// Runtime shares the control plane's preference key. Storage remains scoped to
// the current origin; a gateway never assumes access to another host's preference.
const Runtime = `<style>
.language-switcher{display:flex;justify-content:flex-end;gap:6px;margin-bottom:18px}
.language-switcher button{font:inherit;font-size:12px;color:inherit;background:transparent;border:1px solid currentColor;border-radius:6px;padding:6px 10px;cursor:pointer;opacity:.7}
.language-switcher button[aria-pressed="true"]{opacity:1;font-weight:700}
.language-switcher button:focus-visible{outline:2px solid currentColor;outline-offset:2px}
</style><script>
(function(){
  var storageKey = 'auth-gate.locale';
  function normalize(value) {
    value = (value || '').toLowerCase();
    if (value === 'en' || value.indexOf('en-') === 0) return 'en';
    if (value === 'zh' || value.indexOf('zh-') === 0) return 'zh-CN';
    return null;
  }
  var locale;
  try { locale = normalize(localStorage.getItem(storageKey)); } catch (_) {}
  var languages = navigator.languages || [];
  for (var i = 0; !locale && i < languages.length; i++) locale = normalize(languages[i]);
  locale = locale || normalize(navigator.language) || 'en';
  document.documentElement.lang = locale;
  var messages = {};
  function text(key) { return (messages[locale] || {})[key] || (messages.en || {})[key] || key; }
  function apply() {
    document.documentElement.lang = locale;
    document.querySelectorAll('[data-i18n]').forEach(function(el){ el.textContent = text(el.getAttribute('data-i18n')); });
    document.querySelectorAll('[data-language-switcher]').forEach(function(el){ el.setAttribute('aria-label', locale === 'zh-CN' ? '语言切换' : 'Language switcher'); });
    document.querySelectorAll('[data-locale]').forEach(function(el){ el.setAttribute('aria-pressed', String(el.getAttribute('data-locale') === locale)); });
  }
  window.authGateI18n = {
    text: text,
    setText: function(el, key) { el.setAttribute('data-i18n', key); el.textContent = text(key); },
    init: function(nextMessages) {
      messages = nextMessages;
      document.querySelectorAll('[data-locale]').forEach(function(el){
        el.addEventListener('click', function(){
          locale = normalize(el.getAttribute('data-locale')) || 'en';
          try { localStorage.setItem(storageKey, locale); } catch (_) {}
          apply();
        });
      });
      apply();
    }
  };
})();
</script>`

const Switcher = `<div class="language-switcher" role="group" data-language-switcher aria-label="Language switcher"><button type="button" data-locale="en" lang="en">EN</button><button type="button" data-locale="zh-CN" lang="zh-CN">中文</button></div>`
