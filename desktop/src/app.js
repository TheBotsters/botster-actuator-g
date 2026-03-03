// Botster Desktop — plain JS, no frameworks
// Uses Tauri's global API when available, gracefully degrades to demo mode

var invoke = (window.__TAURI__ && window.__TAURI__.core) ? window.__TAURI__.core.invoke : null;

// ─── Demo Data ───

var DEMO_ACTIVITY = [
  { icon: '🔗', type: 'connect', text: 'Agent <strong>connected</strong> to broker', time: '11:02' },
  { icon: '📄', type: 'file', text: 'File access granted: <strong>project-brief.pdf</strong>', time: '10:58' },
  { icon: '⚡', type: 'exec', text: 'Agent executed: <strong>git status</strong>', time: '10:45' },
  { icon: '📁', type: 'file', text: 'Directory access granted: <strong>~/Projects/my-app/</strong>', time: '10:32' },
  { icon: '⚡', type: 'exec', text: 'Agent executed: <strong>npm test</strong> — 42 tests passed', time: '10:18' },
  { icon: '📄', type: 'file', text: 'File access granted: <strong>requirements.txt</strong>', time: '09:55' },
  { icon: '🔗', type: 'connect', text: 'Broker health check: <strong>OK</strong> (12ms)', time: '09:30' },
  { icon: '⚡', type: 'exec', text: 'Agent executed: <strong>docker ps</strong>', time: '09:12' },
];

var DEMO_ACCESS = [
  { path: '~/Documents/project-brief.pdf', type: 'file', granted: 'Today, 10:58 AM' },
  { path: '~/Projects/my-app/', type: 'dir', granted: 'Today, 10:32 AM' },
  { path: '~/Documents/requirements.txt', type: 'file', granted: 'Today, 9:55 AM' },
];

var DEMO_LOGS = [
  { ts: '11:02:14', level: 'success', msg: 'Connected to broker.botsters.dev' },
  { ts: '11:02:14', level: 'info', msg: 'Agent ID: peter-agent-01' },
  { ts: '11:02:13', level: 'info', msg: 'Capabilities: shell, filesystem, git, docker' },
  { ts: '11:02:12', level: 'info', msg: 'TLS handshake complete (ECDHE-RSA-AES256-GCM)' },
  { ts: '11:02:11', level: 'info', msg: 'Initiating WebSocket connection...' },
  { ts: '10:58:03', level: 'info', msg: 'File access granted: ~/Documents/project-brief.pdf' },
  { ts: '10:45:22', level: 'info', msg: 'Command executed: git status (exit 0, 234ms)' },
  { ts: '10:32:17', level: 'info', msg: 'Directory access granted: ~/Projects/my-app/' },
  { ts: '10:18:44', level: 'success', msg: 'Command executed: npm test (exit 0, 3.2s) — 42/42 passed' },
  { ts: '10:18:41', level: 'info', msg: 'Spawning: npm test --no-color' },
  { ts: '09:55:08', level: 'info', msg: 'File access granted: ~/Documents/requirements.txt' },
  { ts: '09:30:00', level: 'info', msg: 'Health check: broker OK (12ms RTT)' },
  { ts: '09:12:33', level: 'info', msg: 'Command executed: docker ps (exit 0, 89ms)' },
  { ts: '09:12:30', level: 'warn', msg: 'Docker socket permission: using user group (no sudo)' },
  { ts: '08:28:01', level: 'success', msg: 'Connected to broker.botsters.dev' },
  { ts: '08:28:00', level: 'info', msg: 'Actuator G v0.1.0 starting...' },
];

// ─── Tab Navigation ───

var navItems = document.querySelectorAll('.nav-item');
var tabPanels = document.querySelectorAll('.tab-panel');

navItems.forEach(function (item) {
  item.addEventListener('click', function () {
    var tab = this.getAttribute('data-tab');
    navItems.forEach(function (n) { n.classList.remove('active'); });
    tabPanels.forEach(function (p) { p.classList.remove('active'); });
    this.classList.add('active');
    document.getElementById('tab-' + tab).classList.add('active');
  });
});

// ─── Render Activity Feed ───

function renderActivity() {
  var feed = document.getElementById('activity-feed');
  feed.innerHTML = DEMO_ACTIVITY.map(function (a) {
    return '<div class="activity-item">' +
      '<div class="activity-icon ' + a.type + '">' + a.icon + '</div>' +
      '<div class="activity-text">' + a.text + '</div>' +
      '<div class="activity-time">' + a.time + '</div>' +
      '</div>';
  }).join('');
}

// ─── Render Access List ───

function renderAccessList() {
  var list = document.getElementById('access-list');
  var count = document.getElementById('access-count');
  count.textContent = DEMO_ACCESS.length;

  list.innerHTML = DEMO_ACCESS.map(function (a, i) {
    var isDir = a.type === 'dir';
    return '<div class="access-item" data-index="' + i + '">' +
      '<div class="access-icon ' + a.type + '">' + (isDir ? '📁' : '📄') + '</div>' +
      '<div class="access-info">' +
        '<div class="access-path">' + a.path + '</div>' +
        '<div class="access-meta">' + (isDir ? 'Directory' : 'File') + ' · Granted ' + a.granted + '</div>' +
      '</div>' +
      '<button class="btn-revoke" data-index="' + i + '">Revoke</button>' +
      '</div>';
  }).join('');

  // Revoke handlers
  list.querySelectorAll('.btn-revoke').forEach(function (btn) {
    btn.addEventListener('click', function (e) {
      e.stopPropagation();
      var idx = parseInt(this.getAttribute('data-index'));
      DEMO_ACCESS.splice(idx, 1);
      renderAccessList();
    });
  });
}

// ─── Render Logs ───

function renderLogs(filter) {
  var output = document.getElementById('log-output');
  var filtered = filter && filter !== 'all'
    ? DEMO_LOGS.filter(function (l) {
        if (filter === 'error') return l.level === 'error' || l.level === 'warn';
        return l.level === filter;
      })
    : DEMO_LOGS;

  output.innerHTML = filtered.map(function (l) {
    return '<div class="log-line ' + l.level + '">' +
      '<span class="log-ts">' + l.ts + '</span>' +
      '<span class="log-msg">' + l.msg + '</span>' +
      '</div>';
  }).join('');
}

// ─── Log Filters ───

var filterBtns = document.querySelectorAll('.filter-btn');
filterBtns.forEach(function (btn) {
  btn.addEventListener('click', function () {
    filterBtns.forEach(function (b) { b.classList.remove('active'); });
    this.classList.add('active');
    renderLogs(this.getAttribute('data-filter'));
  });
});

document.getElementById('btn-clear-logs').addEventListener('click', function () {
  DEMO_LOGS.length = 0;
  renderLogs('all');
});

// ─── Drag and Drop ───

var dropZone = document.getElementById('drop-zone');

dropZone.addEventListener('dragover', function (e) {
  e.preventDefault();
  e.stopPropagation();
  this.classList.add('drag-over');
});

dropZone.addEventListener('dragleave', function (e) {
  e.preventDefault();
  e.stopPropagation();
  this.classList.remove('drag-over');
});

dropZone.addEventListener('drop', function (e) {
  e.preventDefault();
  e.stopPropagation();
  this.classList.remove('drag-over');

  var files = e.dataTransfer.files;
  for (var i = 0; i < files.length; i++) {
    var f = files[i];
    var isDir = !f.type && f.size === 0; // Heuristic
    var now = new Date();
    var timeStr = 'Today, ' + now.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
    DEMO_ACCESS.unshift({
      path: f.name + (isDir ? '/' : ''),
      type: isDir ? 'dir' : 'file',
      granted: timeStr,
    });
  }
  renderAccessList();

  // Flash the file access nav
  var filesNav = document.querySelector('[data-tab="files"]');
  filesNav.style.background = 'var(--accent-mid)';
  setTimeout(function () { filesNav.style.background = ''; }, 800);
});

// Click to add (mockup — just adds a demo entry)
dropZone.addEventListener('click', function () {
  var timeStr = 'Today, ' + new Date().toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
  DEMO_ACCESS.unshift({
    path: '~/Documents/new-file-' + Date.now() + '.txt',
    type: 'file',
    granted: timeStr,
  });
  renderAccessList();
});

document.getElementById('btn-add-file').addEventListener('click', function () {
  var timeStr = 'Today, ' + new Date().toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
  DEMO_ACCESS.unshift({
    path: '~/Documents/selected-file.pdf',
    type: 'file',
    granted: timeStr,
  });
  renderAccessList();
});

document.getElementById('btn-add-folder').addEventListener('click', function () {
  var timeStr = 'Today, ' + new Date().toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
  DEMO_ACCESS.unshift({
    path: '~/Projects/new-project/',
    type: 'dir',
    granted: timeStr,
  });
  renderAccessList();
});

// ─── Settings: Role Toggle ───

var roleBtns = document.querySelectorAll('.role-btn');
roleBtns.forEach(function (btn) {
  btn.addEventListener('click', function () {
    roleBtns.forEach(function (b) { b.classList.remove('active'); });
    this.classList.add('active');
    var role = this.getAttribute('data-role');
    if (role === 'simple') {
      document.body.classList.add('simple-mode');
    } else {
      document.body.classList.remove('simple-mode');
    }
  });
});

// Start in simple mode
document.body.classList.add('simple-mode');

// ─── Settings: Chip toggles ───

document.querySelectorAll('.chip').forEach(function (chip) {
  chip.addEventListener('click', function () {
    this.classList.toggle('active');
  });
});

// ─── Settings: Save/Reconnect (mockup) ───

document.getElementById('btn-save-settings').addEventListener('click', function () {
  this.textContent = 'Saved ✓';
  this.style.background = 'var(--success)';
  var self = this;
  setTimeout(function () {
    self.textContent = 'Save';
    self.style.background = '';
  }, 1500);
});

document.getElementById('btn-reconnect').addEventListener('click', function () {
  this.textContent = 'Reconnecting...';
  this.disabled = true;
  var self = this;
  setTimeout(function () {
    self.textContent = 'Reconnect';
    self.disabled = false;
  }, 2000);
});

// ─── Uptime Counter ───

var uptimeStart = Date.now() - (2 * 60 + 34) * 60 * 1000; // 2h34m ago

function updateUptime() {
  var diff = Date.now() - uptimeStart;
  var h = Math.floor(diff / 3600000);
  var m = Math.floor((diff % 3600000) / 60000);
  document.getElementById('uptime').textContent = h + 'h ' + m + 'm';
}

setInterval(updateUptime, 60000);

// ─── Init ───

renderActivity();
renderAccessList();
renderLogs('all');
