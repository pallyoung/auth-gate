package proxy

import "github.com/pallyoung/auth-gate/packages/server/internal/pagei18n"

// loginPageHTML is a self-contained HTML page for gateway access login.
// It is served by the proxy engine at /_authgate/access-login when a
// protected route requires gateway authentication.
//
// Query parameters (from buildAccessLoginURL):
//   - route_id   : the route ID
//   - route_name : display name of the route
//   - next       : redirect target after successful login
const loginPageHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title data-i18n="title">Auth Gate - Gateway Login</title>
` + pagei18n.Runtime + `
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:#f5f5f5;display:flex;align-items:center;justify-content:center;min-height:100vh;color:#333}
.card{background:#fff;border-radius:12px;box-shadow:0 2px 12px rgba(0,0,0,.08);padding:32px;width:100%;max-width:400px;margin:16px}
h1{font-size:20px;font-weight:600;margin-bottom:4px}
.sub{color:#888;font-size:13px;margin-bottom:24px}
.field{margin-bottom:16px}
.field label{display:block;font-size:13px;font-weight:500;margin-bottom:6px;color:#555}
.field input{width:100%;padding:10px 12px;border:1px solid #ddd;border-radius:8px;font-size:14px;outline:none;transition:border .2s}
.field input:focus{border-color:#4f7cff}
.btn{width:100%;padding:11px 0;border:none;border-radius:8px;font-size:14px;font-weight:600;color:#fff;background:#4f7cff;cursor:pointer;transition:background .2s}
.btn:hover{background:#3d6af0}
.btn:disabled{background:#b0c4ff;cursor:not-allowed}
.error{background:#fef2f2;color:#dc2626;border:1px solid #fecaca;border-radius:8px;padding:10px 12px;font-size:13px;margin-bottom:16px;display:none}
.route-info{background:#f0f4ff;border-radius:8px;padding:10px 12px;font-size:13px;color:#4f7cff;margin-bottom:20px}
</style>
</head>
<body>
<div class="card">
  ` + pagei18n.Switcher + `
  <h1 data-i18n="heading">Gateway Login</h1>
  <p class="sub" data-i18n="description">Sign in to access protected resources</p>
  <div id="route-info" class="route-info" style="display:none"><span data-i18n="target">Target: </span><span id="route-name"></span></div>
  <div id="error" class="error" role="alert"></div>
  <form id="login-form" novalidate>
    <div class="field">
      <label for="username" data-i18n="username">Username</label>
      <input id="username" name="username" type="text" autocomplete="username" required>
    </div>
    <div class="field">
      <label for="password" data-i18n="password">Password</label>
      <input id="password" name="password" type="password" autocomplete="current-password" required>
    </div>
    <button class="btn" type="submit" id="submit-btn" data-i18n="submit">Sign In</button>
  </form>
</div>
<script>
(function(){
  var i18n = window.authGateI18n;
  i18n.init({
    en: {title:'Auth Gate - Gateway Login',heading:'Gateway Login',description:'Sign in to access protected resources',target:'Target: ',username:'Username',password:'Password',submit:'Sign In',submitting:'Signing in...',required:'Enter your username and password.',failed:'Sign-in failed. Check your credentials and try again.',denied:'You do not have access to this resource. Contact your administrator.',network:'Network error. Please try again.'},
    'zh-CN': {title:'Auth Gate - 网关登录',heading:'网关登录',description:'请登录以访问受保护的资源',target:'目标：',username:'用户名',password:'密码',submit:'登录',submitting:'登录中...',required:'请输入用户名和密码。',failed:'登录失败。请检查凭据后重试。',denied:'你无权访问此资源。请联系管理员。',network:'网络错误，请重试。'}
  });
  var params = new URLSearchParams(window.location.search);
  var routeId = params.get('route_id') || '';
  var routeName = params.get('route_name') || '';
  var next = params.get('next') || '/';
  var info = document.getElementById('route-info');
  if (routeName) { document.getElementById('route-name').textContent = routeName; info.style.display = 'block'; }
  document.getElementById('login-form').addEventListener('submit', function(e){
    e.preventDefault();
    var errEl = document.getElementById('error');
    var btn = document.getElementById('submit-btn');
    if (btn.disabled) return;
    if (!document.getElementById('username').value.trim() || !document.getElementById('password').value) {
      i18n.setText(errEl, 'required'); errEl.style.display = 'block';
      return;
    }
    errEl.style.display = 'none';
    btn.disabled = true; i18n.setText(btn, 'submitting');
    fetch('/api/access/login', {
      method: 'POST',
      headers: {'Content-Type':'application/json'},
      body: JSON.stringify({
        route_id: routeId,
        username: document.getElementById('username').value,
        password: document.getElementById('password').value,
        next: next
      })
    }).then(function(r){ return r.json().then(function(d){ return {ok:r.ok, data:d}; }); })
    .then(function(res){
      if (res.ok) {
        window.location.href = res.data.next || next;
      } else {
        var code = res.data && res.data.error && res.data.error.code;
        i18n.setText(errEl, code === 'access_denied' || code === 'route_access_denied' ? 'denied' : 'failed');
        errEl.style.display = 'block';
        btn.disabled = false; i18n.setText(btn, 'submit');
      }
    }).catch(function(){
      i18n.setText(errEl, 'network'); errEl.style.display = 'block';
      btn.disabled = false; i18n.setText(btn, 'submit');
    });
  });
  document.getElementById('username').focus();
})();
</script>
</body>
</html>`
