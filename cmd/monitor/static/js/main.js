// Main application initialization and coordination

// Initialize the application when DOM is loaded
document.addEventListener("DOMContentLoaded", function () {
  // Initial data load
  fetchData().then(() => {
    updateDashboard();
    drawLatencyChart();
    drawAvailabilityChart();
    drawHeatmap();
    updateOutagesTable();
  });

  // Set up event listeners
  setupEventListeners();

  // Auto-refresh every 30 seconds
  setInterval(() => refreshData(), 30000);
});

// Global functions that need to be accessible from HTML
window.refreshData = refreshData;
window.generateReport = generateReport;
