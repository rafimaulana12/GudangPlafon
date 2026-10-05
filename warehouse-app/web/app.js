// State Aplikasi
let currentView = 'dashboard';
let currentLoanPeriod = new Date().toISOString().slice(0, 7);
let items = [];
let categories = [];
let locations = [];
let loans = [];
let unifiedLogs = [];

// Helper API Call
async function api(url, options = {}) {
  try {
    const res = await fetch(url, options);
    if (res.status === 401 && !url.includes('/api/auth/')) {
      showAuthScreen();
      throw new Error('Sesi telah berakhir. Silakan login kembali.');
    }
    if (!res.ok) {
      const err = await res.json().catch(() => ({ error: 'Permintaan gagal (' + res.status + ')' }));
      throw new Error(err.error || 'Terjadi kesalahan sistem');
    }
    return await res.json();
  } catch (e) {
    if (!url.includes('/api/auth/')) {
      showStatus(e.message, true);
    }
    throw e;
  }
}

// Utility Helpers
function escape(str) {
  if (!str) return '';
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}

function formatRupiah(num) {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0
  }).format(num || 0);
}

function formatPeriodName(periodStr) {
  if (!periodStr) return '';
  const [y, m] = periodStr.split('-');
  const months = [
    'Januari', 'Februari', 'Maret', 'April', 'Mei', 'Juni',
    'Juli', 'Agustus', 'September', 'Oktober', 'November', 'Desember'
  ];
  const idx = parseInt(m, 10) - 1;
  return `${months[idx] || m} ${y}`;
}

function showStatus(msg, isError = false) {
  const toast = document.getElementById('status-toast');
  toast.textContent = msg;
  toast.className = 'status-toast' + (isError ? ' error' : '');
  toast.classList.remove('hidden');
  clearTimeout(toast._timeout);
  toast._timeout = setTimeout(() => toast.classList.add('hidden'), 3500);
}

// Navigasi Tampilan (Drawer & Views)
const drawer = document.getElementById('nav-drawer');
const drawerBackdrop = document.getElementById('drawer-backdrop');
const btnMenuToggle = document.getElementById('btn-menu-toggle');
const btnDrawerClose = document.getElementById('btn-drawer-close');

function openDrawer() {
  if (drawer) drawer.classList.add('open');
  if (drawerBackdrop) drawerBackdrop.classList.remove('hidden');
}

function closeDrawer() {
  if (drawer) drawer.classList.remove('open');
  if (drawerBackdrop) drawerBackdrop.classList.add('hidden');
}

if (btnMenuToggle) btnMenuToggle.addEventListener('click', openDrawer);
if (btnDrawerClose) btnDrawerClose.addEventListener('click', closeDrawer);
if (drawerBackdrop) drawerBackdrop.addEventListener('click', closeDrawer);

window.switchView = switchView;
function switchView(viewName) {
  if (!viewName) return;
  currentView = viewName;

  // Sembunyikan semua view dan tampilkan view yang dituju
  document.querySelectorAll('.app-view').forEach(v => {
    v.classList.add('hidden');
    v.style.display = 'none';
  });

  const targetView = document.getElementById(`view-${viewName}`);
  if (targetView) {
    targetView.classList.remove('hidden');
    targetView.style.display = 'block';
  }

  // Update indikator aktif pada seluruh menu (Sidebar Desktop, Mobile Bottom Bar, dan Drawer)
  document.querySelectorAll('.drawer-link, .nav-btn, .bottom-tab-btn').forEach(l => {
    const v = l.getAttribute('data-view');
    if (v === viewName) {
      l.classList.add('active');
    } else {
      l.classList.remove('active');
    }
  });

  // Perbarui Hash URL jika berbeda
  if (window.location.hash !== '#' + viewName) {
    history.pushState(null, '', '#' + viewName);
  }

  closeDrawer();

  // Scroll to top
  window.scrollTo(0, 0);

  // Panggil data loader sesuai view
  if (viewName === 'dashboard') {
    loadCentralDashboard();
  } else if (viewName === 'loans') {
    loadLoans();
  } else if (viewName === 'inventory') {
    loadInventory();
    loadStats();
  } else if (viewName === 'audit') {
    loadUnifiedLogs();
  }
}

// Bind langsung ke setiap tombol navigasi
function setupNavBindings() {
  document.querySelectorAll('.nav-btn, .bottom-tab-btn, .drawer-link').forEach(btn => {
    const v = btn.getAttribute('data-view');
    btn.onclick = (e) => {
      e.preventDefault();
      e.stopPropagation();
      switchView(v);
    };
  });

  document.querySelectorAll('.btn-link-view').forEach(btn => {
    const target = btn.getAttribute('data-target');
    btn.onclick = (e) => {
      e.preventDefault();
      e.stopPropagation();
      switchView(target);
    };
  });
}
setupNavBindings();

// Listener perubahan URL hash saat pengguna menggunakan tombol Back/Forward browser
window.addEventListener('hashchange', () => {
  const hash = window.location.hash.replace('#', '');
  if (['dashboard', 'inventory', 'loans', 'audit'].includes(hash) && hash !== currentView) {
    switchView(hash);
  }
});

// -------------------------------------------------------------
// View 0: Dashboard Analitik Terpusat
// -------------------------------------------------------------
async function loadCentralDashboard() {
  const periodTag = document.getElementById('dash-period-tag');
  if (periodTag) periodTag.textContent = formatPeriodName(currentLoanPeriod);

  try {
    const data = await api(`/api/dashboard/overview?period=${currentLoanPeriod}`);
    renderCentralDashboard(data);
  } catch (_) {}
}

function formatHumanActivity(action, details, entity) {
  let actionLabel = action;
  let actionClass = 'badge';
  let detailDesc = '';

  let d = details;
  if (typeof d === 'string') {
    try { d = JSON.parse(d); } catch (_) { d = {}; }
  } else if (!d) {
    d = {};
  }

  switch (action) {
    case 'LOAN_RETURN':
      actionLabel = 'Kembali';
      actionClass = 'badge badge-kembali';
      const setsRet = d.sets || d.quantity || '';
      const periodStr = d.period ? `Periode ${formatPeriodName(d.period)}` : '';
      detailDesc = `Pengembalian ${setsRet ? setsRet + ' set kapolding' : 'unit steger'}${periodStr ? ' (' + periodStr + ')' : ''} • Masuk gudang`;
      break;

    case 'LOAN_CREATE':
      actionLabel = 'Sewa Steger Baru';
      actionClass = 'badge badge-aktif';
      const setsNew = d.quantity || d.sets || 1;
      const fee = d.rental_fee ? ` • Sewa: ${formatRupiah(d.rental_fee)}/bln` : '';
      const loc = d.project_location ? ` • Lokasi: ${d.project_location}` : '';
      const ktp = d.id_card_given ? ' • KTP ditahan' : '';
      detailDesc = `Pencatatan sewa ${setsNew} set kapolding (${d.elbow_count || setsNew * 2} siku, ${d.shock_count || 0} shock)${fee}${loc}${ktp}`;
      break;

    case 'MUTASI_IN':
      actionLabel = 'Barang Masuk';
      actionClass = 'badge badge-lunas';
      detailDesc = `Penambahan stok +${d.quantity || 0} unit ${d.reference ? '• Keterangan: ' + d.reference : ''}`;
      break;

    case 'MUTASI_OUT':
      actionLabel = 'Barang Keluar';
      actionClass = 'badge badge-belum';
      detailDesc = `Pengeluaran stok -${d.quantity || 0} unit ${d.reference ? '• Keterangan: ' + d.reference : ''}`;
      break;

    case 'MUTASI_ADJUSTMENT':
      actionLabel = 'Opname / Koreksi';
      actionClass = 'badge badge-rollover';
      detailDesc = `Stok fisik disesuaikan menjadi ${d.quantity || 0} unit ${d.reference ? '• Ref: ' + d.reference : ''}`;
      break;

    case 'CREATE':
      actionLabel = 'Tambah Barang Toko';
      actionClass = 'badge badge-lunas';
      const initQty = d.quantity !== undefined ? ` • Stok awal: ${d.quantity} ${d.unit || 'pcs'}` : '';
      const priceStr = d.price ? ` • Harga: ${formatRupiah(d.price)}` : '';
      detailDesc = `Pendaftaran item baru SKU ${d.sku || entity || ''}${initQty}${priceStr}`;
      break;

    case 'UPDATE':
      actionLabel = 'Edit';
      actionClass = 'badge';
      if (d.old && d.new) {
        const changes = [];
        if (d.old.name !== d.new.name) changes.push(`Nama: "${d.old.name}" &rarr; "${d.new.name}"`);
        if (d.old.price !== d.new.price) changes.push(`Harga: ${formatRupiah(d.old.price)} &rarr; ${formatRupiah(d.new.price)}`);
        if (d.old.category !== d.new.category) changes.push(`Kategori: ${d.old.category} &rarr; ${d.new.category}`);
        if (d.old.location !== d.new.location) changes.push(`Lokasi: ${d.old.location || '-'} &rarr; ${d.new.location || '-'}`);
        if (d.old.min_threshold !== d.new.min_threshold) changes.push(`Batas Min: ${d.old.min_threshold} &rarr; ${d.new.min_threshold}`);
        detailDesc = changes.length > 0 ? changes.join(' • ') : 'Diperbarui';
      } else {
        detailDesc = 'Diperbarui';
      }
      break;

    case 'DELETE':
      actionLabel = 'Hapus Data';
      actionClass = 'badge badge-belum';
      detailDesc = d.code ? `Data sewa ${d.code} dihapus dari sistem` : `Barang toko dihapus dari katalog`;
      break;

    default:
      if (action.startsWith('MUTASI_')) {
        actionLabel = 'Mutasi Toko';
        detailDesc = `Jumlah: ${d.quantity || '-'} • Keterangan: ${d.reference || '-'}`;
      } else {
        actionLabel = action;
        detailDesc = typeof d === 'object' ? JSON.stringify(d) : String(d);
      }
  }

  return { actionLabel, actionClass, detailDesc };
}

function renderCentralDashboard(d) {
  if (!d) return;

  // 1. Top KPI Row
  document.getElementById('kpi-store-value').textContent = formatRupiah(d.store.total_value);
  document.getElementById('kpi-store-sub').textContent = `${d.store.total_items} SKU • ${d.store.total_units} unit fisik`;

  document.getElementById('kpi-rental-util').textContent = `${d.rental.utilization_rate.toFixed(1)}%`;
  document.getElementById('kpi-rental-sub').textContent = `${d.rental.rented_sets} dari ${d.rental.total_fleet_sets} set tersewa`;

  document.getElementById('kpi-net-profit').textContent = formatRupiah(d.rental.net_profit_projected);
  document.getElementById('kpi-profit-sub').textContent = `Terkumpul: ${formatRupiah(d.rental.net_profit_collected)}`;

  const alertsTotal = (d.rental.id_cards_held || 0) + (d.rental.unpaid_borrowers_count || 0) + (d.store.low_stock_count || 0);
  document.getElementById('kpi-alerts-count').textContent = `${alertsTotal} Poin`;
  document.getElementById('kpi-alerts-sub').textContent = `KTP: ${d.rental.id_cards_held} | Tertunggak: ${d.rental.unpaid_borrowers_count} | Tipis: ${d.store.low_stock_count}`;

  // 2. Kolom Rental
  document.getElementById('dash-rental-contracts-badge').textContent = `${d.rental.active_contracts} Kontrak Aktif`;
  document.getElementById('dash-meter-label').textContent = `${d.rental.utilization_rate.toFixed(1)}% Terpakai`;
  document.getElementById('dash-meter-fill').style.width = `${Math.min(d.rental.utilization_rate, 100)}%`;

  document.getElementById('dash-rented-sets').textContent = `${d.rental.rented_sets} Set`;
  document.getElementById('dash-avail-sets').textContent = `${d.rental.available_sets} Set`;
  document.getElementById('dash-total-sets').textContent = `${d.rental.total_fleet_sets} Set`;

  document.getElementById('dash-fin-revenue').textContent = formatRupiah(d.rental.expected_revenue);
  document.getElementById('dash-fin-rev-collected').textContent = formatRupiah(d.rental.collected_revenue);
  document.getElementById('dash-fin-rev-unpaid').textContent = formatRupiah(d.rental.uncollected_revenue);

  document.getElementById('dash-fin-cost').textContent = formatRupiah(d.rental.expected_cost);
  document.getElementById('dash-fin-cost-settled').textContent = formatRupiah(d.rental.settled_cost);
  document.getElementById('dash-fin-cost-unsettled').textContent = formatRupiah(d.rental.unsettled_cost);

  document.getElementById('dash-fin-profit').textContent = `${formatRupiah(d.rental.net_profit_projected)} (Terkumpul: ${formatRupiah(d.rental.net_profit_collected)})`;

  // 3. Kolom Toko
  document.getElementById('dash-store-sku-badge').textContent = `${d.store.total_items} SKU`;
  const catList = document.getElementById('dash-category-list');
  if (catList) {
    if (!d.store.category_breakdown || d.store.category_breakdown.length === 0) {
      catList.innerHTML = `<div class="dash-cat-row" style="color:var(--text-muted); font-size:12px;">Belum ada data barang toko.</div>`;
    } else {
      const maxVal = Math.max(...d.store.category_breakdown.map(c => c.total_value), 1);
      catList.innerHTML = d.store.category_breakdown.map(c => {
        const pct = Math.min((c.total_value / maxVal) * 100, 100);
        return `
          <div class="dash-cat-row">
            <div class="dash-cat-header">
              <span class="dash-cat-name">${escape(c.category)} (${c.items_count} item)</span>
              <span class="dash-cat-val">${formatRupiah(c.total_value)}</span>
            </div>
            <div class="dash-cat-progress">
              <div class="dash-cat-fill" style="width: ${pct}%;"></div>
            </div>
          </div>
        `;
      }).join('');
    }
  }

  // 4. Tabel 5 Aktivitas Terkini
  const recentTbody = document.getElementById('dash-recent-tbody');
  if (recentTbody) {
    if (!d.recent_activities || d.recent_activities.length === 0) {
      recentTbody.innerHTML = `<tr><td colspan="4" class="cell-empty">Belum ada riwayat aktivitas terbaru.</td></tr>`;
    } else {
      recentTbody.innerHTML = d.recent_activities.map(a => {
        const timeStr = a.created_at ? a.created_at.replace('T', ' ').slice(0, 16) : '-';
        const { actionLabel, actionClass, detailDesc } = formatHumanActivity(a.action, a.details, a.name || a.sku);
        return `
          <tr>
            <td style="font-family: var(--font-mono); font-size: 11px; color: var(--text-muted);">${timeStr}</td>
            <td><span class="${actionClass}">${escape(actionLabel)}</span></td>
            <td><strong>${escape(a.name || a.sku || '-')}</strong></td>
            <td style="font-size: 12px; color: var(--text);">${detailDesc}</td>
          </tr>
        `;
      }).join('');
    }
  }
}

// -------------------------------------------------------------
// View 1: Rental Steger / Kapolding
// -------------------------------------------------------------
const btnPrevMonth = document.getElementById('btn-prev-month');
const btnNextMonth = document.getElementById('btn-next-month');
const btnCurrentMonth = document.getElementById('btn-current-month');
const loanPeriodLabel = document.getElementById('loan-period-label');
const loanPeriodInput = document.getElementById('loan-period-input');
const monthFinancialSummary = document.getElementById('month-financial-summary');

const bannerKpdSets = document.getElementById('banner-kpd-sets');
const bannerKpdElbows = document.getElementById('banner-kpd-elbows');
const bannerKpdShocks = document.getElementById('banner-kpd-shocks');

const filterLoanSearch = document.getElementById('filter-loan-search');
const filterLoanStatus = document.getElementById('filter-loan-status');
const filterLoanPayment = document.getElementById('filter-loan-payment');
const loansTbody = document.getElementById('loans-tbody');

const panelAddLoan = document.getElementById('panel-add-loan');
const loanFormTitle = document.getElementById('loan-form-title');
const loanForm = document.getElementById('loan-form');
const btnOpenAddLoan = document.getElementById('btn-open-add-loan');
const btnCloseLoanForm = document.getElementById('btn-close-loan-form');
const btnCancelLoan = document.getElementById('btn-cancel-loan');

const inputLoanQty = document.getElementById('loan-qty');
const inputLoanElbows = document.getElementById('loan-elbows');
const inputLoanShocks = document.getElementById('loan-shocks');
const inputLoanRateSet = document.getElementById('loan-rate-set');
const inputLoanCostSet = document.getElementById('loan-cost-set');
const inputLoanRentalFee = document.getElementById('loan-rental-fee');
const inputLoanOwnerCost = document.getElementById('loan-owner-cost');

function recalculateLoanPricing() {
  const sets = parseInt(inputLoanQty.value, 10) || 0;
  const rateSet = parseFloat(inputLoanRateSet.value) || 0;
  const costSet = parseFloat(inputLoanCostSet.value) || 0;
  inputLoanRentalFee.value = sets * rateSet;
  inputLoanOwnerCost.value = sets * costSet;
}

inputLoanQty.addEventListener('input', () => {
  const sets = parseInt(inputLoanQty.value, 10) || 0;
  inputLoanElbows.value = sets * 2;
  recalculateLoanPricing();
});
inputLoanRateSet.addEventListener('input', recalculateLoanPricing);
inputLoanCostSet.addEventListener('input', recalculateLoanPricing);

function getMaxAllowedMonth() {
  const now = new Date();
  const nextMonth = new Date(now.getFullYear(), now.getMonth() + 1, 1);
  const y = nextMonth.getFullYear();
  const m = String(nextMonth.getMonth() + 1).padStart(2, '0');
  return `${y}-${m}`;
}

function updatePeriodDisplay() {
  const maxMonth = getMaxAllowedMonth();
  loanPeriodInput.max = maxMonth;
  loanPeriodInput.value = currentLoanPeriod;
  loanPeriodLabel.textContent = formatPeriodName(currentLoanPeriod);

  if (currentLoanPeriod >= maxMonth) {
    btnNextMonth.disabled = true;
    btnNextMonth.style.opacity = '0.5';
    btnNextMonth.style.cursor = 'not-allowed';
    btnNextMonth.title = 'Maksimal pratinjau 1 bulan ke depan';
  } else {
    btnNextMonth.disabled = false;
    btnNextMonth.style.opacity = '1';
    btnNextMonth.style.cursor = 'pointer';
    btnNextMonth.title = '';
  }
}

function shiftMonth(offset) {
  const [yStr, mStr] = currentLoanPeriod.split('-');
  let y = parseInt(yStr, 10);
  let m = parseInt(mStr, 10) + offset;
  if (m < 1) {
    m = 12;
    y -= 1;
  } else if (m > 12) {
    m = 1;
    y += 1;
  }
  const targetPeriod = `${y}-${String(m).padStart(2, '0')}`;
  const maxMonth = getMaxAllowedMonth();

  if (offset > 0 && targetPeriod > maxMonth) {
    showStatus('Pratinjau dibatasi maksimal 1 bulan ke depan');
    return;
  }

  currentLoanPeriod = targetPeriod;
  updatePeriodDisplay();
  loadLoans();
}

btnPrevMonth.addEventListener('click', () => shiftMonth(-1));
btnNextMonth.addEventListener('click', () => shiftMonth(1));
btnCurrentMonth.addEventListener('click', () => {
  currentLoanPeriod = new Date().toISOString().slice(0, 7);
  updatePeriodDisplay();
  loadLoans();
});

loanPeriodInput.addEventListener('change', () => {
  if (loanPeriodInput.value) {
    const maxMonth = getMaxAllowedMonth();
    if (loanPeriodInput.value > maxMonth) {
      showStatus('Maksimal dapat melihat 1 bulan ke depan');
      currentLoanPeriod = maxMonth;
    } else {
      currentLoanPeriod = loanPeriodInput.value;
    }
    updatePeriodDisplay();
    loadLoans();
  }
});

function applyStockBoxStatus(boxId, qty, minThreshold = 5) {
  const box = document.getElementById(boxId);
  if (!box) return;
  box.classList.remove('status-empty', 'status-warning', 'status-ok');
  if (qty <= 0) {
    box.classList.add('status-empty');
  } else if (qty <= minThreshold) {
    box.classList.add('status-warning');
  } else {
    box.classList.add('status-ok');
  }
}

async function updateScaffoldStockBanner() {
  try {
    const fleet = await api('/api/fleet');
    const setItem = fleet.find(f => f.code === 'KPD_SET');
    const elbowItem = fleet.find(f => f.code === 'KPD_ELBOW');
    const shockItem = fleet.find(f => f.code === 'KPD_SHOCK');

    if (setItem) {
      bannerKpdSets.textContent = `${setItem.available_in_warehouse} Set`;
      const sub = document.getElementById('banner-kpd-sub');
      if (sub) sub.innerHTML = `(${setItem.currently_rented} sewa / total ${setItem.total_owned})`;
    }

    if (elbowItem) {
      bannerKpdElbows.textContent = `${elbowItem.available_in_warehouse} Pcs`;
      const sub = document.getElementById('banner-siku-sub');
      if (sub) sub.innerHTML = `(${elbowItem.currently_rented} sewa / total ${elbowItem.total_owned})`;
    }

    if (shockItem) {
      bannerKpdShocks.textContent = `${shockItem.available_in_warehouse} Pcs`;
      const sub = document.getElementById('banner-shock-sub');
      if (sub) {
        if (shockItem.total_owned === 0) {
          sub.innerHTML = `(0 aset)`;
        } else {
          sub.innerHTML = `(${shockItem.currently_rented} sewa / total ${shockItem.total_owned})`;
        }
      }
    }
  } catch (_) {}
}

// Handler tombol Ubah Total Armada
document.addEventListener('click', async e => {
  const btn = e.target.closest('.btn-quick-fleet');
  if (!btn) return;
  const code = btn.dataset.code;
  try {
    const fleet = await api('/api/fleet');
    const target = fleet.find(f => f.code === code);
    if (!target) return;

    const inputVal = prompt(`Masukkan jumlah total kepemilikan armada untuk:\n"${target.name}"\n(Total aset fisik Anda, baik yang ada di gudang maupun sedang di proyek)`, target.total_owned);
    if (inputVal === null) return;
    const newTotal = parseInt(inputVal.trim(), 10);
    if (isNaN(newTotal) || newTotal < 0) {
      showStatus('Jumlah armada harus berupa angka positif', true);
      return;
    }

    await api(`/api/fleet/${code}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ total_owned: newTotal })
    });

    showStatus(`Total armada ${target.name} berhasil diubah menjadi ${newTotal} ${target.unit}`);
    await updateScaffoldStockBanner();
    if (modalFleetManager && !modalFleetManager.classList.contains('hidden')) {
      await renderFleetModalTable();
    }
  } catch (_) {}
});

async function loadLoans() {
  updatePeriodDisplay();
  const p = new URLSearchParams();
  p.append('period', currentLoanPeriod);

  if (filterLoanSearch && filterLoanSearch.value && filterLoanSearch.value.trim()) p.append('search', filterLoanSearch.value.trim());
  if (filterLoanStatus && filterLoanStatus.value) p.append('status', filterLoanStatus.value);
  
  const payVal = filterLoanPayment ? filterLoanPayment.value : '';
  if (payVal === 'unpaid_borrower') p.append('unpaid_borrower', 'true');
  else if (payVal === 'unpaid_owner') p.append('unpaid_owner', 'true');
  else if (payVal === 'id_held') p.append('id_held', 'true');

  try {
    const [loansData, statsData] = await Promise.all([
      api('/api/loans?' + p.toString()),
      api(`/api/loans/stats?period=${currentLoanPeriod}`)
    ]);

    loans = loansData;
    renderLoans();
    renderMonthlyFinancialSummary(statsData);
    await updateScaffoldStockBanner();
  } catch (_) {}
}

function renderMonthlyFinancialSummary(s) {
  if (!monthFinancialSummary || !s) return;
  const profitExpected = (s.expected_revenue || 0) - (s.expected_cost || 0);
  const profitCollected = (s.collected_revenue || 0) - (s.settled_cost || 0);

  monthFinancialSummary.innerHTML = `
    <span class="badge" style="background:#ffffff; border:1px solid var(--border); color:var(--text); padding:3px 8px;">
      Sewa: <strong>${formatRupiah(s.collected_revenue)}</strong><span style="color:var(--text-dim); font-size:10.5px;">/${formatRupiah(s.expected_revenue)}</span>
    </span>
    <span class="badge" style="background:#ffffff; border:1px solid var(--border); color:var(--text); padding:3px 8px;">
      Setor: <strong>${formatRupiah(s.settled_cost)}</strong><span style="color:var(--text-dim); font-size:10.5px;">/${formatRupiah(s.expected_cost)}</span>
    </span>
    <span class="badge" style="background:var(--success-bg); border:1px solid var(--success-border); color:var(--success); font-weight:700; padding:3px 8px;">
      Laba: ${formatRupiah(profitCollected)}
    </span>
  `;
}

function renderLoans() {
  const loansCards = document.getElementById('loans-cards');
  if (loans.length === 0) {
    loansTbody.innerHTML = `<tr><td colspan="9" class="cell-empty">Tidak ada sewa untuk ${formatPeriodName(currentLoanPeriod)}.</td></tr>`;
    if (loansCards) loansCards.innerHTML = `<div class="cell-empty">Tidak ada sewa untuk ${formatPeriodName(currentLoanPeriod)}.</div>`;
    return;
  }

  loansTbody.innerHTML = loans.map(l => {
    const isReturned = l.status === 'RETURNED';
    const loanDateStr = l.loan_date ? l.loan_date.slice(0, 10) : '-';
    const returnDateStr = l.return_date ? l.return_date.slice(0, 10) : '';

    const returnInfo = isReturned && returnDateStr ? `<div style="font-size:10px; color: var(--success);">Kembali: ${returnDateStr}</div>` : '';
    const rolloverTag = l.is_rollover ? `<div style="margin-top:2px;"><span class="badge-rollover">Rollover</span></div>` : '';

    const badgeStatus = isReturned 
      ? `<button type="button" class="btn-toggle-badge badge-kembali" data-action="toggle-return" data-id="${l.id}" title="Klik untuk membuka kembali sewa">KEMBALI</button>`
      : `<button type="button" class="btn-toggle-badge badge-aktif" data-action="toggle-return" data-id="${l.id}" title="Klik jika barang sudah dikembalikan">AKTIF</button>`;

    const badgeIdCard = l.id_card_given
      ? `<button type="button" class="btn-toggle-badge badge-ktp-ya" data-action="toggle-idcard" data-id="${l.id}" title="Klik untuk ubah status KTP">DITAHAN</button>`
      : `<button type="button" class="btn-toggle-badge badge-ktp-tidak" data-action="toggle-idcard" data-id="${l.id}" title="Klik jika KTP ditahan">TIDAK</button>`;

    const badgePaid = l.is_paid
      ? `<button type="button" class="btn-toggle-badge badge-lunas" data-action="toggle-paid" data-id="${l.id}">LUNAS</button>`
      : `<button type="button" class="btn-toggle-badge badge-belum" data-action="toggle-paid" data-id="${l.id}">BELUM</button>`;

    const badgeOwner = l.is_paid_to_owner
      ? `<button type="button" class="btn-toggle-badge badge-lunas" data-action="toggle-owner" data-id="${l.id}">DISETOR</button>`
      : `<button type="button" class="btn-toggle-badge badge-belum" data-action="toggle-owner" data-id="${l.id}">BELUM</button>`;

    const shockDetail = l.shock_count > 0 ? ` + ${l.shock_count} shock` : '';
    const projectInfo = l.project_location ? `<div style="font-size:11px; color: var(--accent-blue);">Loc: ${escape(l.project_location)}</div>` : '';
    const phoneInfo = l.borrower_phone ? `<div style="font-size:10px; color:var(--text-muted);">${escape(l.borrower_phone)}</div>` : '';

    return `
      <tr class="${isReturned ? 'row-returned' : ''}">
        <td>
          <strong style="font-family: var(--font-mono); color: var(--accent-blue);">${escape(l.loan_code)}</strong>
        </td>
        <td>
          <div style="font-weight: 600; color: var(--text);">${escape(l.borrower_name)}</div>
          ${phoneInfo}
          ${projectInfo}
        </td>
        <td>
          <div style="font-weight: 700; color: var(--text);">${l.quantity} Set</div>
          <div style="font-size: 11px; color: var(--text-muted);">${l.elbow_count} siku${shockDetail}</div>
        </td>
        <td>
          <div>Mulai: ${loanDateStr}</div>
          ${returnInfo}
          ${rolloverTag}
        </td>
        <td style="text-align: center;">
          ${badgeIdCard}
        </td>
        <td style="text-align: right;">
          <div style="font-weight: 600; font-family: var(--font-mono); color: var(--accent-blue);">${formatRupiah(l.rental_fee)}</div>
          <div style="margin-top: 2px;">${badgePaid}</div>
        </td>
        <td style="text-align: right;">
          <div style="font-weight: 600; font-family: var(--font-mono); color: var(--warning);">${formatRupiah(l.owner_cost)}</div>
          <div style="margin-top: 2px;">${badgeOwner}</div>
        </td>
        <td style="text-align: center;">
          ${badgeStatus}
        </td>
        <td style="text-align: right;">
          <div class="cell-actions">
            <button type="button" class="btn btn-sm" data-action="edit-loan" data-id="${l.id}">Edit</button>
            <button type="button" class="btn btn-sm" data-action="del-loan" data-id="${l.id}">Hapus</button>
          </div>
        </td>
      </tr>
    `;
  }).join('');

  if (loansCards) {
    loansCards.innerHTML = loans.map(l => {
      const isReturned = l.status === 'RETURNED';
      const loanDateStr = l.loan_date ? l.loan_date.slice(0, 10) : '-';
      const returnDateStr = l.return_date ? l.return_date.slice(0, 10) : '';
      const returnInfo = isReturned && returnDateStr ? `<span style="font-size:11px; color: var(--success);">(Kembali: ${returnDateStr})</span>` : '';
      const rolloverTag = l.is_rollover ? `<span class="badge-rollover">Rollover</span>` : '';

      const badgeStatus = isReturned 
        ? `<button type="button" class="btn-toggle-badge badge-kembali" data-action="toggle-return" data-id="${l.id}">KEMBALI</button>`
        : `<button type="button" class="btn-toggle-badge badge-aktif" data-action="toggle-return" data-id="${l.id}">AKTIF</button>`;

      const badgeIdCard = l.id_card_given
        ? `<button type="button" class="btn-toggle-badge badge-ktp-ya" data-action="toggle-idcard" data-id="${l.id}">KTP DITAHAN</button>`
        : `<button type="button" class="btn-toggle-badge badge-ktp-tidak" data-action="toggle-idcard" data-id="${l.id}">KTP TIDAK</button>`;

      const badgePaid = l.is_paid
        ? `<button type="button" class="btn-toggle-badge badge-lunas" data-action="toggle-paid" data-id="${l.id}">LUNAS</button>`
        : `<button type="button" class="btn-toggle-badge badge-belum" data-action="toggle-paid" data-id="${l.id}">BELUM LUNAS</button>`;

      const badgeOwner = l.is_paid_to_owner
        ? `<button type="button" class="btn-toggle-badge badge-lunas" data-action="toggle-owner" data-id="${l.id}">DISETOR</button>`
        : `<button type="button" class="btn-toggle-badge badge-belum" data-action="toggle-owner" data-id="${l.id}">BELUM SETOR</button>`;

      const shockDetail = l.shock_count > 0 ? ` + ${l.shock_count} shock` : '';
      const projectInfo = l.project_location ? `<div style="font-size:11px; color: var(--accent-blue); margin-top:2px;">Proyek: ${escape(l.project_location)}</div>` : '';
      const phoneInfo = l.borrower_phone ? `<div style="font-size:11px; color:var(--text-muted);">${escape(l.borrower_phone)}</div>` : '';
      const netProfit = (l.rental_fee || 0) - (l.owner_cost || 0);

      return `
        <div class="mobile-data-card ${isReturned ? 'row-returned' : ''}">
          <div class="mobile-card-header">
            <div>
              <div style="font-size:11px; font-family:var(--font-mono); color: var(--accent-blue);">${escape(l.loan_code)} ${rolloverTag}</div>
              <div class="mobile-card-title">${escape(l.borrower_name)}</div>
              ${phoneInfo}
              ${projectInfo}
            </div>
            <div>
              ${badgeStatus}
            </div>
          </div>

          <div class="mobile-card-grid">
            <div class="mobile-card-field">
              <span class="mobile-field-label">Kuantitas</span>
              <span class="mobile-field-val" style="color: var(--text);">${l.quantity} Set <span style="font-size:10px; color:var(--text-dim);">(${l.elbow_count} siku${shockDetail})</span></span>
            </div>
            <div class="mobile-card-field">
              <span class="mobile-field-label">Tgl Mulai</span>
              <span class="mobile-field-val">${loanDateStr} ${returnInfo}</span>
            </div>
            <div class="mobile-card-field">
              <span class="mobile-field-label">Tagihan Sewa</span>
              <span class="mobile-field-val" style="color: var(--accent-blue);">${formatRupiah(l.rental_fee)}</span>
            </div>
            <div class="mobile-card-field">
              <span class="mobile-field-label">Setor Pemilik</span>
              <span class="mobile-field-val" style="color: var(--warning);">${formatRupiah(l.owner_cost)}</span>
            </div>
          </div>

          <div style="display:flex; justify-content:space-between; align-items:center; gap:6px; flex-wrap:wrap; padding-top:6px; border-top:1px solid var(--border);">
            <div style="display:flex; gap:6px; align-items:center; flex-wrap:wrap;">
              ${badgePaid}
              ${badgeOwner}
              ${badgeIdCard}
            </div>
            <div class="mobile-card-actions" style="border:none; padding:0;">
              <button type="button" class="btn btn-sm" data-action="edit-loan" data-id="${l.id}">Edit</button>
              <button type="button" class="btn btn-sm btn-danger" data-action="del-loan" data-id="${l.id}">&times;</button>
            </div>
          </div>
        </div>
      `;
    }).join('');
  }
}

function openAddLoan() {
  loanFormTitle.textContent = 'Tambah Sewa';
  loanForm.reset();
  document.getElementById('loan-id').value = '';
  document.getElementById('loan-date').value = new Date().toISOString().slice(0, 10);
  inputLoanQty.value = '1';
  inputLoanElbows.value = '2';
  inputLoanShocks.value = '0';
  inputLoanRateSet.value = '50000';
  inputLoanCostSet.value = '35000';
  inputLoanRentalFee.value = '50000';
  inputLoanOwnerCost.value = '35000';
  document.getElementById('loan-id-card').checked = false;
  document.getElementById('loan-is-paid').checked = false;
  document.getElementById('loan-is-paid-owner').checked = false;
  panelAddLoan.classList.remove('hidden');
}

function openEditLoan(l) {
  loanFormTitle.textContent = 'Edit Data Sewa: ' + l.loan_code;
  document.getElementById('loan-id').value = l.id;
  inputLoanQty.value = l.quantity;
  inputLoanElbows.value = l.elbow_count;
  inputLoanShocks.value = l.shock_count;
  document.getElementById('loan-borrower').value = l.borrower_name;
  document.getElementById('loan-phone').value = l.borrower_phone || '';
  document.getElementById('loan-project').value = l.project_location || '';
  document.getElementById('loan-date').value = l.loan_date ? l.loan_date.slice(0, 10) : '';

  const sets = l.quantity || 1;
  inputLoanRateSet.value = Math.round(l.rental_fee / sets);
  inputLoanCostSet.value = Math.round(l.owner_cost / sets);
  inputLoanRentalFee.value = l.rental_fee;
  inputLoanOwnerCost.value = l.owner_cost;

  document.getElementById('loan-notes').value = l.notes || '';
  document.getElementById('loan-id-card').checked = !!l.id_card_given;
  document.getElementById('loan-is-paid').checked = !!l.is_paid;
  document.getElementById('loan-is-paid-owner').checked = !!l.is_paid_to_owner;
  panelAddLoan.classList.remove('hidden');
}

btnOpenAddLoan.addEventListener('click', openAddLoan);
btnCloseLoanForm.addEventListener('click', () => panelAddLoan.classList.add('hidden'));
btnCancelLoan.addEventListener('click', () => panelAddLoan.classList.add('hidden'));

loanForm.addEventListener('submit', async e => {
  e.preventDefault();
  const id = document.getElementById('loan-id').value;
  const sets = parseInt(inputLoanQty.value, 10) || 1;
  const elbows = parseInt(inputLoanElbows.value, 10) || (sets * 2);
  const shocks = parseInt(inputLoanShocks.value, 10) || 0;

  const payload = {
    quantity: sets,
    elbow_count: elbows,
    shock_count: shocks,
    borrower_name: document.getElementById('loan-borrower').value.trim(),
    borrower_phone: document.getElementById('loan-phone').value.trim(),
    project_location: document.getElementById('loan-project').value.trim(),
    loan_date: document.getElementById('loan-date').value,
    period: currentLoanPeriod,
    rental_fee: parseFloat(inputLoanRentalFee.value) || 0,
    owner_cost: parseFloat(inputLoanOwnerCost.value) || 0,
    id_card_given: document.getElementById('loan-id-card').checked,
    is_paid: document.getElementById('loan-is-paid').checked,
    is_paid_to_owner: document.getElementById('loan-is-paid-owner').checked,
    notes: document.getElementById('loan-notes').value.trim()
  };

  try {
    if (id) {
      await api(`/api/loans/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      showStatus('Data sewa berhasil diperbarui');
    } else {
      await api('/api/loans', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      showStatus('Sewa kapolding baru berhasil dicatat');
    }
    panelAddLoan.classList.add('hidden');
    loadLoans();
  } catch (_) {}
});

async function handleLoanClickAction(e) {
  const btn = e.target.closest('[data-action]');
  if (!btn) return;
  const action = btn.dataset.action;
  const id = btn.dataset.id;
  const l = loans.find(x => String(x.id) === String(id));

  if (action === 'edit-loan' && l) {
    openEditLoan(l);
  } else if (action === 'del-loan' && l) {
    if (!confirm(`Hapus data sewa ${l.loan_code} untuk ${l.borrower_name}?`)) return;
    try {
      await api(`/api/loans/${id}`, { method: 'DELETE' });
      showStatus('Data sewa berhasil dihapus');
      loadLoans();
    } catch (_) {}
  } else if (action.startsWith('toggle-')) {
    let field = '';
    if (action === 'toggle-paid') field = 'is_paid';
    else if (action === 'toggle-owner') field = 'is_paid_to_owner';
    else if (action === 'toggle-idcard') field = 'id_card_given';
    else if (action === 'toggle-return') field = 'return';

    try {
      await api(`/api/loans/${id}/toggle`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ field, period: currentLoanPeriod })
      });
      loadLoans();
    } catch (_) {}
  }
}

if (loansTbody) loansTbody.addEventListener('click', handleLoanClickAction);
const loansCardsContainer = document.getElementById('loans-cards');
if (loansCardsContainer) loansCardsContainer.addEventListener('click', handleLoanClickAction);

filterLoanSearch.addEventListener('input', () => {
  clearTimeout(filterLoanSearch._t);
  filterLoanSearch._t = setTimeout(loadLoans, 200);
});
filterLoanStatus.addEventListener('change', loadLoans);
filterLoanPayment.addEventListener('change', loadLoans);

// -------------------------------------------------------------
// View 2: Stok Toko Bahan Bangunan (Barang Konsumsi)
// -------------------------------------------------------------
const filterSearch = document.getElementById('filter-search');
const filterCategory = document.getElementById('filter-category');
const inventoryTbody = document.getElementById('inventory-tbody');

const formPanel = document.getElementById('form-panel');
const formTitle = document.getElementById('form-title');
const itemForm = document.getElementById('item-form');
const btnOpenAdd = document.getElementById('btn-open-add');
const btnCloseForm = document.getElementById('btn-close-form');
const btnCancelForm = document.getElementById('btn-cancel-form');

const selectItemCategory = document.getElementById('item-category');
const modalCatManager = document.getElementById('modal-category-manager');
const btnToolbarCatManager = document.getElementById('btn-toolbar-cat-manager');
const btnFormCatManager = document.getElementById('btn-form-cat-manager');
const btnCloseCatModal = document.getElementById('btn-close-cat-modal');
const btnDoneCatModal = document.getElementById('btn-done-cat-modal');

const selectItemLocation = document.getElementById('item-location');
const filterLocation = document.getElementById('filter-location');
const modalLocManager = document.getElementById('modal-location-manager');
const btnToolbarLocManager = document.getElementById('btn-toolbar-loc-manager');
const btnFormLocManager = document.getElementById('btn-form-loc-manager');
const btnCloseLocModal = document.getElementById('btn-close-loc-modal');
const btnDoneLocModal = document.getElementById('btn-done-loc-modal');
const formAddLocation = document.getElementById('form-add-location');
const inputNewLocName = document.getElementById('input-new-loc-name');
const inputNewLocDesc = document.getElementById('input-new-loc-desc');
const locationTbody = document.getElementById('location-tbody');
const formAddCategory = document.getElementById('form-add-category');
const inputNewCatName = document.getElementById('input-new-cat-name');
const categoryTbody = document.getElementById('category-tbody');

const movementDialog = document.getElementById('movement-dialog');
const movementForm = document.getElementById('movement-form');
const movementTitle = document.getElementById('movement-title');
const movementItemName = document.getElementById('movement-item-name');
const movementItemId = document.getElementById('movement-item-id');
const btnCloseDialog = document.getElementById('btn-close-dialog');
const btnCancelDialog = document.getElementById('btn-cancel-dialog');

async function loadCategories() {
  try {
    categories = await api('/api/categories');
    renderCategoryDropdowns();
    renderCategoryTable();
  } catch (_) {}
}

function renderCategoryDropdowns() {
  const currentFilterVal = filterCategory.value;
  const currentItemVal = selectItemCategory.value;

  const filterOpts = ['<option value="">Semua Kategori</option>'];
  categories.forEach(c => {
    filterOpts.push(`<option value="${escape(c.name)}">${escape(c.name)}</option>`);
  });
  filterCategory.innerHTML = filterOpts.join('');
  filterCategory.value = currentFilterVal;

  const formOpts = ['<option value="">-- Pilih Kategori --</option>'];
  categories.forEach(c => {
    formOpts.push(`<option value="${escape(c.name)}">${escape(c.name)}</option>`);
  });
  selectItemCategory.innerHTML = formOpts.join('');
  selectItemCategory.value = currentItemVal;
}

function renderCategoryTable() {
  if (!categoryTbody) return;
  if (categories.length === 0) {
    categoryTbody.innerHTML = `<tr><td colspan="3" class="cell-empty">Belum ada kategori toko.</td></tr>`;
    return;
  }

  categoryTbody.innerHTML = categories.map(c => `
    <tr>
      <td><strong>${escape(c.name)}</strong></td>
      <td style="text-align: center; font-family: var(--font-mono);">${c.items_count}</td>
      <td style="text-align: right;">
        <div class="cell-actions">
          <button type="button" class="btn btn-sm" data-action="edit-cat" data-id="${c.id}" data-name="${escape(c.name)}">Edit</button>
          <button type="button" class="btn btn-sm" data-action="del-cat" data-id="${c.id}" data-name="${escape(c.name)}" data-count="${c.items_count}">Hapus</button>
        </div>
      </td>
    </tr>
  `).join('');
}

btnToolbarCatManager.addEventListener('click', () => {
  loadCategories();
  modalCatManager.classList.remove('hidden');
});
btnFormCatManager.addEventListener('click', () => {
  loadCategories();
  modalCatManager.classList.remove('hidden');
});
btnCloseCatModal.addEventListener('click', () => modalCatManager.classList.add('hidden'));
btnDoneCatModal.addEventListener('click', () => modalCatManager.classList.add('hidden'));

formAddCategory.addEventListener('submit', async e => {
  e.preventDefault();
  const name = inputNewCatName.value.trim();
  if (!name) return;
  try {
    await api('/api/categories', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name })
    });
    inputNewCatName.value = '';
    showStatus('Kategori toko baru berhasil ditambahkan');
    await loadCategories();
    selectItemCategory.value = name;
  } catch (_) {}
});

categoryTbody.addEventListener('click', async e => {
  const btn = e.target.closest('[data-action]');
  if (!btn) return;
  const action = btn.dataset.action;
  const id = btn.dataset.id;
  const name = btn.dataset.name;
  const count = parseInt(btn.dataset.count, 10) || 0;

  if (action === 'edit-cat') {
    const newName = prompt(`Ubah nama kategori "${name}":`, name);
    if (!newName || newName.trim() === '' || newName.trim() === name) return;
    try {
      await api(`/api/categories/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newName.trim() })
      });
      showStatus('Nama kategori berhasil diubah');
      await Promise.all([loadCategories(), loadInventory()]);
    } catch (_) {}
  } else if (action === 'del-cat') {
    let confirmMsg = `Hapus kategori "${name}"?`;
    if (count > 0) {
      confirmMsg = `Kategori "${name}" saat ini digunakan oleh ${count} barang toko.\nJika dihapus, barang-barang tersebut dialihkan ke kategori default "Umum".\nLanjutkan?`;
    }
    if (!confirm(confirmMsg)) return;
    try {
      await api(`/api/categories/${id}`, { method: 'DELETE' });
      showStatus('Kategori berhasil dihapus');
      await Promise.all([loadCategories(), loadInventory()]);
    } catch (_) {}
  }
});

async function loadLocations() {
  try {
    locations = await api('/api/locations');
    renderLocationDropdowns();
    renderLocationTable();
  } catch (_) {}
}

function renderLocationDropdowns() {
  if (!selectItemLocation) return;
  const currentFilterVal = filterLocation ? filterLocation.value : '';
  const currentItemVal = selectItemLocation.value;

  if (filterLocation) {
    const filterOpts = ['<option value="">Semua Lokasi / Rak</option>'];
    locations.forEach(l => {
      filterOpts.push('<option value="' + escape(l.name) + '">' + escape(l.name) + '</option>');
    });
    filterLocation.innerHTML = filterOpts.join('');
    filterLocation.value = currentFilterVal;
  }

  const formOpts = ['<option value="">-- Pilih Lokasi / Rak --</option>'];
  locations.forEach(l => {
    formOpts.push('<option value="' + escape(l.name) + '">' + escape(l.name) + '</option>');
  });
  selectItemLocation.innerHTML = formOpts.join('');
  selectItemLocation.value = currentItemVal;
}

function renderLocationTable() {
  if (!locationTbody) return;
  if (locations.length === 0) {
    locationTbody.innerHTML = '<tr><td colspan="4" class="cell-empty">Belum ada lokasi/rak tersimpan.</td></tr>';
    return;
  }

  locationTbody.innerHTML = locations.map(l => {
    return '<tr>' +
      '<td><strong>' + escape(l.name) + '</strong></td>' +
      '<td style="font-size:11px; color:var(--text-muted);">' + escape(l.description || '-') + '</td>' +
      '<td style="text-align: center; font-family: var(--font-mono);">' + l.items_count + '</td>' +
      '<td style="text-align: right;">' +
        '<div class="cell-actions">' +
          '<button type="button" class="btn btn-sm" data-action="edit-loc" data-id="' + l.id + '" data-name="' + escape(l.name) + '" data-desc="' + escape(l.description || '') + '">Edit</button>' +
          '<button type="button" class="btn btn-sm" data-action="del-loc" data-id="' + l.id + '" data-name="' + escape(l.name) + '" data-count="' + l.items_count + '">Hapus</button>' +
        '</div>' +
      '</td>' +
    '</tr>';
  }).join('');
}

if (btnToolbarLocManager) {
  btnToolbarLocManager.addEventListener('click', () => {
    loadLocations();
    modalLocManager.classList.remove('hidden');
  });
}
if (btnFormLocManager) {
  btnFormLocManager.addEventListener('click', () => {
    loadLocations();
    modalLocManager.classList.remove('hidden');
  });
}
if (btnCloseLocModal) btnCloseLocModal.addEventListener('click', () => modalLocManager.classList.add('hidden'));
if (btnDoneLocModal) btnDoneLocModal.addEventListener('click', () => modalLocManager.classList.add('hidden'));

if (formAddLocation) {
  formAddLocation.addEventListener('submit', async e => {
    e.preventDefault();
    const name = inputNewLocName.value.trim();
    const description = inputNewLocDesc ? inputNewLocDesc.value.trim() : '';
    if (!name) return;
    try {
      await api('/api/locations', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, description })
      });
      inputNewLocName.value = '';
      if (inputNewLocDesc) inputNewLocDesc.value = '';
      showStatus('Lokasi/rak baru berhasil ditambahkan');
      await loadLocations();
      selectItemLocation.value = name;
    } catch (_) {}
  });
}

if (locationTbody) {
  locationTbody.addEventListener('click', async e => {
    const btn = e.target.closest('[data-action]');
    if (!btn) return;
    const action = btn.dataset.action;
    const id = btn.dataset.id;
    const name = btn.dataset.name;
    const desc = btn.dataset.desc;
    const count = parseInt(btn.dataset.count, 10) || 0;

    if (action === 'edit-loc') {
      const newName = prompt('Ubah nama lokasi/rak "' + name + '":', name);
      if (!newName || newName.trim() === '') return;
      const newDesc = prompt('Keterangan lokasi/rak:', desc);
      try {
        await api('/api/locations/' + id, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name: newName.trim(), description: (newDesc || '').trim() })
        });
        showStatus('Lokasi/rak berhasil diperbarui');
        await Promise.all([loadLocations(), loadInventory()]);
      } catch (_) {}
    } else if (action === 'del-loc') {
      let confirmMsg = 'Hapus lokasi/rak "' + name + '"?';
      if (count > 0) {
        confirmMsg = 'Lokasi "' + name + '" saat ini digunakan oleh ' + count + ' barang toko.\nBarang-barang ini akan dikosongkan lokasinya.\nLanjutkan?';
      }
      if (!confirm(confirmMsg)) return;
      try {
        await api('/api/locations/' + id, { method: 'DELETE' });
        showStatus('Lokasi berhasil dihapus');
        await Promise.all([loadLocations(), loadInventory()]);
      } catch (_) {}
    }
  });
}



async function loadStats() {
  try {
    const s = await api('/api/stats');
    document.getElementById('stat-total-items').textContent = s.total_items;
    document.getElementById('stat-total-units').textContent = (s.total_units || 0) + ' unit';
    document.getElementById('stat-total-value').textContent = formatRupiah(s.total_value);
    document.getElementById('stat-low-stock').textContent = s.low_stock_count;
  } catch (_) {}
}

async function loadInventory() {
  const p = new URLSearchParams();
  if (filterSearch && filterSearch.value && filterSearch.value.trim()) p.append('search', filterSearch.value.trim());
  if (filterCategory && filterCategory.value) p.append('category', filterCategory.value);

  try {
    let rawItems = await api('/api/items?' + p.toString());
    if (filterLocation && filterLocation.value) {
      rawItems = rawItems.filter(it => it.location === filterLocation.value);
    }
    items = rawItems;
    renderInventory();
    loadCategories();
    loadLocations();
  } catch (_) {}
}

function renderInventory() {
  const invCards = document.getElementById('inventory-cards');
  if (items.length === 0) {
    inventoryTbody.innerHTML = `<tr><td colspan="8" class="cell-empty">Tidak ada barang.</td></tr>`;
    if (invCards) invCards.innerHTML = `<div class="cell-empty">Tidak ada barang.</div>`;
    return;
  }

  inventoryTbody.innerHTML = items.map(it => {
    const isLow = it.quantity <= it.min_threshold;
    const price = it.price || 0;
    const totalVal = it.quantity * price;

    return `
      <tr>
        <td>
          <span style="font-family: var(--font-mono); color: var(--accent-blue);">${escape(it.sku)}</span>
        </td>
        <td>
          <strong>${escape(it.name)}</strong>
          ${it.description ? `<div style="font-size:11px; color:var(--text-muted);">${escape(it.description)}</div>` : ''}
        </td>
        <td><span class="badge">${escape(it.category || 'Umum')}</span></td>
        <td>${escape(it.location || '-')}</td>
        <td style="text-align: right; font-family: var(--font-mono);">${formatRupiah(price)}</td>
        <td style="text-align: right;">
          <span class="${isLow ? 'stock-badge-low' : 'stock-badge-ok'}">${it.quantity} ${escape(it.unit)}</span>
        </td>
        <td style="text-align: right; font-weight: 600; font-family: var(--font-mono); color: var(--success);">${formatRupiah(totalVal)}</td>
        <td style="text-align: right;">
          <div class="cell-actions">
            <button type="button" class="btn btn-sm" data-action="adjust" data-id="${it.id}">+/-</button>
            <button type="button" class="btn btn-sm" data-action="edit" data-id="${it.id}">Edit</button>
            <button type="button" class="btn btn-sm" data-action="delete" data-id="${it.id}">Hapus</button>
          </div>
        </td>
      </tr>
    `;
  }).join('');

  if (invCards) {
    invCards.innerHTML = items.map(it => {
      const isLow = it.quantity <= it.min_threshold;
      const price = it.price || 0;
      const totalVal = it.quantity * price;

      return `
        <div class="mobile-data-card">
          <div class="mobile-card-header">
            <div>
              <div style="font-size:11px; font-family:var(--font-mono); color: var(--accent-blue);">${escape(it.sku)}</div>
              <div class="mobile-card-title">${escape(it.name)}</div>
              ${it.description ? `<div style="font-size:11px; color:var(--text-muted);">${escape(it.description)}</div>` : ''}
            </div>
            <div>
              <span class="${isLow ? 'badge badge-red' : 'badge badge-green'}">${it.quantity} ${escape(it.unit)}</span>
            </div>
          </div>

          <div class="mobile-card-grid">
            <div class="mobile-card-field">
              <span class="mobile-field-label">Kategori</span>
              <span class="mobile-field-val">${escape(it.category || 'Umum')}</span>
            </div>
            <div class="mobile-card-field">
              <span class="mobile-field-label">Lokasi Rak</span>
              <span class="mobile-field-val">${escape(it.location || '-')}</span>
            </div>
            <div class="mobile-card-field">
              <span class="mobile-field-label">Harga Satuan</span>
              <span class="mobile-field-val">${formatRupiah(price)}</span>
            </div>
            <div class="mobile-card-field">
              <span class="mobile-field-label">Total Valuasi</span>
              <span class="mobile-field-val" style="color: var(--success);">${formatRupiah(totalVal)}</span>
            </div>
          </div>

          <div class="mobile-card-actions">
            <button type="button" class="btn btn-sm" data-action="adjust" data-id="${it.id}">+/- Stok</button>
            <button type="button" class="btn btn-sm" data-action="edit" data-id="${it.id}">Edit</button>
            <button type="button" class="btn btn-sm btn-danger" data-action="delete" data-id="${it.id}">&times;</button>
          </div>
        </div>
      `;
    }).join('');
  }
}

function openAdd() {
  formTitle.textContent = 'Tambah Barang';
  itemForm.reset();
  document.getElementById('item-id').value = '';
  selectItemCategory.value = '';
  if (selectItemLocation) selectItemLocation.value = '';
  const unitSel = document.getElementById('item-unit'); if (unitSel) unitSel.value = 'pcs';
  document.getElementById('item-quantity').value = '0';
  document.getElementById('item-min').value = '5';
  document.getElementById('item-price').value = '0';
  formPanel.classList.remove('hidden');
}

function openEdit(it) {
  formTitle.textContent = 'Edit Barang';
  document.getElementById('item-id').value = it.id;
  document.getElementById('item-sku').value = it.sku;
  document.getElementById('item-name').value = it.name;
  selectItemCategory.value = it.category;
  if (selectItemLocation) selectItemLocation.value = it.location || '';
  const unitSel = document.getElementById('item-unit');
  if (unitSel) {
    const hasOpt = Array.from(unitSel.options).some(o => o.value === it.unit);
    if (!hasOpt && it.unit) {
      const customOpt = document.createElement('option');
      customOpt.value = it.unit;
      customOpt.textContent = it.unit;
      unitSel.appendChild(customOpt);
    }
    unitSel.value = it.unit || 'pcs';
  }
  document.getElementById('item-quantity').value = it.quantity;
  document.getElementById('item-quantity').disabled = true;
  document.getElementById('item-min').value = it.min_threshold;
  document.getElementById('item-price').value = it.price || 0;
  document.getElementById('item-description').value = it.description || '';
  formPanel.classList.remove('hidden');
}

btnOpenAdd.addEventListener('click', openAdd);
btnCloseForm.addEventListener('click', () => {
  formPanel.classList.add('hidden');
  document.getElementById('item-quantity').disabled = false;
});
btnCancelForm.addEventListener('click', () => {
  formPanel.classList.add('hidden');
  document.getElementById('item-quantity').disabled = false;
});

itemForm.addEventListener('submit', async e => {
  e.preventDefault();
  const id = document.getElementById('item-id').value;
  const payload = {
    sku: document.getElementById('item-sku').value.trim(),
    name: document.getElementById('item-name').value.trim(),
    category: selectItemCategory.value.trim() || 'Umum',
    location: (selectItemLocation ? selectItemLocation.value : '').trim(),
    unit: document.getElementById('item-unit').value.trim() || 'pcs',
    min_threshold: parseInt(document.getElementById('item-min').value, 10) || 5,
    price: parseFloat(document.getElementById('item-price').value) || 0,
    description: document.getElementById('item-description').value.trim()
  };

  if (!id) {
    payload.quantity = parseInt(document.getElementById('item-quantity').value, 10) || 0;
  }

  try {
    if (id) {
      await api(`/api/items/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      showStatus('Barang toko berhasil diperbarui');
    } else {
      await api('/api/items', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      showStatus('Barang toko berhasil ditambahkan');
    }
    formPanel.classList.add('hidden');
    document.getElementById('item-quantity').disabled = false;
    await Promise.all([loadInventory(), loadStats()]);
  } catch (_) {}
});

async function handleInventoryClickAction(e) {
  const btn = e.target.closest('[data-action]');
  if (!btn) return;
  const action = btn.dataset.action;
  const id = btn.dataset.id;
  const it = items.find(x => String(x.id) === String(id));

  if (action === 'edit' && it) {
    openEdit(it);
  } else if (action === 'delete' && it) {
    if (!confirm(`Hapus barang "${it.name}" dari inventaris toko?`)) return;
    try {
      await api(`/api/items/${id}`, { method: 'DELETE' });
      showStatus('Barang berhasil dihapus');
      await Promise.all([loadInventory(), loadStats()]);
    } catch (_) {}
  } else if (action === 'adjust' && it) {
    movementItemId.value = it.id;
    movementItemName.textContent = `${it.name} (Stok saat ini: ${it.quantity} ${it.unit})`;
    document.getElementById('movement-qty').value = '1';
    document.getElementById('movement-ref').value = '';
    movementDialog.classList.remove('hidden');
  }
}

if (inventoryTbody) inventoryTbody.addEventListener('click', handleInventoryClickAction);
const invCardsContainer = document.getElementById('inventory-cards');
if (invCardsContainer) invCardsContainer.addEventListener('click', handleInventoryClickAction);

btnCloseDialog.addEventListener('click', () => movementDialog.classList.add('hidden'));
btnCancelDialog.addEventListener('click', () => movementDialog.classList.add('hidden'));

movementForm.addEventListener('submit', async e => {
  e.preventDefault();
  const id = movementItemId.value;
  const payload = {
    type: document.getElementById('movement-type').value,
    quantity: parseInt(document.getElementById('movement-qty').value, 10) || 0,
    reference: document.getElementById('movement-ref').value.trim()
  };

  try {
    await api(`/api/items/${id}/movements`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    showStatus('Mutasi stok toko berhasil disimpan');
    movementDialog.classList.add('hidden');
    await Promise.all([loadInventory(), loadStats()]);
  } catch (_) {}
});

filterSearch.addEventListener('input', () => {
  clearTimeout(filterSearch._t);
  filterSearch._t = setTimeout(loadInventory, 200);
});
filterCategory.addEventListener('change', loadInventory);
if (filterLocation) filterLocation.addEventListener('change', loadInventory);

// -------------------------------------------------------------
// View 3: Riwayat Log Terpadu
// -------------------------------------------------------------
const filterUnifiedSearch = document.getElementById('filter-unified-search');
const filterUnifiedType = document.getElementById('filter-unified-type');
const btnRefreshUnifiedLogs = document.getElementById('btn-refresh-unified-logs');
const unifiedLogsTbody = document.getElementById('unified-logs-tbody');

async function loadUnifiedLogs() {
  try {
    const [auditData, movData] = await Promise.all([
      api('/api/audit-logs'),
      api('/api/movements')
    ]);

    const mappedAudit = (auditData || []).map(a => {
      let cat = 'AUDIT';
      if (a.action && a.action.startsWith('LOAN_')) {
        cat = 'LOANS';
      }
      return {
        timestamp: a.created_at,
        category: cat,
        action: a.action,
        entity: a.name || a.sku || 'Item #' + (a.item_id || '-'),
        details: a.details,
        rawType: 'audit'
      };
    });

    const mappedMovements = (movData || []).map(m => {
      return {
        timestamp: m.created_at,
        category: 'MOVEMENTS',
        action: 'MUTASI_' + m.type,
        entity: m.item_name || 'Item #' + m.item_id,
        details: { quantity: m.quantity, reference: m.reference },
        rawType: 'movement'
      };
    });

    unifiedLogs = [...mappedAudit, ...mappedMovements];
    unifiedLogs.sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp));
    renderUnifiedLogs();
  } catch (_) {}
}

function renderUnifiedLogs() {
  const searchTerm = filterUnifiedSearch.value.trim().toLowerCase();
  const selectedType = filterUnifiedType.value;

  const filtered = unifiedLogs.filter(log => {
    if (selectedType !== 'ALL' && log.category !== selectedType) {
      return false;
    }
    if (searchTerm) {
      const ent = (log.entity || '').toLowerCase();
      const act = (log.action || '').toLowerCase();
      const det = JSON.stringify(log.details || '').toLowerCase();
      if (!ent.includes(searchTerm) && !act.includes(searchTerm) && !det.includes(searchTerm)) {
        return false;
      }
    }
    return true;
  });

  if (filtered.length === 0) {
    unifiedLogsTbody.innerHTML = `<tr><td colspan="5" class="cell-empty">Tidak ada log.</td></tr>`;
    return;
  }

  unifiedLogsTbody.innerHTML = filtered.map(log => {
    let catBadge = '';
    if (log.category === 'MOVEMENTS') {
      catBadge = `<span class="badge" style="background:#0284c7; color: var(--text);">MUTASI TOKO</span>`;
    } else if (log.category === 'LOANS') {
      catBadge = `<span class="badge" style="background:#7c3aed; color: var(--text);">RENTAL STEGER</span>`;
    } else {
      catBadge = `<span class="badge" style="background:#4b5563; color: var(--text);">AUDIT DATA</span>`;
    }

    const { actionLabel, actionClass, detailDesc } = formatHumanActivity(log.action, log.details, log.entity);

    return `
      <tr>
        <td style="font-family: var(--font-mono); font-size: 11px; color: var(--text-muted);">
          ${log.timestamp ? log.timestamp.replace('T', ' ').slice(0, 19) : '-'}
        </td>
        <td>${catBadge}</td>
        <td><strong>${escape(log.entity)}</strong></td>
        <td style="text-align: center;"><span class="${actionClass}">${escape(actionLabel)}</span></td>
        <td style="font-size: 12px; color: var(--text);">${detailDesc}</td>
      </tr>
    `;
  }).join('');

  const unifiedLogsCards = document.getElementById('unified-logs-cards');
  if (unifiedLogsCards) {
    unifiedLogsCards.innerHTML = filtered.map(log => {
      let catBadge = '';
      if (log.category === 'MOVEMENTS') {
        catBadge = `<span class="badge" style="background:#0284c7; color: var(--text);">MUTASI TOKO</span>`;
      } else if (log.category === 'LOANS') {
        catBadge = `<span class="badge" style="background:#7c3aed; color: var(--text);">RENTAL STEGER</span>`;
      } else {
        catBadge = `<span class="badge" style="background:#4b5563; color: var(--text);">AUDIT DATA</span>`;
      }

      const { actionLabel, actionClass, detailDesc } = formatHumanActivity(log.action, log.details, log.entity);
      const timeStr = log.timestamp ? log.timestamp.replace('T', ' ').slice(0, 19) : '-';

      return `
        <div class="mobile-data-card">
          <div class="mobile-card-header">
            <div>
              <div style="font-size:10px; font-family:var(--font-mono); color:var(--text-dim);">${timeStr}</div>
              <div class="mobile-card-title">${escape(log.entity || '-')}</div>
            </div>
            <div style="display:flex; gap:4px; align-items:center;">
              ${catBadge}
              <span class="${actionClass}">${escape(actionLabel)}</span>
            </div>
          </div>
          <div style="font-size: 12px; color: #d4d4d8; margin-top: 4px;">${detailDesc}</div>
        </div>
      `;
    }).join('');
  }
}

filterUnifiedSearch.addEventListener('input', () => {
  clearTimeout(filterUnifiedSearch._t);
  filterUnifiedSearch._t = setTimeout(renderUnifiedLogs, 150);
});
filterUnifiedType.addEventListener('change', renderUnifiedLogs);
btnRefreshUnifiedLogs.addEventListener('click', loadUnifiedLogs);

// Inisialisasi awal dengan pemeriksaan autentikasi
const initialHash = window.location.hash.replace('#', '');
const initialView = ['dashboard', 'inventory', 'loans', 'audit'].includes(initialHash) ? initialHash : 'dashboard';

checkAuth().then(authed => {
  if (authed) {
    switchView(initialView);
  }
});

// Keyboard Navigation & Escape Handler for Modals and Forms
document.addEventListener('keydown', e => {
  if (e.key === 'Escape') {
    // Close any open modals
    const openModals = document.querySelectorAll('.dialog-backdrop:not(.hidden)');
    if (openModals.length > 0) {
      openModals.forEach(m => m.classList.add('hidden'));
      return;
    }
    // Close open form panels
    if (formPanel && !formPanel.classList.contains('hidden')) {
      formPanel.classList.add('hidden');
      return;
    }
    if (panelAddLoan && !panelAddLoan.classList.contains('hidden')) {
      panelAddLoan.classList.add('hidden');
      return;
    }
    // Close nav drawer
    if (drawer && drawer.classList.contains('open')) {
      closeDrawer();
      return;
    }
  }

  // Alt + 1, 2, 3, 4 for rapid view switching
  if (e.altKey) {
    if (e.key === '1') { switchView('dashboard'); e.preventDefault(); }
    else if (e.key === '2') { switchView('inventory'); e.preventDefault(); }
    else if (e.key === '3') { switchView('loans'); e.preventDefault(); }
    else if (e.key === '4') { switchView('audit'); e.preventDefault(); }
  }
});


window.loadCentralDashboard = loadCentralDashboard;
window.loadLoans = loadLoans;
window.loadInventory = loadInventory;
window.loadStats = loadStats;
window.loadUnifiedLogs = loadUnifiedLogs;


// Modal Kelola Armada Kapolding
const modalFleetManager = document.getElementById('modal-fleet-manager');
const btnOpenFleetModal = document.getElementById('btn-open-fleet-modal');
const btnCloseFleetModal = document.getElementById('btn-close-fleet-modal');
const btnDoneFleetModal = document.getElementById('btn-done-fleet-modal');
const fleetManagerTbody = document.getElementById('fleet-manager-tbody');

async function renderFleetModalTable() {
  if (!fleetManagerTbody) return;
  try {
    const fleet = await api('/api/fleet');
    fleetManagerTbody.innerHTML = fleet.map(f => `
      <tr>
        <td><strong>${escape(f.name)}</strong> <span style="font-size:10.5px; font-family:var(--font-mono); color:var(--text-dim);">(${escape(f.code)})</span></td>
        <td style="text-align:center; font-family:var(--font-mono); font-weight:600; color:var(--success);">${f.available_in_warehouse} ${escape(f.unit)}</td>
        <td style="text-align:center; font-family:var(--font-mono); font-weight:600; color:var(--warning);">${f.currently_rented} ${escape(f.unit)}</td>
        <td style="text-align:center; font-family:var(--font-mono); font-weight:700;">${f.total_owned} ${escape(f.unit)}</td>
        <td style="text-align:right;">
          <button type="button" class="btn btn-sm btn-quick-fleet" data-code="${f.code}">Ubah Total</button>
        </td>
      </tr>
    `).join('');
  } catch (_) {}
}

async function openFleetModal() {
  if (!modalFleetManager) return;
  modalFleetManager.classList.remove('hidden');
  await renderFleetModalTable();
}

if (btnOpenFleetModal) btnOpenFleetModal.addEventListener('click', openFleetModal);
if (btnCloseFleetModal) btnCloseFleetModal.addEventListener('click', () => modalFleetManager.classList.add('hidden'));
if (btnDoneFleetModal) btnDoneFleetModal.addEventListener('click', () => modalFleetManager.classList.add('hidden'));


// -------------------------------------------------------------
// Modul Autentikasi & Sesi Pengguna
// -------------------------------------------------------------
let currentUser = null;
const authScreen = document.getElementById('auth-screen');
const appWorkspace = document.getElementById('app-workspace');
const formLogin = document.getElementById('form-login');
const loginErrorMsg = document.getElementById('login-error-msg');
const inputLoginUser = document.getElementById('login-username');
const inputLoginPass = document.getElementById('login-password');
const sidebarUserName = document.getElementById('sidebar-user-name');
const btnLogout = document.getElementById('btn-logout');

const modalChangePassword = document.getElementById('modal-change-password');
const btnOpenPwdModal = document.getElementById('btn-open-pwd-modal');
const btnClosePwdModal = document.getElementById('btn-close-pwd-modal');
const btnCancelPwdModal = document.getElementById('btn-cancel-pwd-modal');
const formChangePassword = document.getElementById('form-change-password');
const pwdErrorMsg = document.getElementById('pwd-error-msg');

function showAuthScreen() {
  if (authScreen) authScreen.classList.remove('hidden');
  if (appWorkspace) appWorkspace.classList.add('hidden');
  if (inputLoginPass) inputLoginPass.value = '';
}

function hideAuthScreen() {
  if (authScreen) authScreen.classList.add('hidden');
  if (appWorkspace) appWorkspace.classList.remove('hidden');
}

async function checkAuth() {
  try {
    const res = await fetch('/api/auth/me');
    const data = await res.json();
    if (data && data.authenticated) {
      currentUser = data.username;
      if (sidebarUserName) sidebarUserName.textContent = data.username;
      hideAuthScreen();
      return true;
    } else {
      currentUser = null;
      showAuthScreen();
      return false;
    }
  } catch (_) {
    showAuthScreen();
    return false;
  }
}

if (formLogin) {
  formLogin.addEventListener('submit', async e => {
    e.preventDefault();
    loginErrorMsg.classList.add('hidden');
    loginErrorMsg.textContent = '';

    const username = inputLoginUser.value.trim();
    const password = inputLoginPass.value;

    try {
      const res = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password })
      });
      const data = await res.json();
      if (!res.ok || !data.success) {
        throw new Error(data.error || 'Username atau password salah');
      }

      currentUser = data.username;
      if (sidebarUserName) sidebarUserName.textContent = data.username;
      hideAuthScreen();
      inputLoginPass.value = '';
      showStatus('Selamat datang, ' + currentUser);
      switchView(currentView || 'dashboard');
    } catch (err) {
      loginErrorMsg.textContent = err.message;
      loginErrorMsg.classList.remove('hidden');
    }
  });
}

if (btnLogout) {
  btnLogout.addEventListener('click', async () => {
    try {
      await fetch('/api/auth/logout', { method: 'POST' });
    } catch (_) {}
    currentUser = null;
    showAuthScreen();
    showStatus('Berhasil keluar');
  });
}

if (btnOpenPwdModal) {
  btnOpenPwdModal.addEventListener('click', () => {
    if (modalChangePassword) {
      modalChangePassword.classList.remove('hidden');
      if (pwdErrorMsg) pwdErrorMsg.classList.add('hidden');
      if (formChangePassword) formChangePassword.reset();
    }
  });
}

if (btnClosePwdModal) btnClosePwdModal.addEventListener('click', () => modalChangePassword.classList.add('hidden'));
if (btnCancelPwdModal) btnCancelPwdModal.addEventListener('click', () => modalChangePassword.classList.add('hidden'));

if (formChangePassword) {
  formChangePassword.addEventListener('submit', async e => {
    e.preventDefault();
    pwdErrorMsg.classList.add('hidden');
    pwdErrorMsg.textContent = '';

    const currentPassword = document.getElementById('input-current-pwd').value;
    const newPassword = document.getElementById('input-new-pwd').value;
    const confirmPassword = document.getElementById('input-confirm-pwd').value;

    if (newPassword !== confirmPassword) {
      pwdErrorMsg.textContent = 'Konfirmasi password baru tidak cocok';
      pwdErrorMsg.classList.remove('hidden');
      return;
    }

    try {
      const res = await fetch('/api/auth/change-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ current_password: currentPassword, new_password: newPassword })
      });
      const data = await res.json();
      if (!res.ok || !data.success) {
        throw new Error(data.error || 'Gagal mengubah password');
      }

      modalChangePassword.classList.add('hidden');
      showStatus('Password berhasil diperbarui');
      formChangePassword.reset();
    } catch (err) {
      pwdErrorMsg.textContent = err.message;
      pwdErrorMsg.classList.remove('hidden');
    }
  });
}


// -------------------------------------------------------------
// Export CSV (pemisah ";" + BOM supaya langsung rapi di Excel Indonesia)
// -------------------------------------------------------------
function downloadCsv(filename, headers, rows) {
  if (!rows.length) {
    showStatus('Tidak ada data untuk diexport', true);
    return;
  }
  const cell = (v) => {
    let s = String(v ?? '');
    if (/^[=+\-@\t\r]/.test(s)) s = "'" + s; // cegah formula injection di Excel
    return '"' + s.replace(/"/g, '""') + '"';
  };
  const lines = [headers, ...rows].map(r => r.map(cell).join(';'));
  const blob = new Blob(['\uFEFF' + lines.join('\r\n')], { type: 'text/csv;charset=utf-8' });
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = filename;
  a.click();
  setTimeout(() => URL.revokeObjectURL(a.href), 1000);
  showStatus(`${rows.length} baris diexport ke ${filename}`);
}

const exportStamp = () => new Date().toISOString().slice(0, 10);
const fmtDate = (s) => (s ? String(s).slice(0, 10) : '');
const fmtTime = (s) => (s ? String(s).replace('T', ' ').slice(0, 19) : '');

// Stok Toko — sesuai filter yang sedang tampil
document.getElementById('btn-export-inventory').addEventListener('click', () => {
  downloadCsv(`stok-toko-${exportStamp()}.csv`,
    ['SKU', 'Nama Barang', 'Kategori', 'Lokasi', 'Stok', 'Satuan', 'Batas Minimum', 'Harga', 'Nilai Stok', 'Status', 'Keterangan'],
    items.map(it => [it.sku, it.name, it.category, it.location, it.quantity, it.unit, it.min_threshold,
      it.price || 0, it.quantity * (it.price || 0), it.quantity <= it.min_threshold ? 'Kritis' : 'Aman', it.description]));
});

// Sewa Kapolding — periode & filter yang sedang tampil
document.getElementById('btn-export-loans').addEventListener('click', () => {
  downloadCsv(`sewa-kapolding-${currentLoanPeriod}.csv`,
    ['Kode', 'Peminjam', 'Telepon', 'Proyek', 'Barang', 'Set', 'Siku', 'Shock', 'Tgl Sewa', 'Jatuh Tempo', 'Tgl Kembali',
      'Status', 'KTP Ditahan', 'Biaya Sewa', 'Biaya Pemilik', 'Sudah Bayar', 'Sudah Setor Pemilik', 'Rollover', 'Catatan'],
    loans.map(l => [l.loan_code, l.borrower_name, l.borrower_phone, l.project_location, l.item_name,
      l.quantity, l.elbow_count, l.shock_count, fmtDate(l.loan_date), fmtDate(l.due_date), fmtDate(l.return_date),
      l.status === 'RETURNED' ? 'Kembali' : 'Aktif', l.id_card_given ? 'Ya' : 'Tidak',
      l.rental_fee, l.owner_cost, l.is_paid ? 'Ya' : 'Belum', l.is_paid_to_owner ? 'Ya' : 'Belum',
      l.is_rollover ? 'Ya' : 'Tidak', l.notes]));
});

// Riwayat Log — sesuai pencarian & jenis log yang sedang tampil
document.getElementById('btn-export-logs').addEventListener('click', () => {
  const term = filterUnifiedSearch.value.trim().toLowerCase();
  const type = filterUnifiedType.value;
  const rows = unifiedLogs
    .filter(log => (type === 'ALL' || log.category === type) &&
      (!term || `${log.entity} ${log.action} ${JSON.stringify(log.details || '')}`.toLowerCase().includes(term)))
    .map(log => {
      const { actionLabel, detailDesc } = formatHumanActivity(log.action, log.details, log.entity);
      return [fmtTime(log.timestamp), log.category, actionLabel, log.entity, detailDesc];
    });
  downloadCsv(`riwayat-log-${exportStamp()}.csv`, ['Waktu', 'Kategori', 'Aksi', 'Barang / Sewa', 'Detail'], rows);
});
