let selectedAddress = "";
let lastNoticeId = 0;
let syncing = false;
const escapeHtml = value => String(value ?? "").replace(/[&<>'"]/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;',"'":'&#39;','"':'&quot;'}[c]));

async function syncState() {
  if (syncing) return;
  syncing = true;
  try {
    const s = await appState();
    document.getElementById('status').textContent = s.status || '未连接';
    document.getElementById('battery').textContent = s.battery || '手机电量：等待手机状态';
    const pill = document.getElementById('connectionPill');
    const connected = (s.status || '').includes('已连接');
    pill.classList.toggle('connected', connected);
    pill.querySelector('span').textContent = connected ? '蓝牙已连接' : '等待连接';
    if (!selectedAddress) selectedAddress = s.selectedAddress || s.devices?.[0]?.address || '';
    renderDevices(s.devices || []);
    renderLogs(s.logs || []);
    for (const n of s.notices || []) if (n.id > lastNoticeId) { showToast(n.title, n.body); lastNoticeId = n.id; }
  } catch (e) { document.getElementById('status').textContent = '界面通信暂时中断'; }
  finally { syncing = false; }
}

function renderDevices(devices) {
  const root = document.getElementById('devices');
  if (!devices.length) { root.innerHTML = '<div class="empty">没有发现已配对设备，请先在 Windows 设置中配对手机</div>'; return; }
  root.innerHTML = devices.map(d => `<button class="device-card ${d.address===selectedAddress?'selected':''}" onclick="selectDevice('${d.address}')"><strong>${escapeHtml(d.name)}</strong><small>${escapeHtml(d.address)}</small>${d.address===selectedAddress?'<em></em>':''}</button>`).join('');
}
function renderLogs(logs) {
  const root = document.getElementById('logs');
  root.innerHTML = logs.length ? logs.slice().reverse().map(line => { const m=line.match(/^\[([^\]]+)\]\s*(.*)$/s); return `<div class="log"><i></i><time>${escapeHtml(m?.[1]||'')}</time><span>${escapeHtml(m?.[2]||line)}</span></div>`; }).join('') : '<div class="empty">还没有运行记录</div>';
}
function selectDevice(address){ selectedAddress=address; syncState(); }
async function refreshDevices(){ await appRefresh(); await syncState(); }
async function connectSelected(){ if(!selectedAddress){showToast('请选择手机','先选择一个已配对蓝牙设备');return;} await appConnect(selectedAddress); await syncState(); }
async function nativeStop(){ await appStop(); await syncState(); }
async function nativeTest(){ await appTest(); await syncState(); }
async function clearHistory(){ await appClear(); await syncState(); }
function nativeExit(){ appExit(); }
function showToast(title,body){ const t=document.createElement('div');t.className='toast';t.innerHTML=`<strong>${escapeHtml(title)}</strong><span>${escapeHtml(body)}</span>`;document.getElementById('toastRoot').appendChild(t);setTimeout(()=>t.remove(),6500); }
syncState(); setInterval(syncState, 900);
