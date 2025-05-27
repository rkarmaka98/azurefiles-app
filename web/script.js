const API_BASE = 'http://localhost:8080/api';

async function fetchJSON(path) {
  const res = await fetch(`${API_BASE}/${path}`, {
    method: 'GET',
    mode: 'cors',
    headers: { 'Content-Type': 'application/json' }
  });
  if (!res.ok) throw new Error(`Failed ${path}: ${res.status}`);
  return res.json();
}

function renderShares(shares, anomalies) {
  const tbody = document.querySelector('#shares-table tbody');
  tbody.innerHTML = '';
  shares.forEach(s => {
    const tr = document.createElement('tr');
    ['name','quotaGB','iops','bandwidthMiB','latencyMs','transactions'].forEach(key => {
      const td = document.createElement('td');
      td.textContent = (typeof s[key] === 'number')
        ? (key==='transactions' ? s[key].toFixed(0) : s[key].toFixed(1))
        : s[key];
      tr.appendChild(td);
    });
    const td = document.createElement('td');
    if (anomalies[s.name]) {
      td.textContent = `⚠ ${anomalies[s.name]}`;
      td.classList.add('alert');
    } else {
      td.textContent = 'OK';
      td.classList.add('ok');
    }
    tr.appendChild(td);
    tbody.appendChild(tr);
  });
}

async function initDashboard() {
  try {
    const [shares, anomalies] = await Promise.all([
      fetchJSON('shares'),
      fetchJSON('anomalies')
    ]);
    renderShares(shares, anomalies);
  } catch (e) { console.error(e); }
}

initDashboard();
setInterval(async () => {
  try {
    const [shares, anomalies] = await Promise.all([
      fetchJSON('shares'),
      fetchJSON('anomalies')
    ]);
    renderShares(shares, anomalies);
  } catch (e) { console.error(e); }
}, 60_000);
