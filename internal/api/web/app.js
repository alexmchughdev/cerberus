const packetLabels = [
  ["total", "Total"],
  ["arp", "ARP"],
  ["tcp", "TCP"],
  ["udp", "UDP"],
  ["icmp", "ICMP"],
  ["dns", "DNS"],
  ["http", "HTTP"],
  ["tls", "TLS"],
];

function renderStats(packetStats, deviceCount) {
  const root = document.getElementById("stats-grid");
  const cards = packetLabels
    .map(([key, label]) => {
      const value = key === "totalDevices" ? deviceCount : packetStats[key] || 0;
      return `<article class="stat"><div class="stat-label">${label}</div><div class="stat-value">${value}</div></article>`;
    })
    .join("");
  root.innerHTML =
    cards +
    `<article class="stat"><div class="stat-label">Devices</div><div class="stat-value">${deviceCount}</div></article>`;
}

function renderRankList(id, values) {
  const root = document.getElementById(id);
  const entries = Object.entries(values || {});
  if (!entries.length) {
    root.innerHTML = "<li><span>No data yet</span><span class='badge'>0</span></li>";
    return;
  }
  root.innerHTML = entries
    .map(([name, count]) => `<li><span>${name}</span><span class="badge">${count}</span></li>`)
    .join("");
}

function renderDevices(items) {
  const root = document.getElementById("recent-devices");
  if (!items.length) {
    root.innerHTML = "<p class='muted'>No devices observed yet.</p>";
    return;
  }
  root.innerHTML = items
    .map(
      (d) => `
      <article class="device">
        <strong>${d.ip} · ${d.vendor || "Unknown"}</strong>
        <div class="line"><span>MAC ${d.mac}</span><span>DNS ${d.dns_queries}</span></div>
        <div class="line"><span>HTTP ${d.http_requests}</span><span>TLS ${d.tls}</span></div>
      </article>
    `,
    )
    .join("");
}

async function refresh() {
  const res = await fetch("/api/v1/summary", { headers: { Accept: "application/json" } });
  if (!res.ok) return;
  const data = await res.json();
  document.getElementById("generated-at").textContent = `Last refresh: ${new Date(
    data.generated_at,
  ).toLocaleTimeString()}`;
  renderStats(data.packet_stats, data.device_count);
  renderRankList("top-services", data.top_services);
  renderRankList("top-vendors", data.top_vendors);
  renderDevices(data.recent_devices || []);
}

function setupThemeToggle() {
  const btn = document.getElementById("theme-toggle");
  btn.addEventListener("click", () => {
    const isDark = document.documentElement.getAttribute("data-theme") === "dark";
    document.documentElement.setAttribute("data-theme", isDark ? "light" : "dark");
    btn.textContent = isDark ? "Dark Mode" : "Light Mode";
  });
}

setupThemeToggle();
refresh();
setInterval(refresh, 3000);
