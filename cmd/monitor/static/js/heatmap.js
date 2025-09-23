// Heatmap visualization functions for the network monitor dashboard

function drawHeatmap() {
  const container = d3.select("#heatmap-chart");
  container.selectAll("*").remove();

  if (!heatmapData || heatmapData.length === 0) {
    container
      .append("text")
      .attr("x", 20)
      .attr("y", 30)
      .text(
        "Insufficient data for heatmap. Data will appear after an hour of monitoring."
      );
    return;
  }

  const margin = { top: 50, right: 150, bottom: 50, left: 100 };
  const width =
    container.node().getBoundingClientRect().width - margin.left - margin.right;
  const cellSize = Math.floor(width / 24);
  const targets = [...new Set(heatmapData.map((d) => d.target))];
  const height = cellSize * targets.length;

  const svg = container
    .attr("width", width + margin.left + margin.right)
    .attr("height", height + margin.top + margin.bottom);

  const g = svg
    .append("g")
    .attr("transform", `translate(${margin.left},${margin.top})`);

  // Create scales
  const xScale = d3
    .scaleBand()
    .domain(d3.range(24))
    .range([0, width])
    .padding(0.05);

  const yScale = d3
    .scaleBand()
    .domain(targets)
    .range([0, height])
    .padding(0.05);

  // Color scale for failure rate
  const colorScale = d3
    .scaleSequential()
    .domain([0, 20]) // 0-20% failure rate
    .interpolator(d3.interpolateRgb("#10b981", "#ef4444"))
    .clamp(true);

  // Alternative: Color scale for latency
  const latencyColorScale = d3
    .scaleSequential()
    .domain([0, 100]) // 0-100ms latency
    .interpolator(d3.interpolateRgb("#3b82f6", "#f59e0b"))
    .clamp(true);

  // Draw cells
  const cells = g
    .selectAll(".heatmap-cell")
    .data(heatmapData)
    .enter()
    .append("rect")
    .attr("class", "heatmap-cell")
    .attr("x", (d) => xScale(d.hour))
    .attr("y", (d) => yScale(d.target))
    .attr("width", xScale.bandwidth())
    .attr("height", yScale.bandwidth())
    .attr("fill", (d) => {
      // Use failure rate as primary indicator
      if (d.failure_rate > 0) {
        return colorScale(d.failure_rate);
      }
      // If no failures, show latency
      return d.avg_latency > 50 ? latencyColorScale(d.avg_latency) : "#10b981";
    })
    .attr("opacity", (d) => {
      // Make opacity based on data density
      return Math.min(1, 0.3 + (d.days_with_data / 30) * 0.7);
    })
    .on("mouseover", function (event, d) {
      const days = document.getElementById("heatmapDays").value;
      tooltip.style("opacity", 1).html(`
              <strong>${d.target} - ${d.hour}:00</strong><br/>
              Failure Rate: ${d.failure_rate.toFixed(1)}%<br/>
              Avg Latency: ${d.avg_latency.toFixed(1)} ms<br/>
              Max Latency: ${d.max_latency.toFixed(1)} ms<br/>
              Failed/Total: ${d.total_failures}/${d.total_pings}<br/>
              Days with data: ${d.days_with_data}/${days}
          `);
    })
    .on("mousemove", function (event) {
      tooltip
        .style("left", event.pageX + 10 + "px")
        .style("top", event.pageY - 28 + "px");
    })
    .on("mouseout", function () {
      tooltip.style("opacity", 0);
    })
    .on("click", function (event, d) {
      showPatternDetails(d.hour);
    });

  // Add text labels for significant issues
  g.selectAll(".issue-label")
    .data(heatmapData.filter((d) => d.failure_rate > 10))
    .enter()
    .append("text")
    .attr("class", "issue-label")
    .attr("x", (d) => xScale(d.hour) + xScale.bandwidth() / 2)
    .attr("y", (d) => yScale(d.target) + yScale.bandwidth() / 2)
    .attr("text-anchor", "middle")
    .attr("dominant-baseline", "middle")
    .style("fill", "white")
    .style("font-size", "10px")
    .style("font-weight", "bold")
    .style("pointer-events", "none")
    .text((d) => Math.round(d.failure_rate) + "%");

  // X-axis (hours)
  g.append("g")
    .attr("transform", `translate(0,${height})`)
    .call(d3.axisBottom(xScale).tickFormat((d) => d + ":00"))
    .selectAll("text")
    .attr("class", "heatmap-label")
    .style("text-anchor", "end")
    .attr("dx", "-.8em")
    .attr("dy", ".15em")
    .attr("transform", "rotate(-45)");

  // Y-axis (targets)
  g.append("g")
    .call(d3.axisLeft(yScale))
    .selectAll("text")
    .attr("class", "heatmap-label");

  // Title
  g.append("text")
    .attr("x", width / 2)
    .attr("y", -margin.top / 2)
    .attr("text-anchor", "middle")
    .style("font-size", "14px")
    .style("font-weight", "bold")
    .text("Connectivity Issues by Hour of Day");

  // Color legend
  const legendWidth = 100;
  const legendHeight = 20;

  const legendScale = d3.scaleLinear().domain([0, 20]).range([0, legendWidth]);

  const legendAxis = d3
    .axisBottom(legendScale)
    .ticks(5)
    .tickFormat((d) => d + "%");

  const legend = svg
    .append("g")
    .attr("class", "heatmap-legend")
    .attr("transform", `translate(${width + margin.left + 20}, ${margin.top})`);

  // Gradient definition
  const defs = svg.append("defs");
  const gradient = defs
    .append("linearGradient")
    .attr("id", "heatmap-gradient")
    .attr("x1", "0%")
    .attr("x2", "100%")
    .attr("y1", "0%")
    .attr("y2", "0%");

  const stops = d3.range(0, 1.1, 0.1);
  stops.forEach((stop) => {
    gradient
      .append("stop")
      .attr("offset", stop * 100 + "%")
      .attr("stop-color", colorScale(stop * 20));
  });

  legend
    .append("rect")
    .attr("width", legendWidth)
    .attr("height", legendHeight)
    .style("fill", "url(#heatmap-gradient)");

  legend
    .append("g")
    .attr("transform", `translate(0,${legendHeight})`)
    .call(legendAxis);

  legend
    .append("text")
    .attr("x", legendWidth / 2)
    .attr("y", -5)
    .attr("text-anchor", "middle")
    .style("font-size", "12px")
    .text("Failure Rate");

  // Add note about clicking for details
  svg
    .append("text")
    .attr("x", margin.left)
    .attr("y", height + margin.top + margin.bottom - 5)
    .style("font-size", "11px")
    .style("fill", "#999")
    .text("Click any cell to see detailed daily breakdown for that hour");
}
