/**
 * Network Latency Heatmap Visualization
 * 
 * This script creates interactive visualizations for the network latency heatmap plugin.
 * It generates both a color-coded heatmap table and an interactive time-series chart.
 */

// Namespace for Network Latency Heatmap
const LatencyHeatmap = {
    // Initialize the visualization with plugin results
    initialize: function(results) {
        // Add CSS if not already added
        if (!document.getElementById('latency-heatmap-css')) {
            const link = document.createElement('link');
            link.id = 'latency-heatmap-css';
            link.rel = 'stylesheet';
            link.href = '/static/css/latency-heatmap.css';
            document.head.appendChild(link);
        }

        // Create container for results
        const container = document.createElement('div');
        container.className = 'latency-results-container';

        // Add statistics table
        container.appendChild(this.createStatsTable(results.statistics));

        // Create heatmap visualization if data is available
        if (results.heatmapData && results.heatmapData.targets.length > 0) {
            container.appendChild(this.createHeatmapTable(results.heatmapData));
            
            // Add interactive chart if enabled
            if (results.showGraph) {
                const chartContainer = document.createElement('div');
                chartContainer.className = 'chart-container';
                chartContainer.id = 'latency-chart';
                container.appendChild(chartContainer);
                
                // Load Chart.js if not already loaded
                if (typeof Chart === 'undefined') {
                    const script = document.createElement('script');
                    script.src = 'https://cdn.jsdelivr.net/npm/chart.js';
                    script.onload = () => this.createChart(results.heatmapData, 'latency-chart');
                    document.head.appendChild(script);
                } else {
                    // Chart.js already loaded, create chart directly
                    this.createChart(results.heatmapData, 'latency-chart');
                }
            }
        }

        return container;
    },

    // Create a table displaying latency statistics for each target
    createStatsTable: function(statistics) {
        const table = document.createElement('table');
        table.className = 'latency-stats-table';
        
        // Create header row
        const thead = document.createElement('thead');
        const headerRow = document.createElement('tr');
        
        const headers = ['Target', 'Min (ms)', 'Avg (ms)', 'Max (ms)', 'Median (ms)', 'Jitter (ms)', 'Packet Loss (%)'];
        
        headers.forEach(header => {
            const th = document.createElement('th');
            th.textContent = header;
            headerRow.appendChild(th);
        });
        
        thead.appendChild(headerRow);
        table.appendChild(thead);
        
        // Create body with statistics for each target
        const tbody = document.createElement('tbody');
        
        statistics.forEach(stat => {
            const row = document.createElement('tr');
            
            // Add data cells
            [
                stat.target,
                stat.minRtt.toFixed(2),
                stat.avgRtt.toFixed(2),
                stat.maxRtt.toFixed(2),
                stat.medianRtt.toFixed(2),
                stat.jitter.toFixed(2),
                stat.packetLoss.toFixed(2)
            ].forEach(value => {
                const td = document.createElement('td');
                td.textContent = value;
                row.appendChild(td);
            });
            
            tbody.appendChild(row);
        });
        
        table.appendChild(tbody);
        return table;
    },

    // Create the heatmap visualization as a table
    createHeatmapTable: function(heatmapData) {
        const container = document.createElement('div');
        container.className = 'latency-heatmap-container';
        
        const table = document.createElement('table');
        table.className = 'latency-heatmap';
        
        // Create header with timestamps
        const thead = document.createElement('thead');
        const headerRow = document.createElement('tr');
        
        // Add empty cell for the corner
        const cornerCell = document.createElement('th');
        headerRow.appendChild(cornerCell);
        
        // Add timestamp headers (use shorter format for display)
        heatmapData.timestamps.forEach(timestamp => {
            const th = document.createElement('th');
            // Convert ISO timestamp to just time part (HH:MM:SS)
            const date = new Date(timestamp);
            th.textContent = date.toLocaleTimeString();
            headerRow.appendChild(th);
        });
        
        thead.appendChild(headerRow);
        table.appendChild(thead);
        
        // Create table body with latency data
        const tbody = document.createElement('tbody');
        
        // For each target, create a row with latency cells
        heatmapData.targets.forEach((target, targetIndex) => {
            const row = document.createElement('tr');
            
            // Add target name as first cell
            const targetCell = document.createElement('td');
            targetCell.className = 'target-label';
            targetCell.textContent = target;
            row.appendChild(targetCell);
            
            // Add latency cells
            const latencyValues = heatmapData.latencyData[targetIndex];
            latencyValues.forEach(latency => {
                const cell = document.createElement('td');
                
                if (latency < 0) {
                    // Failed ping
                    cell.className = 'ping-failed';
                    cell.textContent = 'X';
                } else {
                    // Successful ping, apply color based on latency value
                    const colorClass = this.getLatencyColorClass(
                        latency, 
                        heatmapData.minLatency, 
                        heatmapData.maxLatency
                    );
                    cell.className = colorClass;
                    cell.textContent = Math.round(latency);
                    
                    // Add tooltip with exact value
                    cell.title = `${latency.toFixed(2)} ms`;
                }
                
                row.appendChild(cell);
            });
            
            tbody.appendChild(row);
        });
        
        table.appendChild(tbody);
        container.appendChild(table);
        
        return container;
    },

    // Determine color class based on latency value
    getLatencyColorClass: function(latency, minLatency, maxLatency) {
        if (latency < 0) return 'ping-failed';
        
        // Calculate relative position in the latency range
        const range = maxLatency - minLatency;
        const normalized = range > 0 ? (latency - minLatency) / range : 0;
        
        // Assign color class based on normalized value
        if (normalized < 0.2) return 'latency-low';
        if (normalized < 0.4) return 'latency-medium-low';
        if (normalized < 0.6) return 'latency-medium';
        if (normalized < 0.8) return 'latency-medium-high';
        if (normalized < 0.95) return 'latency-high';
        return 'latency-very-high';
    },

    // Create an interactive chart using Chart.js
    createChart: function(heatmapData, containerId) {
        const ctx = document.getElementById(containerId);
        if (!ctx) return;
        
        // Prepare datasets for each target
        const datasets = heatmapData.targets.map((target, index) => {
            // Get latency values for this target
            const data = heatmapData.latencyData[index].map(value => 
                value < 0 ? null : value  // Convert failed pings to null for gaps in chart
            );
            
            // Generate a color based on index
            const hue = (index * 137) % 360;  // Golden angle to distribute colors
            const color = `hsl(${hue}, 70%, 50%)`;
            
            return {
                label: target,
                data: data,
                borderColor: color,
                backgroundColor: `${color}33`,  // Add transparency
                borderWidth: 2,
                pointRadius: 3,
                tension: 0.3,  // Slight curve for better visibility
                fill: false
            };
        });
        
        // Convert ISO timestamps to more readable format
        const labels = heatmapData.timestamps.map(timestamp => {
            const date = new Date(timestamp);
            return date.toLocaleTimeString();
        });
        
        // Create the chart
        new Chart(ctx, {
            type: 'line',
            data: {
                labels: labels,
                datasets: datasets
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    title: {
                        display: true,
                        text: 'Network Latency Over Time',
                        font: {
                            size: 16
                        }
                    },
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                const value = context.raw;
                                if (value === null) return `${context.dataset.label}: Failed ping`;
                                return `${context.dataset.label}: ${value.toFixed(2)} ms`;
                            }
                        }
                    }
                },
                scales: {
                    x: {
                        title: {
                            display: true,
                            text: 'Time'
                        }
                    },
                    y: {
                        title: {
                            display: true,
                            text: 'Latency (ms)'
                        },
                        beginAtZero: true
                    }
                }
            }
        });
    }
};

// Register the plugin with the plugin manager if it exists
if (typeof PluginManager !== 'undefined') {
    PluginManager.registerCustomDisplay('network_latency_heatmap', function(results, element) {
        element.innerHTML = '';
        element.appendChild(LatencyHeatmap.initialize(results));
    });
}
