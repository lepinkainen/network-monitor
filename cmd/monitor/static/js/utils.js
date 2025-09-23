// Utility functions for the network monitor dashboard

function parseDuration(duration) {
  // Parse Go duration string
  const match = duration.match(/(\d+h)?(\d+m)?(\d+\.?\d*s)?/);
  if (!match) return duration;

  let result = "";
  if (match[1]) result += match[1] + " ";
  if (match[2]) result += match[2] + " ";
  if (match[3]) result += parseFloat(match[3]).toFixed(0) + "s";
  return result.trim();
}

// Enhanced data aggregation function
function aggregateData(data, hours) {
  // Determine aggregation interval based on time range
  let intervalMinutes;
  if (hours <= 1) {
    intervalMinutes = 0.5; // 30-second intervals for very short ranges
  } else if (hours <= 6) {
    intervalMinutes = 2; // 2-minute intervals for short ranges
  } else if (hours <= 24) {
    intervalMinutes = 5; // 5-minute intervals for 24 hours
  } else if (hours <= 72) {
    intervalMinutes = 15; // 15-minute intervals for 3 days
  } else {
    intervalMinutes = 60; // 1-hour intervals for longer ranges
  }

  const intervalMs = intervalMinutes * 60 * 1000;
  const aggregated = {};

  // Include both successful and failed pings for better analysis
  data.forEach((d) => {
    const bucketKey = Math.floor(d.date.getTime() / intervalMs) * intervalMs;
    const targetKey = d.target;

    if (!aggregated[targetKey]) {
      aggregated[targetKey] = {};
    }

    if (!aggregated[targetKey][bucketKey]) {
      aggregated[targetKey][bucketKey] = {
        date: new Date(bucketKey),
        values: [],
        failures: 0,
        total: 0,
      };
    }

    aggregated[targetKey][bucketKey].total++;

    if (d.success && d.rtt_ms > 0) {
      aggregated[targetKey][bucketKey].values.push(d.rtt_ms);
    } else {
      aggregated[targetKey][bucketKey].failures++;
    }
  });

  // Convert to final format with statistics
  const result = {};
  Object.keys(aggregated).forEach((target) => {
    result[target] = Object.values(aggregated[target])
      .map((bucket) => {
        const values = bucket.values;
        const successRate = values.length / bucket.total;

        if (values.length === 0) {
          return {
            date: bucket.date,
            avg: null,
            min: null,
            max: null,
            p95: null,
            count: bucket.total,
            successRate: successRate,
            hasData: false,
          };
        }

        const sorted = values.sort((a, b) => a - b);
        const p95Index = Math.floor(sorted.length * 0.95);

        return {
          date: bucket.date,
          avg: d3.mean(values),
          min: d3.min(values),
          max: d3.max(values),
          p95: sorted[p95Index] || d3.max(values),
          median: d3.median(values),
          count: bucket.total,
          successCount: values.length,
          successRate: successRate,
          hasData: true,
        };
      })
      .sort((a, b) => a.date - b.date);
  });

  return result;
}
