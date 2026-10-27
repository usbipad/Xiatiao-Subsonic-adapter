package main

// 管理页 HTML（单文件，内联样式与脚本）。
const adminPageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Subsonic Adapter 管理</title>
<style>
  :root {
    --bg: #f4f6f9;
    --card: #ffffff;
    --text: #1c1f23;
    --muted: #71767c;
    --line: #e6e9ee;
    --primary: #2f6fdf;
    --primary-dark: #2559b8;
    --danger: #d93025;
    --ok-bg: #e6f6ea;
    --ok-fg: #17692f;
    --bad-bg: #fdecea;
    --bad-fg: #b3261e;
    --radius: 12px;
  }
  * { box-sizing: border-box; }
  body {
    font-family: system-ui, -apple-system, "Segoe UI", "Noto Sans CJK SC", sans-serif;
    margin: 0;
    background: var(--bg);
    color: var(--text);
    -webkit-font-smoothing: antialiased;
  }
  .wrap { max-width: 1080px; margin: 0 auto; padding: 28px 20px 48px; }

  /* 顶部标题栏 */
  .topbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 22px; flex-wrap: wrap; }
  .topbar h1 { font-size: 19px; margin: 0; font-weight: 650; letter-spacing: .2px; }
  .topbar .sub { font-size: 12.5px; color: var(--muted); margin-top: 4px; }
  .topbar-right { display: flex; align-items: center; gap: 10px; }

  /* 概览条 */
  .overview { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 12px; margin-bottom: 20px; }
  .stat { background: var(--card); border-radius: var(--radius); padding: 14px 16px; box-shadow: 0 1px 2px rgba(16,24,40,.06); border: 1px solid var(--line); }
  .stat .label { font-size: 12px; color: var(--muted); margin-bottom: 6px; }
  .stat .value { font-size: 15px; font-weight: 600; word-break: break-all; }

  /* 分组标题 */
  .group-title { font-size: 12.5px; font-weight: 600; color: var(--muted); text-transform: uppercase; letter-spacing: .8px; margin: 26px 0 10px; padding-left: 2px; }
  .group-title:first-of-type { margin-top: 0; }

  /* 卡片 */
  .card { background: var(--card); border-radius: var(--radius); padding: 18px 20px; margin-bottom: 14px; box-shadow: 0 1px 2px rgba(16,24,40,.06); border: 1px solid var(--line); }
  .card h2 { font-size: 14.5px; margin: 0 0 14px; font-weight: 600; display: flex; align-items: center; gap: 8px; }
  .card h2 .dot { width: 7px; height: 7px; border-radius: 50%; background: var(--primary); flex: none; }

  /* 两栏 */
  .grid2 { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
  .grid2 > .card { margin-bottom: 0; }
  @media (max-width: 760px) { .grid2 { grid-template-columns: 1fr; } .wrap { padding: 20px 14px 40px; } }

  /* 行与输入 */
  .row { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
  input[type=text] { flex: 1; min-width: 220px; padding: 9px 12px; border: 1px solid #d3d8e0; border-radius: 8px; font-size: 14px; background: #fbfcfe; transition: border-color .15s, box-shadow .15s; }
  input[type=text]:focus { outline: none; border-color: var(--primary); box-shadow: 0 0 0 3px rgba(47,111,223,.12); background: #fff; }
  button { padding: 9px 16px; border: none; border-radius: 8px; background: var(--primary); color: #fff; font-size: 13.5px; font-weight: 500; cursor: pointer; transition: background .15s, transform .05s; }
  button:hover { background: var(--primary-dark); }
  button:active { transform: translateY(1px); }
  button.secondary { background: #eef1f6; color: #3a4048; }
  button.secondary:hover { background: #e3e8f0; }
  button.danger { background: var(--danger); }
  button.danger:hover { background: #b7241a; }
  button:disabled { opacity: .5; cursor: default; }

  /* 键值表 */
  .kv { display: grid; grid-template-columns: 108px 1fr; gap: 9px 14px; font-size: 13.5px; align-items: center; }
  .kv .k { color: var(--muted); }
  .kv .v, .kv > div:nth-child(even) { word-break: break-all; }
  .mono { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12.5px; }

  /* 徽章 */
  .badge { display: inline-block; padding: 3px 10px; border-radius: 20px; font-size: 12px; font-weight: 500; }
  .badge.ok { background: var(--ok-bg); color: var(--ok-fg); }
  .badge.bad { background: var(--bad-bg); color: var(--bad-fg); }

  /* 二维码 */
  #qrwrap { display: none; margin-top: 14px; text-align: center; }
  #qr { max-width: 200px; border-radius: 10px; border: 1px solid var(--line); padding: 8px; background: #fff; }

  /* 提示文字 */
  .hint { font-size: 12.2px; color: var(--muted); margin-top: 9px; line-height: 1.5; }
  #msg { font-size: 12.5px; color: var(--muted); margin-top: 9px; min-height: 16px; }

  /* 通道切换 */
  .channels { display: flex; gap: 8px; margin-bottom: 14px; flex-wrap: wrap; }
  .channels button { background: #eef1f6; color: #3a4048; padding: 7px 14px; }
  .channels button.active { background: var(--primary); color: #fff; }

  /* 通知条 */
  .notice { margin-top: 12px; padding: 10px 13px; border-radius: 9px; font-size: 13px; display: none; line-height: 1.5; }
  .notice.show { display: block; }
  .notice.success { background: var(--ok-bg); color: var(--ok-fg); border: 1px solid #bfe6c9; }
  .notice.error { background: var(--bad-bg); color: var(--bad-fg); border: 1px solid #f3c9c5; }

  /* 开关 */
  .switch-row { display: flex; align-items: center; gap: 12px; }
  .switch { position: relative; display: inline-block; width: 44px; height: 24px; flex: none; }
  .switch input { opacity: 0; width: 0; height: 0; }
  .slider { position: absolute; cursor: pointer; inset: 0; background: #cfd5de; transition: .2s; border-radius: 24px; }
  .slider:before { position: absolute; content: ""; height: 18px; width: 18px; left: 3px; bottom: 3px; background: #fff; transition: .2s; border-radius: 50%; box-shadow: 0 1px 2px rgba(0,0,0,.2); }
  .switch input:checked + .slider { background: var(--primary); }
  .switch input:checked + .slider:before { transform: translateX(20px); }
  .switch-label { font-size: 13.5px; }

  /* 登录 / 首次设置覆盖层 */
  #authLayer { position: fixed; inset: 0; background: rgba(244,246,249,.96); backdrop-filter: blur(3px); display: none; align-items: center; justify-content: center; z-index: 100; padding: 20px; }
  #authLayer.show { display: flex; }
  .auth-card { background: #fff; border-radius: 14px; box-shadow: 0 8px 32px rgba(16,24,40,.14); border: 1px solid var(--line); padding: 28px 26px; width: 100%; max-width: 360px; }
  .auth-card h2 { font-size: 17px; margin: 0 0 6px; }
  .auth-card .desc { font-size: 12.5px; color: var(--muted); margin-bottom: 20px; line-height: 1.5; }
  .auth-field { margin-bottom: 14px; }
  .auth-field label { display: block; font-size: 12.5px; color: var(--muted); margin-bottom: 6px; }
  .auth-field input { width: 100%; padding: 10px 12px; border: 1px solid #d3d8e0; border-radius: 8px; font-size: 14px; background: #fbfcfe; }
  .auth-field input:focus { outline: none; border-color: var(--primary); box-shadow: 0 0 0 3px rgba(47,111,223,.12); background: #fff; }
  .auth-card button { width: 100%; padding: 11px; font-size: 14px; margin-top: 4px; }
  .auth-error { font-size: 12.5px; color: var(--bad-fg); margin-top: 10px; min-height: 16px; }
  .topbar .logout-btn { background: transparent; color: var(--muted); border: 1px solid var(--line); padding: 5px 12px; font-size: 12.5px; }

  /* 已登录视图 */
  .account { display: flex; align-items: center; gap: 12px; padding: 14px 16px; background: linear-gradient(135deg, #f0f6ff, #eaf1fb); border-radius: 10px; border: 1px solid #d6e3f8; }
  .account .avatar { width: 42px; height: 42px; border-radius: 50%; background: var(--primary); color: #fff; display: flex; align-items: center; justify-content: center; font-size: 17px; font-weight: 600; flex: none; overflow: hidden; }
  .account .avatar img { width: 100%; height: 100%; object-fit: cover; display: block; }
  .account .meta { min-width: 0; }
  .account .name { font-size: 15px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .account .desc { font-size: 12.5px; color: var(--muted); margin-top: 2px; }
  #loginBox { display: block; }
  #accountBox { display: none; }
  .logged-in #loginBox { display: none; }
  .logged-in #accountBox { display: block; }
  .topbar .badge-wrap { display: flex; align-items: center; gap: 8px; }
  .topbar .nick { font-size: 13.5px; color: var(--muted); max-width: 180px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
</head>
<body>

<div id="authLayer">
  <div class="auth-card">
    <h2 id="authTitle">管理页登录</h2>
    <div class="desc" id="authDesc">请输入管理页的用户名和密码。</div>
    <div class="auth-field">
      <label>用户名</label>
      <input type="text" id="authUser" autocomplete="username">
    </div>
    <div class="auth-field">
      <label>密码</label>
      <input type="password" id="authPass" autocomplete="current-password">
    </div>
    <button id="authSubmit">登录</button>
    <div class="auth-error" id="authError"></div>
  </div>
</div>

<div class="wrap">

  <div class="topbar">
    <div>
      <h1>Subsonic Adapter</h1>
      <div class="sub">QQ 音乐 → Subsonic 网关管理</div>
    </div>
    <div class="topbar-right">
      <div class="badge-wrap">
        <span class="nick" id="topNick"></span>
        <span class="badge bad" id="login">未登录</span>
        <button class="logout-btn" id="adminLogout" title="退出管理页">退出</button>
      </div>
    </div>
  </div>

  <div class="overview">
    <div class="stat"><div class="label">监听地址</div><div class="value mono" id="addr">-</div></div>
    <div class="stat"><div class="label">上游连通</div><div class="value" id="reach">-</div></div>
    <div class="stat"><div class="label">运行时长</div><div class="value" id="uptime">-</div></div>
    <div class="stat"><div class="label">进程 PID</div><div class="value mono" id="svcPid">-</div></div>
  </div>

  <div class="group-title">连接</div>
  <div class="card">
    <h2><span class="dot"></span>播放器连接地址</h2>
    <div class="kv" id="connBox"></div>
    <div class="hint">在播放器的「在线音源」设置里填写以上地址，用户名与密码任意。</div>
  </div>

  <div class="group-title">配置</div>
  <div class="grid2">
    <div class="card">
      <h2><span class="dot"></span>上游地址</h2>
      <div class="row">
        <input type="text" id="upstream" placeholder="http://127.0.0.1:3200">
        <button id="save">保存</button>
        <button class="secondary" id="check">检测</button>
      </div>
      <div id="msg"></div>
      <div class="hint">上游服务需自行部署运行，本页只负责转发与检测。</div>
    </div>

    <div class="card">
      <h2><span class="dot"></span>启动设置</h2>
      <div class="switch-row">
        <label class="switch">
          <input type="checkbox" id="autostartToggle">
          <span class="slider"></span>
        </label>
        <span class="switch-label">登录后自动启动（后台运行）</span>
      </div>
      <div class="hint" id="autostartHint">开启后，登录桌面时自动在后台启动，不弹出浏览器窗口。</div>
      <div class="notice" id="autostartNotice"></div>
    </div>
  </div>

  <div class="group-title">QQ 音乐账号</div>
  <div class="card">
    <div id="loginBox">
      <h2><span class="dot"></span>扫码登录</h2>
      <div class="channels" id="channels"></div>
      <div class="row">
        <button id="qrbtn">获取二维码</button>
      </div>
      <div id="qrwrap"><img id="qr" alt="二维码"></div>
      <div class="hint">用对应的 App 扫码确认，页面会自动检测登录结果。免登录接口（搜索、歌词、歌单、排行榜）无需登录即可使用。</div>
      <div class="notice" id="notice"></div>
    </div>
    <div id="accountBox">
      <h2><span class="dot"></span>当前账号</h2>
      <div class="account">
        <div class="avatar" id="avatarInitial">?</div>
        <div class="meta">
          <div class="name" id="accountName">-</div>
          <div class="desc">已登录，可使用个人歌单、红心、收藏、云盘等需要登录的功能。</div>
        </div>
      </div>
      <div class="row" style="margin-top:14px">
        <button class="secondary" id="logout">退出登录</button>
      </div>
      <div class="notice" id="notice2"></div>
    </div>
  </div>

  <div class="group-title">管理页安全</div>
  <div class="card">
    <h2><span class="dot"></span>登录凭据</h2>
    <div class="kv">
      <div class="k">用户名</div><div class="v mono" id="adminUserName">-</div>
      <div class="k">密码</div><div class="v">••••••••</div>
    </div>
    <div class="row" style="margin-top:14px">
      <button id="changeCredBtn">修改用户名或密码</button>
    </div>
    <div id="credForm" style="display:none;margin-top:14px">
      <div class="auth-field">
        <label>新用户名</label>
        <input type="text" id="credUser" autocomplete="username">
      </div>
      <div class="auth-field">
        <label>新密码（至少 4 位）</label>
        <input type="password" id="credPass" autocomplete="new-password">
      </div>
      <div class="row">
        <button id="credSave">保存</button>
        <button class="secondary" id="credCancel">取消</button>
      </div>
      <div class="auth-error" id="credError"></div>
    </div>
    <div class="hint">忘记密码时，删除 <span class="mono">~/.config/subsonic-adapter/auth.json</span> 并重启服务即可重新设置。</div>
    <div class="notice" id="credNotice"></div>
  </div>

  <div class="group-title">服务</div>
  <div class="card">
    <h2><span class="dot"></span>进程信息与操作</h2>
    <div class="kv">
      <div class="k">可执行文件</div><div class="v mono" id="svcExe">-</div>
      <div class="k">启动参数</div><div class="v mono" id="svcArgs">-</div>
      <div class="k">上游地址</div><div class="v mono" id="up">-</div>
    </div>
    <div class="row" style="margin-top:14px">
      <button id="svcRestart">重启服务</button>
      <button class="danger" id="svcStop">停止服务</button>
    </div>
    <div class="hint">重启会原地替换进程，页面断开数秒后自动恢复；停止后需手动重新启动（桌面图标或 <span class="mono">launch.sh</span>）。</div>
    <div class="notice" id="svcNotice"></div>
  </div>

</div>

<script>
let pollTimer = null;
let currentChannel = 'qq';
let channelNames = {};

async function api(path, opts) {
  const res = await fetch(path, opts);
  if (res.status === 401) {
    // 会话过期：立刻回到登录层，避免页面显示空白数据。
    showAuthLayer(false, '', '');
  }
  return res.json();
}

let authMode = 'login';

function showAuthLayer(isSetup, title, desc) {
  authMode = isSetup ? 'setup' : 'login';
  const layer = document.getElementById('authLayer');
  layer.classList.add('show');
  document.getElementById('authTitle').textContent = isSetup ? '首次使用：设置管理凭据' : '管理页登录';
  document.getElementById('authDesc').textContent = isSetup
    ? '请设置访问管理页的用户名和密码，设置后每次打开都需要登录。'
    : '请输入管理页的用户名和密码。';
  document.getElementById('authSubmit').textContent = isSetup ? '设置并进入' : '登录';
  document.getElementById('authError').textContent = '';
  const passEl = document.getElementById('authPass');
  passEl.autocomplete = isSetup ? 'new-password' : 'current-password';
  setTimeout(function () { document.getElementById('authUser').focus(); }, 50);
}

function hideAuthLayer() {
  document.getElementById('authLayer').classList.remove('show');
}

async function checkAuthState() {
  try {
    const r = await api('/api/auth/state');
    if (!r.ok) return;
    if (r.username) {
      const el = document.getElementById('adminUserName');
      if (el) el.textContent = r.username;
    }
    if (!r.configured) {
      showAuthLayer(true);
      return;
    }
    if (!r.loggedIn) {
      showAuthLayer(false);
    } else {
      hideAuthLayer();
    }
  } catch (e) {
    /* 网络异常时不阻塞页面 */
  }
}

document.getElementById('authSubmit').onclick = async () => {
  const user = document.getElementById('authUser').value.trim();
  const pass = document.getElementById('authPass').value;
  const errEl = document.getElementById('authError');
  errEl.textContent = '';
  if (!user) { errEl.textContent = '请输入用户名'; return; }
  if (!pass) { errEl.textContent = '请输入密码'; return; }
  const url = authMode === 'setup' ? '/api/auth/setup' : '/api/auth/login';
  try {
    const r = await api(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: user, password: pass })
    });
    if (!r.ok) {
      errEl.textContent = r.error || '操作失败';
      return;
    }
    hideAuthLayer();
    document.getElementById('authPass').value = '';
    await checkAuthState();
    refresh();
    loadAutostart();
    loadServiceInfo();
  } catch (e) {
    errEl.textContent = '请求失败：' + e;
  }
};

document.getElementById('authPass').addEventListener('keydown', function (ev) {
  if (ev.key === 'Enter') document.getElementById('authSubmit').click();
});

document.getElementById('adminLogout').onclick = async () => {
  await api('/api/auth/logout', { method: 'POST' });
  showAuthLayer(false);
};

function showCredNotice(text, kind) {
  const el = document.getElementById('credNotice');
  el.textContent = text;
  el.className = 'notice show ' + (kind || 'success');
}

document.getElementById('changeCredBtn').onclick = function () {
  const form = document.getElementById('credForm');
  const btn = document.getElementById('changeCredBtn');
  const open = form.style.display === 'none';
  form.style.display = open ? 'block' : 'none';
  btn.textContent = open ? '收起' : '修改用户名或密码';
  document.getElementById('credError').textContent = '';
  document.getElementById('credPass').value = '';
  document.getElementById('credNotice').className = 'notice';
  if (open) {
    document.getElementById('credUser').value = document.getElementById('adminUserName').textContent;
    document.getElementById('credUser').focus();
  }
};

document.getElementById('credCancel').onclick = function () {
  document.getElementById('credForm').style.display = 'none';
  document.getElementById('changeCredBtn').textContent = '修改用户名或密码';
};

document.getElementById('credSave').onclick = async function () {
  const user = document.getElementById('credUser').value.trim();
  const pass = document.getElementById('credPass').value;
  const errEl = document.getElementById('credError');
  errEl.textContent = '';
  if (!user) { errEl.textContent = '用户名不能为空'; return; }
  if (pass.length < 4) { errEl.textContent = '密码至少 4 位'; return; }
  try {
    const r = await api('/api/auth/change', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: user, password: pass })
    });
    if (!r.ok) { errEl.textContent = r.error || '保存失败'; return; }
    document.getElementById('adminUserName').textContent = user;
    document.getElementById('credForm').style.display = 'none';
    document.getElementById('changeCredBtn').textContent = '修改用户名或密码';
    document.getElementById('credPass').value = '';
    showCredNotice('✓ 已更新登录凭据，下次登录请使用新用户名和密码。', 'success');
  } catch (e) {
    errEl.textContent = '请求失败：' + e;
  }
};

function activeNoticeEl() {
  return document.body.classList.contains('logged-in')
    ? document.getElementById('notice2')
    : document.getElementById('notice');
}

function showNotice(text, kind) {
  const el = activeNoticeEl();
  if (!el) return;
  el.textContent = text;
  el.className = 'notice show ' + (kind || 'success');
}

function hideNotice() {
  ['notice', 'notice2'].forEach(function (id) {
    const el = document.getElementById(id);
    if (el) el.className = 'notice';
  });
}

function applyLoginState(loggedIn, nickname, avatar) {
  document.body.classList.toggle('logged-in', !!loggedIn);
  badge(document.getElementById('login'), loggedIn, '已登录', '未登录');
  document.getElementById('topNick').textContent = loggedIn && nickname ? nickname : '';
  if (!loggedIn) return;

  const name = nickname || 'QQ 音乐用户';
  document.getElementById('accountName').textContent = name;

  const box = document.getElementById('avatarInitial');
  if (avatar) {
    box.innerHTML = '';
    const img = document.createElement('img');
    img.src = avatar;
    img.alt = name;
    img.referrerPolicy = 'no-referrer';
    img.onerror = function () {
      box.textContent = name.trim().charAt(0).toUpperCase() || '?';
    };
    box.appendChild(img);
  } else {
    box.textContent = name.trim().charAt(0).toUpperCase() || '?';
  }
}

function showSvcNotice(text, kind) {
  const el = document.getElementById('svcNotice');
  el.textContent = text;
  el.className = 'notice show ' + (kind || 'success');
}

async function loadServiceInfo() {
  try {
    const r = await api('/api/service/info');
    if (!r.ok || !r.service) return;
    document.getElementById('svcPid').textContent = r.service.pid;
    document.getElementById('svcExe').textContent = r.service.exe || '-';
    document.getElementById('svcArgs').textContent = (r.service.args || []).join(' ');
  } catch (e) {
    /* 服务重启期间会短暂不可用，忽略 */
  }
}

function waitUntilBack() {
  const notice = document.getElementById('svcNotice');
  let tries = 0;
  const timer = setInterval(async () => {
    tries++;
    try {
      const res = await fetch('/api/service/info', { cache: 'no-store' });
      if (res.ok) {
        clearInterval(timer);
        showSvcNotice('✓ 服务已重启完成，进程已恢复。', 'success');
        loadServiceInfo();
        refresh();
        return;
      }
    } catch (e) { /* 仍未就绪 */ }
    if (tries > 40) {
      clearInterval(timer);
      showSvcNotice('重启超时，请检查日志。', 'error');
    }
  }, 500);
  notice.className = 'notice show success';
  notice.textContent = '正在重启，等待服务恢复…';
}

function showAutostartNotice(text, kind) {
  const el = document.getElementById('autostartNotice');
  el.textContent = text;
  el.className = 'notice show ' + (kind || 'success');
}

function autostartHintText(r) {
  if (!r.enabled) {
    return '开启后，登录桌面时自动在后台启动，不弹出浏览器窗口。';
  }
  if (r.method === 'systemd') {
    return '已启用 systemd 用户服务，登录后自动启动；崩溃会自动重启。';
  }
  if (r.location) {
    return '已开启。配置目录：' + r.location;
  }
  return '已开启：登录桌面时自动在后台启动。';
}

async function loadAutostart() {
  try {
    const r = await api('/api/autostart');
    if (!r.ok) return;
    document.getElementById('autostartToggle').checked = !!r.enabled;
    document.getElementById('autostartHint').textContent = autostartHintText(r);
  } catch (e) { /* ignore */ }
}

async function loadChannels() {
  const r = await api('/api/login/channels');
  const box = document.getElementById('channels');
  box.innerHTML = '';
  (r.channels || []).forEach(function (c) {
    channelNames[c.id] = c.name;
    const btn = document.createElement('button');
    btn.textContent = c.name;
    btn.dataset.id = c.id;
    if (c.id === currentChannel) btn.className = 'active';
    btn.onclick = function () {
      currentChannel = c.id;
      Array.prototype.forEach.call(box.children, function (b) {
        b.className = b.dataset.id === currentChannel ? 'active' : '';
      });
      hideNotice();
    };
    box.appendChild(btn);
  });
}

function badge(el, ok, textOk, textBad) {
  el.className = 'badge ' + (ok ? 'ok' : 'bad');
  el.textContent = ok ? textOk : textBad;
}

function fmtUptime(sec) {
  sec = Math.max(0, Math.floor(sec || 0));
  const d = Math.floor(sec / 86400);
  const h = Math.floor((sec % 86400) / 3600);
  const m = Math.floor((sec % 3600) / 60);
  const s = sec % 60;
  if (d > 0) return d + ' 天 ' + h + ' 小时';
  if (h > 0) return h + ' 小时 ' + m + ' 分';
  if (m > 0) return m + ' 分 ' + s + ' 秒';
  return s + ' 秒';
}

async function refresh() {
  const s = await api('/api/status');
  document.getElementById('addr').textContent = s.host + ':' + s.port;
  document.getElementById('up').textContent = s.upstreamUrl;
  document.getElementById('upstream').value = s.upstreamUrl;
  document.getElementById('uptime').textContent = fmtUptime(s.uptimeSec);
  const reach = await api('/api/upstream/check');
  badge(document.getElementById('reach'), reach.reachable, '正常', '不可达');
  applyLoginState(!!s.loggedIn, s.nickname || '', s.avatar || '');

  const box = document.getElementById('connBox');
  box.innerHTML = '';
  const local = document.createElement('div');
  local.className = 'k'; local.textContent = '本机';
  const localV = document.createElement('div');
  localV.className = 'mono';
  localV.textContent = s.localUrl || '-';
  box.appendChild(local); box.appendChild(localV);
  (s.addresses || []).forEach(a => {
    const k = document.createElement('div'); k.className = 'k'; k.textContent = '局域网';
    const v = document.createElement('div'); v.className = 'mono'; v.textContent = a.url;
    box.appendChild(k); box.appendChild(v);
  });
}

document.getElementById('save').onclick = async () => {
  const upstreamUrl = document.getElementById('upstream').value.trim();
  const r = await api('/api/upstream', { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify({upstreamUrl}) });
  document.getElementById('msg').textContent = r.ok ? '已保存' : (r.error || '保存失败');
  refresh();
};

document.getElementById('check').onclick = async () => {
  const r = await api('/api/upstream/check');
  document.getElementById('msg').textContent = r.reachable ? '上游可达' : '上游不可达';
  refresh();
};

document.getElementById('qrbtn').onclick = async () => {
  hideNotice();
  const r = await api('/api/qrcode?channel=' + encodeURIComponent(currentChannel));
  const img = document.getElementById('qr');
  const wrap = document.getElementById('qrwrap');
  if (!r.ok) { showNotice(r.error || '获取二维码失败', 'error'); return; }
  img.src = r.qrimg;
  img.style.display = 'block';
  wrap.style.display = 'block';
  if (pollTimer) clearInterval(pollTimer);
  pollTimer = setInterval(async () => {
    const c = await api('/api/login/check?key=' + encodeURIComponent(r.key));
    if (c.loggedIn) {
      clearInterval(pollTimer); pollTimer = null;
      img.style.display = 'none';
      wrap.style.display = 'none';
      const name = channelNames[currentChannel] || currentChannel;
      applyLoginState(true, '', '');
      await refresh();
      showNotice('✓ 登录成功（' + name + ' 通道）。', 'success');
    }
  }, 2000);
};

document.getElementById('logout').onclick = async () => {
  await api('/api/logout');
  document.getElementById('qrwrap').style.display = 'none';
  applyLoginState(false, '', '');
  await refresh();
  showNotice('已退出登录', 'success');
};

document.getElementById('svcRestart').onclick = async () => {
  if (!confirm('确定要重启服务吗？页面会短暂断开。')) return;
  try {
    await api('/api/service/restart', { method: 'POST' });
  } catch (e) { /* 连接可能已被切断 */ }
  waitUntilBack();
};

document.getElementById('svcStop').onclick = async () => {
  if (!confirm('确定要停止服务吗？停止后需手动启动。')) return;
  try {
    await api('/api/service/stop', { method: 'POST' });
  } catch (e) { /* 连接可能已被切断 */ }
  showSvcNotice('服务已停止。请通过桌面图标或 launch.sh 重新启动。', 'error');
};

document.getElementById('autostartToggle').onchange = async (ev) => {
  const enabled = ev.target.checked;
  try {
    const r = await api('/api/autostart', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enabled: enabled })
    });
    if (!r.ok) {
      ev.target.checked = !enabled;
      showAutostartNotice(r.error || '设置失败', 'error');
      return;
    }
    document.getElementById('autostartToggle').checked = !!r.enabled;
    document.getElementById('autostartHint').textContent = autostartHintText(r);
    showAutostartNotice(r.enabled ? '✓ 已开启登录自启。' : '✓ 已关闭登录自启。', 'success');
  } catch (e) {
    ev.target.checked = !enabled;
    showAutostartNotice('请求失败：' + e, 'error');
  }
};

checkAuthState();
loadAutostart();
loadChannels();
loadServiceInfo();
refresh();
setInterval(refresh, 15000);
</script>
</body>
</html>`
