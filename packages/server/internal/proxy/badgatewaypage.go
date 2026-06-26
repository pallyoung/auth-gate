package proxy

// badGatewayPageHTML is a self-contained HTML page shown when the upstream
// backend is unreachable or returns an error. It is served when the client
// accepts text/html; otherwise a JSON error envelope is returned for API clients.
const badGatewayPageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>502 - 服务暂不可用</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",sans-serif;background:#0a0f1a;min-height:100vh;display:flex;align-items:center;justify-content:center;color:#e0e6f0}
.container{text-align:center;max-width:480px;padding:40px 24px}
.code{font-size:140px;font-weight:800;line-height:1;color:#1a2942;letter-spacing:-4px;position:relative;display:inline-block}
.code::after{content:"502";position:absolute;inset:0;background:linear-gradient(135deg,#f87171,#fbbf24 60%,#f59e0b);-webkit-background-clip:text;-webkit-text-fill-color:transparent;background-clip:text}
.icon{width:80px;height:80px;border-radius:50%;background:rgba(251,191,36,.1);display:flex;align-items:center;justify-content:center;margin:-24px auto 32px}
.icon svg{width:40px;height:40px;stroke:#fbbf24;fill:none;stroke-width:1.5;stroke-linecap:round;stroke-linejoin:round}
h1{font-size:24px;font-weight:600;margin-bottom:12px;color:#e8ecf0}
p{color:#6b7a90;font-size:15px;line-height:1.7;margin-bottom:32px}
.host{display:inline-block;background:rgba(251,191,36,.08);color:#d4a017;font-family:"IBM Plex Mono","SF Mono",Consolas,monospace;font-size:13px;padding:6px 14px;border-radius:6px;margin-bottom:28px;border:1px solid rgba(251,191,36,.15)}
.actions{display:flex;gap:12px;justify-content:center;flex-wrap:wrap}
.btn{display:inline-flex;align-items:center;gap:8px;padding:11px 22px;border-radius:8px;font-size:14px;font-weight:500;cursor:pointer;text-decoration:none;transition:all .2s}
.btn-ghost{background:#111d2a;border:1px solid #1a2942;color:#b0b8c5}
.btn-ghost:hover{background:#1a2942;color:#e0e6f0}
.btn svg{width:16px;height:16px;stroke:currentColor;fill:none;stroke-width:2;stroke-linecap:round;stroke-linejoin:round}
@media(max-width:480px){.code{font-size:100px}.actions{flex-direction:column;align-items:center}}
@media(prefers-color-scheme:light){body{background:#f5f7fa;color:#1a2942}.code{color:#e2e8f0}h1{color:#1a2942}p{color:#6b7a90}.host{background:rgba(251,191,36,.06);border-color:rgba(251,191,36,.12)}.btn-ghost{background:#fff;border:1px solid #d4dae3;color:#4a5568}.btn-ghost:hover{background:#f0f4f8}}
</style>
</head>
<body>
<div class="container">
  <div class="code">502</div>
  <div class="icon">
    <svg viewBox="0 0 24 24"><path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
  </div>
  <h1>服务暂不可用</h1>
  <p>后端服务暂时无法访问，<br>请稍后再试或联系管理员。</p>
  <div class="host" id="host-info"></div>
  <div class="actions">
    <button class="btn btn-ghost" onclick="location.reload()">
      <svg viewBox="0 0 24 24"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 11-2.12-9.36L23 10"/></svg>
      刷新重试
    </button>
    <button class="btn btn-ghost" onclick="history.back()">
      <svg viewBox="0 0 24 24"><line x1="19" y1="12" x2="5" y2="12"/><polyline points="12 19 5 12 12 5"/></svg>
      返回上一页
    </button>
  </div>
</div>
<script>
document.getElementById('host-info').textContent = window.location.hostname + window.location.pathname;
</script>
</body>
</html>`
