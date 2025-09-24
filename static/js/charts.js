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

  if (!currentData || currentData.length === 0) {
    container
      .append("text")
      .attr("x", 20)
      .attr("y", 30)
      .text("No ping data available for the selected time range");
    return;
  }

  const margin = { top: 40, right: 180, bottom: 60, left: 100 };
  const containerWidth = container.node().getBoundingClientRect().width;
  const availableWidth = containerWidth - margin.left - margin.right;

  // Get targets and sort data chronologically
  const targets = [...new Set(currentData.map((d) => d.target))];
  const sortedData = currentData.sort((a, b) => a.date - b.date);

  // Aggregate data by minute for each target
  const minuteAggregatedData = {};

  targets.forEach(target => {
    const targetPings = sortedData.filter(d => d.target === target);
    const minuteBuckets = {};

    targetPings.forEach(ping => {
      // Create minute bucket key (truncate seconds)
      const minuteKey = new Date(ping.date.getFullYear(), ping.date.getMonth(), ping.date.getDate(),
                                ping.date.getHours(), ping.date.getMinutes()).getTime();

      if (!minuteBuckets[minuteKey]) {
        minuteBuckets[minuteKey] = {
          date: new Date(minuteKey),
          target: target,
          pings: [],
          hasFailure: false,
          successfulLatencies: []
        };
      }

      minuteBuckets[minuteKey].pings.push(ping);

      if (!ping.success) {
        minuteBuckets[minuteKey].hasFailure = true;
      } else if (ping.rtt_ms > 0) {
        minuteBuckets[minuteKey].successfulLatencies.push(ping.rtt_ms);
      }
    });

    // Convert to array and calculate statistics for each minute
    minuteAggregatedData[target] = Object.values(minuteBuckets).map(bucket => {
      const avgLatency = bucket.successfulLatencies.length > 0
        ? d3.mean(bucket.successfulLatencies)
        : null;

      return {
        date: bucket.date,
        target: bucket.target,
        hasFailure: bucket.hasFailure,
        avgLatency: avgLatency,
        totalPings: bucket.pings.length,
        successfulPings: bucket.successfulLatencies.length,
        failedPings: bucket.pings.filter(p => !p.success).length,
        successfulLatencies: bucket.successfulLatencies
      };
    }).sort((a, b) => a.date - b.date);
  });

  // Calculate grid dimensions based on minute data
  const allMinuteData = Object.values(minuteAggregatedData).flat();
  if (allMinuteData.length === 0) {
    container
      .append("text")
      .attr("x", 20)
      .attr("y", 30)
      .text("No minute-level data available for the selected time range");
    return;
  }

  // Determine time range
  const timeExtent = d3.extent(allMinuteData, d => d.date);
  const totalMinutes = Math.ceil((timeExtent[1] - timeExtent[0]) / (60 * 1000));

  // Space-first approach: Calculate optimal grid to fill available width
  const idealCellSize = 12; // Target cell size for good readability
  const minCellSize = 8;    // Minimum viable cell size
  const maxCellSize = 20;   // Maximum before becoming too large

  // Calculate how many columns we can fit with ideal cell size
  const optimalColumns = Math.floor(availableWidth / idealCellSize);

  // Determine minutes per cell to fit the data into available columns
  const minutesPerCell = Math.max(1, Math.ceil(totalMinutes / optimalColumns));
  const actualColumns = Math.ceil(totalMinutes / minutesPerCell);

  // Calculate final cell size to fill the available space
  const cellSize = Math.max(minCellSize, Math.min(maxCellSize, Math.floor(availableWidth / actualColumns)));

  // Calculate rows per target to accommodate the data
  const totalCells = actualColumns;
  const rowsPerTarget = Math.max(1, Math.ceil(totalMinutes / (actualColumns * targets.length))) || 1;
  const totalRows = targets.length * rowsPerTarget;

  const gridWidth = actualColumns * cellSize;
  const gridHeight = totalRows * cellSize;

  const svg = container
    .attr("width", gridWidth + margin.left + margin.right)
    .attr("height", gridHeight + margin.top + margin.bottom);

  const g = svg
    .append("g")
    .attr("transform", `translate(${margin.left},${margin.top})`);

  // Get latency range for color scaling from minute averages
  const successfulMinutes = allMinuteData.filter(d => !d.hasFailure && d.avgLatency);
  let latencyDomain = [0, 100];
  if (successfulMinutes.length > 0) {
    const latencies = successfulMinutes.map(d => d.avgLatency);
    const minLatency = Math.min(...latencies);
    const maxLatency = Math.max(...latencies);
    latencyDomain = [minLatency, Math.max(maxLatency, minLatency + 10)];
  }

  // Color scale for latency
  const latencyColorScale = d3
    .scaleSequential()
    .domain(latencyDomain)
    .interpolator(d3.interpolateGreens)
    .clamp(true);

  // Create grid layout with multi-minute cell support
  const gridData = [];

  targets.forEach((target, targetIndex) => {
    const targetMinutes = minuteAggregatedData[target] || [];
    const targetStartRow = targetIndex * rowsPerTarget;

    // Group minutes into cells based on minutesPerCell
    const cellData = [];
    for (let i = 0; i < targetMinutes.length; i += minutesPerCell) {
      const minutesInCell = targetMinutes.slice(i, i + minutesPerCell);

      if (minutesInCell.length > 0) {
        // Aggregate data for this cell (may contain multiple minutes)
        const hasAnyFailure = minutesInCell.some(m => m.hasFailure);
        const allSuccessfulLatencies = minutesInCell.flatMap(m => m.successfulLatencies);
        const avgLatency = allSuccessfulLatencies.length > 0 ? d3.mean(allSuccessfulLatencies) : null;
        const totalPings = minutesInCell.reduce((sum, m) => sum + m.totalPings, 0);
        const successfulPings = minutesInCell.reduce((sum, m) => sum + m.successfulPings, 0);
        const failedPings = minutesInCell.reduce((sum, m) => sum + m.failedPings, 0);

        cellData.push({
          startDate: minutesInCell[0].date,
          endDate: minutesInCell[minutesInCell.length - 1].date,
          target: target,
          hasFailure: hasAnyFailure,
          avgLatency: avgLatency,
          totalPings: totalPings,
          successfulPings: successfulPings,
          failedPings: failedPings,
          successfulLatencies: allSuccessfulLatencies,
          minutesInCell: minutesInCell,
          timeSpan: minutesPerCell
        });
      }
    }

    // Fill this target's section chronologically (left-to-right, top-to-bottom)
    cellData.forEach((cellInfo, cellIndex) => {
      const cellsPerTarget = actualColumns * rowsPerTarget;
      if (cellIndex < cellsPerTarget) {
        const col = cellIndex % actualColumns;
        const row = Math.floor(cellIndex / actualColumns);
        const globalRow = targetStartRow + row;

        gridData.push({
          target,
          targetIndex,
          data: cellInfo,
          x: col,
          y: globalRow,
          globalRow,
          localRow: row,
          cellIndex
        });
      }
    });
  });

  // Draw grid cells - now representing minutes
  g.selectAll(".minute-cell")
    .data(gridData)
    .enter()
    .append("rect")
    .attr("class", "minute-cell")
    .attr("x", d => d.x * cellSize)
    .attr("y", d => d.y * cellSize)
    .attr("width", cellSize - 0.5)
    .attr("height", cellSize - 0.5)
    .attr("fill", d => {
      const minuteData = d.data;
      if (minuteData.hasFailure) {
        return "#ef4444"; // Red for any failure in that minute
      } else if (minuteData.avgLatency !== null) {
        return latencyColorScale(minuteData.avgLatency);
      } else {
        return "#f3f4f6"; // Light gray for no data
      }
    })
    .attr("stroke", "#fff")
    .attr("stroke-width", 0.3)
    .style("cursor", "pointer")
    .on("mouseover", function (event, d) {
      const cellData = d.data;
      // Make hovered cell more prominent
      d3.select(this)
        .attr("stroke", "#333")
        .attr("stroke-width", 1.5);

      const successRate = cellData.totalPings > 0
        ? ((cellData.successfulPings / cellData.totalPings) * 100).toFixed(1)
        : "0.0";

      // Format time range - single minute vs multi-minute cells
      const timeDisplay = cellData.timeSpan === 1 || cellData.startDate.getTime() === cellData.endDate.getTime()
        ? d3.timeFormat("%H:%M")(cellData.startDate)
        : `${d3.timeFormat("%H:%M")(cellData.startDate)}-${d3.timeFormat("%H:%M")(cellData.endDate)}`;

      // Calculate latency range display
      const latencyRangeText = cellData.successfulLatencies.length > 1
        ? `<strong>Latency Range:</strong> ${Math.min(...cellData.successfulLatencies).toFixed(1)} - ${Math.max(...cellData.successfulLatencies).toFixed(1)} ms<br/>`
        : "";

      tooltip
        .style("opacity", 1)
        .html(`
          <strong>${d.target}</strong><br/>
          <strong>Time:</strong> ${timeDisplay}<br/>
          ${cellData.timeSpan > 1 ? `<strong>Duration:</strong> ${cellData.timeSpan} minute${cellData.timeSpan > 1 ? 's' : ''}<br/>` : ""}
          <strong>Status:</strong> ${cellData.hasFailure ? "Has Failures" : "All Success"}<br/>
          ${cellData.avgLatency !== null ? `<strong>Avg Latency:</strong> ${cellData.avgLatency.toFixed(1)} ms<br/>` : ""}
          <strong>Pings:</strong> ${cellData.successfulPings}/${cellData.totalPings} successful (${successRate}%)<br/>
          ${cellData.failedPings > 0 ? `<strong>Failures:</strong> ${cellData.failedPings}<br/>` : ""}
          ${latencyRangeText}
        `);
    })
    .on("mousemove", function (event) {
      tooltip
        .style("left", event.pageX + 10 + "px")
        .style("top", event.pageY - 28 + "px");
    })
    .on("mouseout", function (d) {
      // Reset cell appearance
      d3.select(this)
        .attr("stroke", "#fff")
        .attr("stroke-width", 0.3);

      tooltip.style("opacity", 0);
    });

  // Target labels and separators
  targets.forEach((target, i) => {
    const targetStartY = i * rowsPerTarget * cellSize;
    const targetCenterY = targetStartY + (rowsPerTarget * cellSize) / 2;

    // Target label
    g.append("text")
      .attr("x", -15)
      .attr("y", targetCenterY)
      .attr("text-anchor", "end")
      .attr("dominant-baseline", "middle")
      .style("font-size", "11px")
      .style("font-weight", "600")
      .text(target);

    // Separator line between targets
    if (i > 0) {
      g.append("line")
        .attr("x1", -10)
        .attr("x2", gridWidth)
        .attr("y1", targetStartY - 1)
        .attr("y2", targetStartY - 1)
        .attr("stroke", "#ccc")
        .attr("stroke-width", 1);
    }
  });

  // Time progression indicators (sample time markers)
  const timeMarkers = [];
  const markersCount = Math.min(actualColumns, 10);
  for (let i = 0; i < markersCount; i++) {
    const colIndex = Math.floor((i / (markersCount - 1)) * (actualColumns - 1));
    const sampleCell = gridData.find(d => d.x === colIndex);
    if (sampleCell) {
      timeMarkers.push({
        x: colIndex * cellSize + cellSize / 2,
        time: sampleCell.data.startDate
      });
    }
  }

  g.selectAll(".time-marker")
    .data(timeMarkers)
    .enter()
    .append("text")
    .attr("class", "time-marker")
    .attr("x", d => d.x)
    .attr("y", gridHeight + 20)
    .attr("text-anchor", "middle")
    .style("font-size", "9px")
    .style("fill", "#666")
    .text(d => d3.timeFormat("%H:%M")(d.time));

  // Legend
  const legend = svg
    .append("g")
    .attr("class", "heatmap-legend")
    .attr("transform", `translate(${gridWidth + margin.left + 20}, ${margin.top})`);

  // Failed ping legend
  const failedLegend = legend
    .append("g")
    .attr("transform", "translate(0, 0)");

  failedLegend
    .append("rect")
    .attr("width", 12)
    .attr("height", 12)
    .attr("fill", "#ef4444");

  failedLegend
    .append("text")
    .attr("x", 18)
    .attr("y", 10)
    .style("font-size", "11px")
    .text("Failed");

  // Latency gradient legend
  const gradientLegend = legend
    .append("g")
    .attr("transform", "translate(0, 35)");

  // Create gradient definition
  const defs = svg.append("defs");
  const gradient = defs
    .append("linearGradient")
    .attr("id", "latency-gradient")
    .attr("x1", "0%")
    .attr("x2", "0%")
    .attr("y1", "100%")
    .attr("y2", "0%");

  const gradientStops = d3.range(0, 1.1, 0.1);
  gradientStops.forEach(stop => {
    gradient
      .append("stop")
      .attr("offset", stop * 100 + "%")
      .attr("stop-color", latencyColorScale(latencyDomain[0] + stop * (latencyDomain[1] - latencyDomain[0])));
  });

  const legendHeight = 60;
  gradientLegend
    .append("rect")
    .attr("width", 12)
    .attr("height", legendHeight)
    .style("fill", "url(#latency-gradient)");

  // Latency scale
  const latencyScale = d3.scaleLinear()
    .domain(latencyDomain)
    .range([legendHeight, 0]);

  const latencyAxis = d3.axisRight(latencyScale)
    .ticks(3)
    .tickFormat(d => d.toFixed(0) + "ms");

  gradientLegend
    .append("g")
    .attr("transform", "translate(12, 0)")
    .call(latencyAxis)
    .selectAll("text")
    .style("font-size", "9px");

  gradientLegend
    .append("text")
    .attr("x", 6)
    .attr("y", -8)
    .attr("text-anchor", "middle")
    .style("font-size", "10px")
    .style("font-weight", "500")
    .text("Latency");

  // Info text
  const timeUnitText = minutesPerCell === 1 ? "1 minute" : `${minutesPerCell} minutes`;
  g.append("text")
    .attr("x", 0)
    .attr("y", -20)
    .style("font-size", "10px")
    .style("fill", "#666")
    .text(`${gridData.length} cells in ${actualColumns}×${totalRows} grid • Each square = ${timeUnitText} • Red = any failure, Green = average latency`);

  // Grid structure info
  g.append("text")
    .attr("x", 0)
    .attr("y", -8)
    .style("font-size", "9px")
    .style("fill", "#888")
    .text(`${targets.length} targets × ${rowsPerTarget} rows each • Time flows left to right`);
}
