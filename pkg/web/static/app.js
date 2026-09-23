// DDS238-2 ZN/S Multi-Meter Modbus Simulator Web Frontend

let ws = null;
let allMeters = [];
let selectedMeterId = 1;
let isUpdatingControls = false;
let isSerialConnected = false;

// Initialize
window.addEventListener('DOMContentLoaded', () => {
  connectWebSocket();
  setupEventListeners();
  loadComPorts();
  checkSerialStatus();
  fetchMeters();
  // Run default voltage query to populate query sandbox & snippet on load
  sendPreset('voltage');
});

// WebSocket Connection
function connectWebSocket() {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsUrl = `${protocol}//${window.location.host}/ws`;
  
  ws = new WebSocket(wsUrl);

  const statusBadge = document.getElementById('ws-status');
  const statusText = document.getElementById('ws-status-text');

  ws.onopen = () => {
    statusBadge.classList.add('active');
    statusText.textContent = 'Web UI Active';
  };

  ws.onclose = () => {
    statusBadge.classList.remove('active');
    statusText.textContent = 'Reconnecting...';
    setTimeout(connectWebSocket, 1500);
  };

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data);
      if (msg.type === 'meters') {
        handleMetersUpdate(msg.data);
      } else if (msg.type === 'state') {
        renderSingleState(msg.data);
      } else if (msg.type === 'packet') {
        appendPacketLog(msg.packet);
      }
    } catch (e) {
      console.error('WS Parse Error:', e);
    }
  };
}

// Fetch all active meters from API
async function fetchMeters() {
  try {
    const res = await fetch('/api/meters');
    const data = await res.json();
    handleMetersUpdate(data);
  } catch (err) {
    console.error('Failed to fetch meters:', err);
  }
}

function handleMetersUpdate(meters) {
  if (!Array.isArray(meters) || meters.length === 0) return;
  allMeters = meters;

  // If selectedMeterId not in list, fallback to first meter
  const exists = allMeters.some(m => m.station_address === selectedMeterId);
  if (!exists) {
    selectedMeterId = allMeters[0].station_address;
  }

  renderMeterTabs();

  const current = allMeters.find(m => m.station_address === selectedMeterId) || allMeters[0];
  renderSingleState(current);
}

// Render the top meter selector tab bar
function renderMeterTabs() {
  const container = document.getElementById('meter-tabs-list');
  if (!container) return;

  container.innerHTML = '';
  allMeters.forEach(m => {
    const id = m.station_address;
    const name = m.name || `Meter #${id}`;
    const isActive = id === selectedMeterId;
    const isSolar = name.toLowerCase().includes('solar');
    const icon = isSolar ? '☀️' : (id === 1 ? '⚡' : '🔌');

    const tab = document.createElement('div');
    tab.className = `meter-tab ${isActive ? 'active' : ''}`;
    tab.onclick = () => selectMeter(id);

    const vStr = (m.voltage_v || 230).toFixed(0);
    const iStr = (m.current_a || 0).toFixed(1);
    const pStr = (m.active_power_w || 0).toFixed(0);

    let deleteBtnHtml = '';
    if (allMeters.length > 1) {
      deleteBtnHtml = `<span class="meter-tab-del" title="Remove meter #${id}" onclick="event.stopPropagation(); removeMeter(${id})">✕</span>`;
    }

    tab.innerHTML = `
      <span>${icon}</span>
      <span class="meter-tab-id">#${id}</span>
      <span class="meter-tab-name">${escapeHtml(name)}</span>
      <span class="meter-tab-preview">${vStr}V | ${pStr}W</span>
      ${deleteBtnHtml}
    `;

    container.appendChild(tab);
  });
}

function selectMeter(id) {
  selectedMeterId = id;
  renderMeterTabs();

  // Update query form default slave ID
  const querySlaveInput = document.getElementById('query-slave');
  if (querySlaveInput) {
    querySlaveInput.value = id;
  }

  const current = allMeters.find(m => m.station_address === id);
  if (current) {
    renderSingleState(current);
    updateControlsForm(current);
  }
}

// Render Telemetry Gauges for Selected Meter
function renderSingleState(state) {
  if (!state || state.station_address !== selectedMeterId) return;

  // Header badges
  document.getElementById('badge-slave-id').textContent = state.station_address || '1';
  const baudMap = { 1: '9600', 2: '4800', 3: '2400', 4: '1200' };
  document.getElementById('badge-baud').textContent = baudMap[state.baud_rate_code] || '9600';

  // Panel labels
  const targetLabel = `${state.name || 'Meter'} (#${state.station_address})`;
  const codeTarget = document.getElementById('c-code-target-name');
  if (codeTarget) codeTarget.textContent = targetLabel;
  const ctrlTarget = document.getElementById('ctrl-panel-meter-name');
  if (ctrlTarget) ctrlTarget.textContent = targetLabel;

  // Metrics
  document.getElementById('val-voltage').textContent = state.voltage_v.toFixed(1);
  document.getElementById('raw-voltage').textContent = `Raw: ${Math.round(state.voltage_v * 10)} (0.1 V)`;

  document.getElementById('val-current').textContent = state.current_a.toFixed(2);
  document.getElementById('raw-current').textContent = `Raw: ${Math.round(state.current_a * 100)} (0.01 A)`;

  const pValEl = document.getElementById('val-active-power');
  pValEl.textContent = state.active_power_w.toFixed(0);
  if (state.active_power_w < 0) {
    pValEl.style.color = 'var(--accent-amber)'; // Solar export indication
  } else {
    pValEl.style.color = 'var(--text-main)';
  }

  document.getElementById('val-reactive-power').textContent = state.reactive_power_var.toFixed(0);
  document.getElementById('val-pf').textContent = state.power_factor.toFixed(3);
  document.getElementById('val-freq').textContent = state.frequency_hz.toFixed(2);
  
  // Energy cards
  const importEl = document.getElementById('val-import-energy');
  if (importEl) importEl.textContent = (state.import_energy_kwh || 0).toFixed(2);
  const exportEl = document.getElementById('val-export-energy');
  if (exportEl) exportEl.textContent = (state.export_energy_kwh || 0).toFixed(2);
  document.getElementById('val-total-energy').textContent = (state.total_energy_kwh || 0).toFixed(2);

  // Relay
  const relayEl = document.getElementById('val-relay');
  const relayBtn = document.getElementById('btn-quick-relay');
  if (state.relay_state) {
    relayEl.textContent = 'ON';
    relayEl.style.color = 'var(--accent-emerald)';
    relayBtn.textContent = 'Turn Relay OFF';
    relayBtn.className = 'btn btn-danger';
  } else {
    relayEl.textContent = 'OFF';
    relayEl.style.color = 'var(--accent-rose)';
    relayBtn.textContent = 'Turn Relay ON';
    relayBtn.className = 'btn btn-success';
  }

  // Update controls if user isn't actively sliding
  if (!isUpdatingControls) {
    updateControlsForm(state);
  }
}

function updateControlsForm(state) {
  const nameInput = document.getElementById('ctrl-meter-name');
  if (nameInput) nameInput.value = state.name || `Meter #${state.station_address}`;

  document.getElementById('ctrl-voltage').value = state.target_voltage_v || state.voltage_v || 230;
  document.getElementById('disp-ctrl-voltage').textContent = (state.target_voltage_v || state.voltage_v || 230).toFixed(1) + ' V';

  document.getElementById('ctrl-current').value = state.target_current_a || state.current_a || 5.25;
  document.getElementById('disp-ctrl-current').textContent = (state.target_current_a || state.current_a || 5.25).toFixed(2) + ' A';

  document.getElementById('ctrl-pf').value = state.target_pf || state.power_factor || 0.995;
  document.getElementById('disp-ctrl-pf').textContent = (state.target_pf || state.power_factor || 0.995).toFixed(3);

  document.getElementById('ctrl-freq').value = state.target_frequency_hz || state.frequency_hz || 50;
  document.getElementById('disp-ctrl-freq').textContent = (state.target_frequency_hz || state.frequency_hz || 50).toFixed(2) + ' Hz';

  document.getElementById('ctrl-noise').checked = state.simulate_noise ?? true;
  document.getElementById('ctrl-dynamic-energy').checked = state.dynamic_energy ?? true;
}

// Serial Port Management
async function loadComPorts() {
  const select = document.getElementById('select-com-port');
  select.innerHTML = '<option value="">Scanning ports...</option>';
  try {
    const res = await fetch('/api/ports');
    const data = await res.json();
    select.innerHTML = '';
    
    if (!data.ports || data.ports.length === 0) {
      select.innerHTML = '<option value="">No COM Ports Found</option>';
      return;
    }

    data.ports.forEach(p => {
      const opt = document.createElement('option');
      opt.value = p;
      opt.textContent = p;
      select.appendChild(opt);
    });
  } catch (err) {
    console.error('Failed to load COM ports:', err);
    select.innerHTML = '<option value="">Scan failed</option>';
  }
}

async function checkSerialStatus() {
  try {
    const res = await fetch('/api/serial/status');
    const data = await res.json();
    renderSerialStatus(data);
  } catch (err) {
    console.error('Failed to get serial status:', err);
  }
}

function renderSerialStatus(status) {
  isSerialConnected = status.connected;
  const pill = document.getElementById('serial-status-pill');
  const label = document.getElementById('serial-status-label');
  const btn = document.getElementById('btn-serial-toggle');
  const select = document.getElementById('select-com-port');
  const baudSelect = document.getElementById('select-baud');
  const tcpBadge = document.getElementById('badge-tcp-port');

  if (status.tcp_addr && tcpBadge) {
    tcpBadge.textContent = status.tcp_addr;
  }
  const tcpStatusPill = document.getElementById('tcp-status');
  if (tcpStatusPill) {
    if (status.tcp_active) {
      tcpStatusPill.className = 'status-badge active clickable';
      tcpStatusPill.title = `Modbus TCP Bridge Active on ${status.tcp_addr} (Click to change)`;
    } else {
      tcpStatusPill.className = 'status-badge clickable';
      tcpStatusPill.title = `Modbus TCP Bridge: ${status.tcp_addr} ${status.tcp_err ? '(' + status.tcp_err + ')' : '(Inactive)'} (Click to change)`;
    }
  }

  if (status.connected) {
    pill.className = 'serial-status-pill connected';
    label.textContent = `Connected: ${status.port} @ ${status.baud} Bd`;
    btn.className = 'btn btn-danger';
    btn.textContent = '🔌 Disconnect';
    select.disabled = true;
    baudSelect.disabled = true;
  } else {
    pill.className = 'serial-status-pill disconnected';
    label.textContent = 'UART Disconnected';
    btn.className = 'btn btn-primary';
    btn.textContent = '⚡ Connect Port';
    select.disabled = false;
    baudSelect.disabled = false;
  }
}

async function toggleSerialConnection() {
  const btn = document.getElementById('btn-serial-toggle');
  const select = document.getElementById('select-com-port');
  const baud = parseInt(document.getElementById('select-baud').value) || 9600;
  const port = select.value;

  if (isSerialConnected) {
    btn.textContent = 'Disconnecting...';
    try {
      const res = await fetch('/api/serial/disconnect', { method: 'POST' });
      const status = await res.json();
      renderSerialStatus(status);
    } catch (err) {
      alert('Failed to disconnect: ' + err);
    }
  } else {
    if (!port) {
      alert('Please select or plug in a serial COM port first!');
      return;
    }
    btn.textContent = 'Connecting...';
    try {
      const res = await fetch('/api/serial/connect', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ port: port, baud: baud })
      });
      if (!res.ok) {
        const errText = await res.text();
        alert('Connection error: ' + errText);
        checkSerialStatus();
        return;
      }
      const status = await res.json();
      renderSerialStatus(status);
    } catch (err) {
      alert('Failed to connect: ' + err);
      checkSerialStatus();
    }
  }
}

// Packet Logger
const MAX_LOGS = 60;
function appendPacketLog(packet) {
  const container = document.getElementById('packet-log-stream');
  if (!container) return;

  if (container.firstElementChild && container.firstElementChild.textContent.includes('Waiting for Modbus')) {
    container.innerHTML = '';
  }

  const item = document.createElement('div');
  const isRx = packet.direction.startsWith('RX');
  item.className = `log-item ${isRx ? 'rx' : 'tx'}`;

  const timeStr = new Date(packet.timestamp).toLocaleTimeString();
  const dirBadgeClass = isRx ? 'rx' : 'tx';
  const dirLabel = isRx ? 'RX (From STM32)' : 'TX (To STM32)';
  const crcBadge = packet.crc_valid ? '<span class="log-badge" style="background:rgba(16,185,129,0.2);color:var(--accent-emerald);">CRC OK</span>' 
                                    : '<span class="log-badge" style="background:rgba(244,63,94,0.2);color:var(--accent-rose);">CRC ERR</span>';
  
  const deviceBadge = packet.device_name ? `<span class="log-badge" style="background:rgba(0,242,254,0.15);color:var(--accent-cyan);">${escapeHtml(packet.device_name)}</span>` : '';

  let detailsHtml = '';
  if (packet.byte_details && packet.byte_details.length > 0) {
    detailsHtml = '<div class="byte-inspector-tags" style="margin-top:0.4rem;">' +
      packet.byte_details.map(d => `<span class="byte-tag"><span class="byte-hex">${d.hex}</span><span class="byte-desc">${d.label}</span></span>`).join('') +
      '</div>';
  }

  item.innerHTML = `
    <div class="log-item-header">
      <span class="log-badge ${dirBadgeClass}">${dirLabel}</span>
      ${deviceBadge}
      <span class="log-badge" style="background:rgba(255,255,255,0.05);">${packet.length} bytes</span>
      ${crcBadge}
      <span class="log-time">${timeStr}</span>
    </div>
    <div class="log-raw-hex">${packet.raw_hex}</div>
    <div class="log-desc">${escapeHtml(packet.description)}</div>
    ${detailsHtml}
  `;

  container.insertBefore(item, container.firstChild);

  while (container.children.length > MAX_LOGS) {
    container.removeChild(container.lastChild);
  }
}

function clearLogs() {
  const container = document.getElementById('packet-log-stream');
  container.innerHTML = '<div style="color:var(--text-dim); font-size:0.8rem; text-align:center; padding:2rem;">Waiting for Modbus RTU communication over UART or TCP...</div>';
}

// Client Reader Queries & Sandbox
async function sendPreset(presetName) {
  const slaveId = parseInt(document.getElementById('query-slave').value) || selectedMeterId;
  await executeQuery({ preset: presetName, slave_id: slaveId });
}

async function sendCustomQuery() {
  const slaveId = parseInt(document.getElementById('query-slave').value) || selectedMeterId;
  const startRaw = document.getElementById('query-start').value.trim();
  const count = parseInt(document.getElementById('query-count').value) || 1;

  let startAddr = 0;
  if (startRaw.startsWith('0x') || startRaw.startsWith('0X')) {
    startAddr = parseInt(startRaw, 16);
  } else {
    startAddr = parseInt(startRaw, 10);
  }

  await executeQuery({
    slave_id: slaveId,
    start_addr: startAddr,
    reg_count: count
  });
}

async function sendRawHex() {
  const hex = document.getElementById('query-raw-hex').value.trim();
  if (!hex) {
    alert('Please enter a hex string (e.g. 01 03 00 0C 00 01)');
    return;
  }
  await executeQuery({ hex_command: hex });
}

async function executeQuery(payload) {
  try {
    const res = await fetch('/api/query', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });

    if (!res.ok) {
      const err = await res.text();
      alert('Query failed: ' + err);
      return;
    }

    const data = await res.json();
    renderQueryResult(data);
    updateCCodeBlock(data);
  } catch (err) {
    console.error('Query execution error:', err);
  }
}

function renderQueryResult(result) {
  const box = document.getElementById('query-result-box');
  box.style.display = 'block';

  document.getElementById('res-request-hex').textContent = result.request_hex || '-';
  document.getElementById('res-response-hex').textContent = result.response_hex || '(No response / Timeout)';

  const reqTags = document.getElementById('res-req-tags');
  reqTags.innerHTML = (result.request_details || []).map(d =>
    `<span class="byte-tag"><span class="byte-hex">${d.hex}</span><span class="byte-desc">${d.label}</span></span>`
  ).join('');

  const respTags = document.getElementById('res-resp-tags');
  respTags.innerHTML = (result.response_details || []).map(d =>
    `<span class="byte-tag"><span class="byte-hex">${d.hex}</span><span class="byte-desc">${d.label}</span></span>`
  ).join('');

  const parsedTags = document.getElementById('res-parsed-tags');
  if (result.parsed_values && Object.keys(result.parsed_values).length > 0) {
    parsedTags.innerHTML = Object.entries(result.parsed_values).map(([k, v]) =>
      `<div class="parsed-pill"><span class="parsed-pill-key">${k}:</span><span class="parsed-pill-val">${v}</span></div>`
    ).join('');
    document.getElementById('res-parsed-box').style.display = 'block';
  } else {
    document.getElementById('res-parsed-box').style.display = 'none';
  }
}

// STM32 HAL C Code Generator
function updateCCodeBlock(result) {
  const codeEl = document.getElementById('c-code-block');
  if (!result || !result.request_hex) return;

  const hexBytes = result.request_hex.split(' ');
  const cArray = hexBytes.map(b => '0x' + b).join(', ');
  const respLen = result.response_hex ? result.response_hex.split(' ').length : 7;
  const slave = parseInt(document.getElementById('query-slave').value) || selectedMeterId;

  const code = `// =================================================================
// STM32 HAL Modbus RTU Query for DDS238-2 Energy Meter (Unit ID: ${slave})
// UART Config: 9600 Baud, 8 Data Bits, 1 Stop Bit, No Parity (8N1)
// =================================================================

// 1. Modbus RTU Command Frame with CRC-16 (Modbus):
uint8_t modbus_req[] = { ${cArray} };
uint8_t modbus_rx[${respLen}];

// 2. Transmit request via UART (USART2 / HAL UART):
if (HAL_UART_Transmit(&huart2, modbus_req, sizeof(modbus_req), 100) == HAL_OK) {
    
    // 3. Receive response frame:
    if (HAL_UART_Receive(&huart2, modbus_rx, sizeof(modbus_rx), 200) == HAL_OK) {
        
        // 4. Verify Server ID and Function Code:
        if (modbus_rx[0] == 0x${slave.toString(16).padStart(2, '0').toUpperCase()} && modbus_rx[1] == 0x03) {
            uint8_t byte_count = modbus_rx[2];
            
            // Example: Parse 16-bit Voltage (0.1 V scale):
            uint16_t raw_v = (modbus_rx[3] << 8) | modbus_rx[4];
            float voltage = raw_v / 10.0f;
            printf("Meter #${slave} Voltage: %.1f V\\r\\n", voltage);
        }
    }
}`;

  codeEl.textContent = code;
}

function copyCCode() {
  const code = document.getElementById('c-code-block').textContent;
  navigator.clipboard.writeText(code).then(() => {
    alert('STM32 C Code copied to clipboard!');
  }).catch(err => {
    console.error('Failed to copy code:', err);
  });
}

// Dynamic Simulation & Fault Injection Controls
function setupEventListeners() {
  // Toggle relay quick button
}

function updateControl() {
  isUpdatingControls = true;
  const v = parseFloat(document.getElementById('ctrl-voltage').value);
  const i = parseFloat(document.getElementById('ctrl-current').value);
  const pf = parseFloat(document.getElementById('ctrl-pf').value);
  const f = parseFloat(document.getElementById('ctrl-freq').value);
  const noise = document.getElementById('ctrl-noise').checked;
  const dyn = document.getElementById('ctrl-dynamic-energy').checked;

  document.getElementById('disp-ctrl-voltage').textContent = v.toFixed(1) + ' V';
  document.getElementById('disp-ctrl-current').textContent = i.toFixed(2) + ' A';
  document.getElementById('disp-ctrl-pf').textContent = pf.toFixed(3);
  document.getElementById('disp-ctrl-freq').textContent = f.toFixed(2) + ' Hz';

  debouncedSendControl({
    meter_id: selectedMeterId,
    target_voltage_v: v,
    target_current_a: i,
    target_pf: pf,
    target_frequency_hz: f,
    simulate_noise: noise,
    dynamic_energy: dyn
  });

  setTimeout(() => { isUpdatingControls = false; }, 400);
}

function updateMeterName() {
  const nameInput = document.getElementById('ctrl-meter-name');
  const newName = nameInput.value.trim();
  if (!newName) return;

  sendControl({
    meter_id: selectedMeterId,
    name: newName
  });
}

function setVoltage(v) {
  document.getElementById('ctrl-voltage').value = v;
  updateControl();
}

function setCurrent(i) {
  document.getElementById('ctrl-current').value = i;
  updateControl();
}

async function toggleRelay() {
  const current = allMeters.find(m => m.station_address === selectedMeterId);
  const newState = !(current && current.relay_state);
  
  await sendControl({
    meter_id: selectedMeterId,
    relay_state: newState
  });
}

let debounceTimer = null;
function debouncedSendControl(payload) {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => {
    sendControl(payload);
  }, 100);
}

async function sendControl(payload) {
  try {
    const res = await fetch('/api/control', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    if (res.ok) {
      const data = await res.json();
      renderSingleState(data);
    }
  } catch (err) {
    console.error('Control error:', err);
  }
}

// Add Virtual Meter Modal
function openAddMeterModal() {
  const modal = document.getElementById('modal-add-meter');
  const idInput = document.getElementById('input-new-meter-id');
  const errBox = document.getElementById('add-meter-error');
  errBox.style.display = 'none';

  // Suggest next unused ID
  const existingIds = allMeters.map(m => m.station_address);
  let nextId = 2;
  while (existingIds.includes(nextId) && nextId < 247) {
    nextId++;
  }
  idInput.value = nextId;

  modal.style.display = 'flex';
}

function closeAddMeterModal() {
  const modal = document.getElementById('modal-add-meter');
  if (modal) modal.style.display = 'none';
}

function setAddMeterRole(id, name) {
  document.getElementById('input-new-meter-id').value = id;
  document.getElementById('input-new-meter-name').value = name;
}

async function submitAddMeter() {
  const id = parseInt(document.getElementById('input-new-meter-id').value);
  const name = document.getElementById('input-new-meter-name').value.trim();
  const btn = document.getElementById('btn-save-new-meter');
  const errBox = document.getElementById('add-meter-error');

  if (isNaN(id) || id < 1 || id > 247) {
    errBox.textContent = '❌ Server ID must be a number between 1 and 247';
    errBox.style.display = 'block';
    return;
  }

  btn.disabled = true;
  btn.textContent = 'Creating...';
  errBox.style.display = 'none';

  try {
    const res = await fetch('/api/meters/add', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id: id, name: name })
    });

    if (!res.ok) {
      const errText = await res.text();
      errBox.textContent = '❌ ' + errText;
      errBox.style.display = 'block';
      return;
    }

    closeAddMeterModal();
    await fetchMeters();
    selectMeter(id);
  } catch (err) {
    errBox.textContent = '❌ Request failed: ' + err;
    errBox.style.display = 'block';
  } finally {
    btn.disabled = false;
    btn.textContent = 'Add Meter to Bus';
  }
}

async function removeMeter(id) {
  if (!confirm(`Are you sure you want to remove simulated Meter #${id}?`)) {
    return;
  }

  try {
    const res = await fetch('/api/meters/remove', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id: id })
    });

    if (!res.ok) {
      const err = await res.text();
      alert('Failed to remove meter: ' + err);
      return;
    }

    await fetchMeters();
  } catch (err) {
    alert('Failed to remove meter: ' + err);
  }
}

// Modbus TCP Configuration Modal
function openTcpModal() {
  const modal = document.getElementById('modal-tcp');
  const input = document.getElementById('input-tcp-port');
  const currentBadge = document.getElementById('badge-tcp-port');
  const errBox = document.getElementById('tcp-modal-error');
  
  errBox.style.display = 'none';
  errBox.textContent = '';
  
  if (currentBadge && currentBadge.textContent) {
    input.value = currentBadge.textContent.replace(':', '');
  }
  
  modal.style.display = 'flex';
  input.focus();
}

function closeTcpModal() {
  const modal = document.getElementById('modal-tcp');
  if (modal) modal.style.display = 'none';
}

function setTcpPortPreset(port) {
  const input = document.getElementById('input-tcp-port');
  if (input) {
    input.value = port;
  }
}

async function submitTcpPortChange() {
  const input = document.getElementById('input-tcp-port');
  const btn = document.getElementById('btn-save-tcp-port');
  const errBox = document.getElementById('tcp-modal-error');
  const port = input.value.trim();

  btn.disabled = true;
  btn.textContent = 'Binding...';
  errBox.style.display = 'none';

  try {
    const res = await fetch('/api/tcp/update', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ port: port })
    });

    if (!res.ok) {
      const errText = await res.text();
      errBox.textContent = '❌ ' + errText;
      errBox.style.display = 'block';
      return;
    }

    closeTcpModal();
    checkSerialStatus();
  } catch (err) {
    errBox.textContent = '❌ Request failed: ' + err;
    errBox.style.display = 'block';
  } finally {
    btn.disabled = false;
    btn.textContent = 'Apply & Listen';
  }
}

function escapeHtml(str) {
  if (!str) return '';
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}
