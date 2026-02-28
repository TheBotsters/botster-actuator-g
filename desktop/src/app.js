// Actuator G Desktop — plain JS, no frameworks
// Uses Tauri's global API (withGlobalTauri: true in config)

var invoke = window.__TAURI__.core.invoke;

var elements = {
  statusBadge: document.getElementById('status-badge'),
  brokerUrl: document.getElementById('broker-url'),
  brokerToken: document.getElementById('broker-token'),
  actuatorId: document.getElementById('actuator-id'),
  btnStart: document.getElementById('btn-start'),
  btnStop: document.getElementById('btn-stop'),
  logOutput: document.getElementById('log-output'),
};

// --- Status management ---

function setStatus(status) {
  var badge = elements.statusBadge;
  badge.className = 'badge';

  switch (status) {
    case 'Connected':
      badge.textContent = 'Connected';
      badge.classList.add('connected');
      elements.btnStart.disabled = true;
      elements.btnStop.disabled = false;
      setFieldsDisabled(true);
      break;
    case 'Connecting':
      badge.textContent = 'Connecting...';
      badge.classList.add('connecting');
      elements.btnStart.disabled = true;
      elements.btnStop.disabled = false;
      setFieldsDisabled(true);
      break;
    case 'Disconnected':
      badge.textContent = 'Disconnected';
      badge.classList.add('disconnected');
      elements.btnStart.disabled = false;
      elements.btnStop.disabled = true;
      setFieldsDisabled(false);
      break;
    default:
      badge.textContent = 'Error';
      badge.classList.add('error');
      elements.btnStart.disabled = false;
      elements.btnStop.disabled = true;
      setFieldsDisabled(false);
  }
}

function setFieldsDisabled(disabled) {
  elements.brokerUrl.disabled = disabled;
  elements.brokerToken.disabled = disabled;
  elements.actuatorId.disabled = disabled;
}

// --- Logging ---

function appendLog(text, type) {
  var line = document.createElement('div');
  line.className = 'log-line';
  if (type) line.classList.add(type);
  line.textContent = new Date().toLocaleTimeString() + ' ' + text;
  elements.logOutput.appendChild(line);
  elements.logOutput.scrollTop = elements.logOutput.scrollHeight;
}

// --- Actions ---

elements.btnStart.addEventListener('click', function () {
  var brokerUrl = elements.brokerUrl.value.trim();
  var brokerToken = elements.brokerToken.value.trim();
  var actuatorId = elements.actuatorId.value.trim();

  if (!brokerUrl || !brokerToken || !actuatorId) {
    appendLog('All fields are required', 'error');
    return;
  }

  setStatus('Connecting');
  appendLog('Starting actuator...', 'info');

  invoke('start_actuator', {
    brokerUrl: brokerUrl,
    brokerToken: brokerToken,
    actuatorId: actuatorId,
  })
    .then(function (result) {
      setStatus('Connected');
      appendLog(result, 'info');
    })
    .catch(function (err) {
      setStatus('Error');
      appendLog('Failed: ' + err, 'error');
    });
});

elements.btnStop.addEventListener('click', function () {
  appendLog('Stopping actuator...', 'info');

  invoke('stop_actuator')
    .then(function (result) {
      setStatus('Disconnected');
      appendLog(result, 'info');
    })
    .catch(function (err) {
      appendLog('Stop failed: ' + err, 'error');
    });
});

// --- Polling ---

function pollStatus() {
  invoke('get_status')
    .then(function (data) {
      if (data && data.status) {
        var s = typeof data.status === 'string' ? data.status : data.status.Error ? 'Error' : 'Disconnected';
        setStatus(s);
      }
    })
    .catch(function () {
      // Ignore poll errors
    });
}

// Poll every 5 seconds
setInterval(pollStatus, 5000);

// --- Load saved config from localStorage ---

function loadConfig() {
  elements.brokerUrl.value = localStorage.getItem('actuator_broker_url') || '';
  elements.actuatorId.value = localStorage.getItem('actuator_id') || '';
  // Never persist token
}

function saveConfig() {
  localStorage.setItem('actuator_broker_url', elements.brokerUrl.value);
  localStorage.setItem('actuator_id', elements.actuatorId.value);
}

elements.brokerUrl.addEventListener('change', saveConfig);
elements.actuatorId.addEventListener('change', saveConfig);

// Init
loadConfig();
appendLog('Actuator G desktop ready', 'info');
