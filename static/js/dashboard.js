// Dashboard functionality for the network monitor

function updateDashboard() {
  const container = document.getElementById("statsCards");
  container.innerHTML = "";

  stats.forEach((stat) => {
    const card = document.createElement("div");
    card.className = "card";

    const uptime = ((stat.successful_pings / stat.total_pings) * 100).toFixed(
      2
    );
    const statusClass =
      uptime >= 99
        ? "status-good"
        : uptime >= 95
        ? "status-warning"
        : "status-bad";

    card.innerHTML = `
            <h2>${stat.target}</h2>
            <div class="stat-value ${statusClass}">${uptime}%</div>
            <div class="stat-label">${stat.total_pings} pings</div>
            <div style="margin-top: 15px; font-size: 14px; color: #666;">
                <div>Avg RTT: ${stat.avg_rtt?.toFixed(1) || "N/A"} ms</div>
                <div>Min/Max: ${stat.min_rtt?.toFixed(1) || "N/A"}/${
      stat.max_rtt?.toFixed(1) || "N/A"
    } ms</div>
                <div>Packet Loss: ${stat.packet_loss.toFixed(1)}%</div>
            </div>
        `;
    container.appendChild(card);
  });
}

function updateOutagesTable() {
  const tbody = document.querySelector("#outagesTable tbody");
  tbody.innerHTML = "";

  if (!outages || outages.length === 0) {
    tbody.innerHTML =
      '<tr><td colspan="4" style="text-align: center; color: #999;">No outages detected</td></tr>';
    return;
  }

  outages.slice(0, 20).forEach((outage) => {
    const row = tbody.insertRow();
    const startTime = new Date(outage.start_time);
    const duration = parseDuration(outage.duration);

    row.innerHTML = `
            <td>${outage.target}</td>
            <td>${startTime.toLocaleString()}</td>
            <td>${duration}</td>
            <td>${outage.failed_checks}</td>
        `;
  });
}

function refreshData() {
  const hours = document.getElementById("timeRange").value;
  fetchData(hours).then(() => {
    updateDashboard();
    drawLatencyChart();
    drawAvailabilityChart();
    drawHeatmap();
    updateOutagesTable();
  });
}

// Event listeners
function setupEventListeners() {
  document.getElementById("timeRange").addEventListener("change", refreshData);
  document
    .getElementById("heatmapDays")
    .addEventListener("change", refreshData);
}
