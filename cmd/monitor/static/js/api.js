// API communication functions for the network monitor dashboard

let currentData = [];
let stats = [];
let outages = [];
let heatmapData = [];

async function fetchData(hours = 24) {
  try {
    const heatmapDays = document.getElementById("heatmapDays").value;
    const [recentRes, statsRes, outagesRes, heatmapRes] = await Promise.all([
      fetch(`/api/recent?hours=${hours}`),
      fetch("/api/stats"),
      fetch("/api/outages"),
      fetch(`/api/heatmap?days=${heatmapDays}`),
    ]);

    currentData = await recentRes.json();
    stats = await statsRes.json();
    outages = await outagesRes.json();
    heatmapData = await heatmapRes.json();

    // Parse timestamps for currentData
    const parseTime = d3.timeParse("%Y-%m-%dT%H:%M:%S");
    currentData.forEach((d) => {
      if (d.timestamp) {
        d.date = parseTime(d.timestamp.split(".")[0]);
      }
    });

    return { currentData, stats, outages, heatmapData };
  } catch (error) {
    console.error("Error fetching data:", error);
    throw error;
  }
}

async function showPatternDetails(hour) {
  try {
    const response = await fetch(`/api/patterns?hour=${hour}`);
    const patterns = await response.json();

    if (!patterns || patterns.length === 0) return;

    // Create or update details panel
    let detailsDiv = document.getElementById("pattern-details");
    if (!detailsDiv) {
      detailsDiv = document.createElement("div");
      detailsDiv.id = "pattern-details";
      detailsDiv.className = "pattern-details";
      document.querySelector(".container").appendChild(detailsDiv);
    }

    // Group by target
    const byTarget = {};
    patterns.forEach((p) => {
      if (!byTarget[p.target]) byTarget[p.target] = [];
      byTarget[p.target].push(p);
    });

    let html = `<h3>Details for ${hour}:00</h3>`;

    Object.keys(byTarget).forEach((target) => {
      const targetPatterns = byTarget[target];
      const avgFailure =
        targetPatterns.reduce((sum, p) => sum + p.failure_rate, 0) /
        targetPatterns.length;

      html += `<h4>${target} (Avg failure: ${avgFailure.toFixed(1)}%)</h4>`;
      html += '<table style="width: 100%; font-size: 12px;">';
      html +=
        "<tr><th>Date</th><th>Failure Rate</th><th>Avg RTT</th><th>Max RTT</th></tr>";

      targetPatterns.slice(0, 10).forEach((p) => {
        const rowClass =
          p.failure_rate > 10
            ? 'style="background: #fee;"'
            : p.failure_rate > 5
            ? 'style="background: #ffc;"'
            : "";
        html += `<tr ${rowClass}>
                        <td>${p.date}</td>
                        <td>${p.failure_rate.toFixed(1)}%</td>
                        <td>${p.avg_rtt.toFixed(1)} ms</td>
                        <td>${p.max_rtt.toFixed(1)} ms</td>
                    </tr>`;
      });

      if (targetPatterns.length > 10) {
        html += `<tr><td colspan="4" style="text-align: center; color: #999;">
                            ... and ${
                              targetPatterns.length - 10
                            } more days</td></tr>`;
      }

      html += "</table>";
    });

    html +=
      '<button onclick="document.getElementById(\'pattern-details\').remove()" style="margin-top: 10px;">Close</button>';

    detailsDiv.innerHTML = html;
    detailsDiv.scrollIntoView({ behavior: "smooth" });
  } catch (error) {
    console.error("Error fetching pattern details:", error);
  }
}

function generateReport() {
  // This would trigger server-side report generation
  window.open(
    `/api/report?hours=${document.getElementById("timeRange").value}`,
    "_blank"
  );
}
