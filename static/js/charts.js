// Chart rendering functions for the network monitor dashboard

const colorScale = d3.scaleOrdinal(d3.schemeCategory10);
const tooltip = d3.select(".tooltip");

function drawLatencyChart() {
  const container = d3.select("#latency-chart");
  container.selectAll("*").remove();

  const margin = { top: 20, right: 120, bottom: 40, left: 60 };
  const width =
    container.node().getBoundingClientRect().width - margin.left - margin.right;
  const height = 300 - margin.top - margin.bottom;

  const svg = container
    .attr("width", width + margin.left + margin.right)
    .attr("height", height + margin.top + margin.bottom);

  const g = svg
    .append("g")
    .attr("transform", `translate(${margin.left},${margin.top})`);

  // Get current time range and aggregate data
  const hours = parseInt(document.getElementById("timeRange").value);
  const aggregatedData = aggregateData(currentData, hours);

  const targets = Object.keys(aggregatedData);
  if (targets.length === 0) {
    g.append("text")
      .attr("x", width / 2)
      .attr("y", height / 2)
      .attr("text-anchor", "middle")
      .style("font-size", "14px")
      .style("fill", "#999")
      .text("No data available for the selected time range");
    return;
  }

  const dataByTarget = targets.map((target) => ({
    target: target,
    values: aggregatedData[target].filter((d) => d.hasData),
  }));

  // Filter out targets with no data
  const validTargets = dataByTarget.filter((d) => d.values.length > 0);
  if (validTargets.length === 0) {
    g.append("text")
      .attr("x", width / 2)
      .attr("y", height / 2)
      .attr("text-anchor", "middle")
      .style("font-size", "14px")
      .style("fill", "#999")
      .text("No latency data available for the selected time range");
    return;
  }

  // Scales
  const allValidValues = validTargets.flatMap((d) => d.values);
  const x = d3
    .scaleTime()
    .domain(d3.extent(allValidValues, (d) => d.date))
    .range([0, width]);

  // Smart Y-axis scaling - use 95th percentile to avoid outlier skewing
  const maxLatencies = allValidValues.map((d) => d.p95 || d.avg);
  const yMax = d3.quantile(maxLatencies.sort(d3.ascending), 0.95) * 1.1;

  const y = d3.scaleLinear().domain([0, yMax]).nice().range([height, 0]);

  // Grid
  g.append("g")
    .attr("class", "grid")
    .attr("transform", `translate(0,${height})`)
    .call(d3.axisBottom(x).tickSize(-height).tickFormat(""));

  g.append("g")
    .attr("class", "grid")
    .call(d3.axisLeft(y).tickSize(-width).tickFormat(""));

  // Axes
  g.append("g")
    .attr("transform", `translate(0,${height})`)
    .attr("class", "axis")
    .call(d3.axisBottom(x).tickFormat(d3.timeFormat("%H:%M")));

  g.append("g").attr("class", "axis").call(d3.axisLeft(y));

  // Y-axis label
  g.append("text")
    .attr("transform", "rotate(-90)")
    .attr("y", 0 - margin.left)
    .attr("x", 0 - height / 2)
    .attr("dy", "1em")
    .style("text-anchor", "middle")
    .style("font-size", "12px")
    .text("Latency (ms)");

  // Line generator for average latency
  const line = d3
    .line()
    .x((d) => x(d.date))
    .y((d) => y(d.avg))
    .curve(d3.curveMonotoneX)
    .defined((d) => d.hasData && d.avg !== null);

  // Area generator for confidence bands (min/max range)
  const area = d3
    .area()
    .x((d) => x(d.date))
    .y0((d) => y(Math.min(d.min, yMax)))
    .y1((d) => y(Math.min(d.max, yMax)))
    .curve(d3.curveMonotoneX)
    .defined((d) => d.hasData && d.min !== null && d.max !== null);

  // Draw confidence bands (min/max range) for each target
  validTargets.forEach((targetData) => {
    g.append("path")
      .datum(targetData.values)
      .attr("class", "confidence-band")
      .attr("d", area)
      .style("fill", colorScale(targetData.target))
      .style("opacity", 0.15)
      .style("stroke", "none");
  });

  // Draw average lines for each target
  validTargets.forEach((targetData) => {
    g.append("path")
      .datum(targetData.values)
      .attr("class", "line")
      .attr("d", line)
      .style("stroke", colorScale(targetData.target))
      .style("stroke-width", 2)
      .style("fill", "none");
  });

  // Add interactive dots for hover
  validTargets.forEach((targetData) => {
    g.selectAll(`.dots-${targetData.target.replace(/\./g, "-")}`)
      .data(targetData.values)
      .enter()
      .append("circle")
      .attr("class", `dots-${targetData.target.replace(/\./g, "-")}`)
      .attr("cx", (d) => x(d.date))
      .attr("cy", (d) => y(d.avg))
      .attr("r", 4)
      .style("fill", colorScale(targetData.target))
      .style("stroke", "white")
      .style("stroke-width", 2)
      .style("opacity", 0)
      .on("mouseover", function (event, d) {
        // Highlight this dot
        d3.select(this).style("opacity", 1);

        // Show tooltip
        const successPercent = (d.successRate * 100).toFixed(1);
        tooltip.style("opacity", 1).html(
          `<strong>${targetData.target}</strong><br/>
          <strong>Average:</strong> ${d.avg.toFixed(1)} ms<br/>
          <strong>Median:</strong> ${d.median.toFixed(1)} ms<br/>
          <strong>Range:</strong> ${d.min.toFixed(1)} - ${d.max.toFixed(
            1
          )} ms<br/>
          <strong>95th percentile:</strong> ${d.p95.toFixed(1)} ms<br/>
          <strong>Success rate:</strong> ${successPercent}%<br/>
          <strong>Samples:</strong> ${d.successCount}/${d.count}<br/>
          <strong>Time:</strong> ${d3.timeFormat("%H:%M")(d.date)}`
        );
      })
      .on("mousemove", function (event) {
        tooltip
          .style("left", event.pageX + 10 + "px")
          .style("top", event.pageY - 28 + "px");
      })
      .on("mouseout", function () {
        d3.select(this).style("opacity", 0);
        tooltip.style("opacity", 0);
      });
  });

  // Add hover line for better time reference
  const hoverLine = g
    .append("line")
    .attr("class", "hover-line")
    .style("stroke", "#999")
    .style("stroke-width", 1)
    .style("stroke-dasharray", "3,3")
    .style("opacity", 0);

  // Invisible overlay for mouse tracking
  g.append("rect")
    .attr("width", width)
    .attr("height", height)
    .style("fill", "none")
    .style("pointer-events", "all")
    .on("mousemove", function (event) {
      const [mouseX] = d3.pointer(event);
      const xDate = x.invert(mouseX);

      // Show hover line
      hoverLine
        .attr("x1", mouseX)
        .attr("x2", mouseX)
        .attr("y1", 0)
        .attr("y2", height)
        .style("opacity", 0.5);

      // Find closest data points and highlight them
      validTargets.forEach((targetData) => {
        const bisect = d3.bisector((d) => d.date).left;
        const i = bisect(targetData.values, xDate, 1);
        const d0 = targetData.values[i - 1];
        const d1 = targetData.values[i];

        if (d0 && d1) {
          const d = xDate - d0.date > d1.date - xDate ? d1 : d0;
          g.selectAll(`.dots-${targetData.target.replace(/\./g, "-")}`).style(
            "opacity",
            (dot) => (dot === d ? 1 : 0)
          );
        }
      });
    })
    .on("mouseleave", function () {
      hoverLine.style("opacity", 0);
      g.selectAll("[class*='dots-']").style("opacity", 0);
      tooltip.style("opacity", 0);
    });

  // Enhanced Legend with toggle functionality
  const legend = svg
    .append("g")
    .attr("class", "legend")
    .attr("transform", `translate(${width + margin.left + 10}, ${margin.top})`);

  validTargets.forEach((targetData, i) => {
    const legendRow = legend
      .append("g")
      .attr("transform", `translate(0, ${i * 25})`)
      .style("cursor", "pointer");

    legendRow
      .append("rect")
      .attr("width", 12)
      .attr("height", 12)
      .attr("fill", colorScale(targetData.target))
      .attr("rx", 2);

    legendRow
      .append("text")
      .attr("x", 18)
      .attr("y", 9)
      .style("font-size", "12px")
      .style("dominant-baseline", "middle")
      .text(targetData.target);

    // Add stats to legend
    const avgLatency = d3.mean(targetData.values, (d) => d.avg);
    const avgSuccess = d3.mean(targetData.values, (d) => d.successRate) * 100;

    legendRow
      .append("text")
      .attr("x", 18)
      .attr("y", 21)
      .style("font-size", "10px")
      .style("fill", "#666")
      .text(`${avgLatency.toFixed(1)}ms, ${avgSuccess.toFixed(1)}% up`);

    // Toggle functionality
    let visible = true;
    legendRow.on("click", function () {
      visible = !visible;
      const opacity = visible ? 1 : 0.1;

      // Toggle line visibility
      g.selectAll(".line")
        .filter((d, j) => j === i)
        .style("opacity", opacity);

      // Toggle confidence band visibility
      g.selectAll(".confidence-band")
        .filter((d, j) => j === i)
        .style("opacity", visible ? 0.15 : 0.02);

      // Update legend appearance
      legendRow.select("rect").style("opacity", opacity);
      legendRow.selectAll("text").style("opacity", opacity);
    });
  });

  // Add chart info
  g.append("text")
    .attr("x", width)
    .attr("y", -5)
    .attr("text-anchor", "end")
    .style("font-size", "11px")
    .style("fill", "#666")
    .text("Click legend items to toggle • Hover for details");
}

function drawAvailabilityChart() {
  const container = d3.select("#availability-chart");
  container.selectAll("*").remove();

  const margin = { top: 20, right: 120, bottom: 40, left: 60 };
  const width =
    container.node().getBoundingClientRect().width - margin.left - margin.right;
  const height = 300 - margin.top - margin.bottom;

  const svg = container
    .attr("width", width + margin.left + margin.right)
    .attr("height", height + margin.top + margin.bottom);

  const g = svg
    .append("g")
    .attr("transform", `translate(${margin.left},${margin.top})`);

  const targets = [...new Set(currentData.map((d) => d.target))];
  const targetHeight = height / targets.length;

  const x = d3
    .scaleTime()
    .domain(d3.extent(currentData, (d) => d.date))
    .range([0, width]);

  // Draw availability bars for each target
  targets.forEach((target, i) => {
    const targetData = currentData
      .filter((d) => d.target === target)
      .sort((a, b) => a.date - b.date);

    const y = i * targetHeight + 10;

    // Background
    g.append("rect")
      .attr("x", 0)
      .attr("y", y)
      .attr("width", width)
      .attr("height", targetHeight - 20)
      .attr("fill", "#f3f3f3");

    // Availability blocks
    g.selectAll(`.availability-${i}`)
      .data(targetData)
      .enter()
      .append("rect")
      .attr("class", `availability-${i}`)
      .attr("x", (d) => x(d.date))
      .attr("y", y)
      .attr("width", 2)
      .attr("height", targetHeight - 20)
      .attr("fill", (d) => (d.success ? "#22c55e" : "#ef4444"))
      .on("mouseover", function (event, d) {
        tooltip
          .style("opacity", 1)
          .html(
            `${target}<br/>${
              d.success ? "Online" : "Offline"
            }<br/>${d3.timeFormat("%H:%M:%S")(d.date)}`
          );
      })
      .on("mousemove", function (event) {
        tooltip
          .style("left", event.pageX + 10 + "px")
          .style("top", event.pageY - 28 + "px");
      })
      .on("mouseout", function () {
        tooltip.style("opacity", 0);
      });

    // Target label
    g.append("text")
      .attr("x", -margin.left + 5)
      .attr("y", y + targetHeight / 2 - 10)
      .style("font-size", "12px")
      .style("dominant-baseline", "middle")
      .text(target);
  });

  // X-axis
  g.append("g")
    .attr("transform", `translate(0,${height})`)
    .attr("class", "axis")
    .call(d3.axisBottom(x).tickFormat(d3.timeFormat("%H:%M")));

  // Legend
  const legend = svg
    .append("g")
    .attr("transform", `translate(${width + margin.left + 10}, ${margin.top})`);

  const legendData = [
    { label: "Online", color: "#22c55e" },
    { label: "Offline", color: "#ef4444" },
  ];

  legendData.forEach((item, i) => {
    const legendRow = legend
      .append("g")
      .attr("transform", `translate(0, ${i * 20})`);

    legendRow
      .append("rect")
      .attr("width", 10)
      .attr("height", 10)
      .attr("fill", item.color);

    legendRow
      .append("text")
      .attr("x", 15)
      .attr("y", 10)
      .style("font-size", "12px")
      .text(item.label);
  });
}
