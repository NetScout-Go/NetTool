// Dashboard JavaScript

document.addEventListener('DOMContentLoaded', function() {    // Apply theme based on user preference
    applyDashboardTheme();
    
    // Initialize the network traffic chart
    initTrafficChart();
    
    // Set up event listeners
    document.getElementById('runSpeedTestBtn')?.addEventListener('click', runSpeedTest);
    
    // Listen for theme changes from main.js
    document.addEventListener('themechange', function(e) {
        // Apply dashboard theme changes
        applyDashboardTheme();
    });
    
    // Simulate some live updates
    startLiveUpdates();
});

// Apply theme to dashboard-specific elements
function applyDashboardTheme() {
    const isDarkMode = document.body.classList.contains('dark-mode') || 
                       localStorage.getItem('theme') === 'dark';
    
    // Apply dashboard-specific dark mode class
    if (isDarkMode) {
        document.body.classList.add('dashboard-dark-mode');
    } else {
        document.body.classList.remove('dashboard-dark-mode');
    }
    
    // Update chart theme if exists
    if (window.trafficChart) {
        const chartTheme = isDarkMode ? 
            {
                color: '#eceff1',
                gridColor: 'rgba(255, 255, 255, 0.1)',
                backgroundColor: '#1e1e1e',
                borderColor: {
                    download: '#42a5f5',  // Adjusted for dark mode
                    upload: '#4db6ac'     // Adjusted for dark mode
                },
                backgroundColor: {
                    download: 'rgba(66, 165, 245, 0.1)',
                    upload: 'rgba(77, 182, 172, 0.1)'
                }
            } : 
            {
                color: '#37474f',
                gridColor: 'rgba(0, 0, 0, 0.1)',
                backgroundColor: '#ffffff',
                borderColor: {
                    download: '#1e88e5',
                    upload: '#26a69a'
                },
                backgroundColor: {
                    download: 'rgba(30, 136, 229, 0.1)',
                    upload: 'rgba(38, 166, 154, 0.1)'
                }
            };
        
        // Update chart colors
        window.trafficChart.options.scales.y.grid.color = chartTheme.gridColor;
        window.trafficChart.options.scales.x.grid.color = chartTheme.gridColor;
        window.trafficChart.options.scales.y.ticks.color = chartTheme.color;
        window.trafficChart.options.scales.x.ticks.color = chartTheme.color;
        window.trafficChart.options.plugins.legend.labels.color = chartTheme.color;
        window.trafficChart.options.plugins.title.color = chartTheme.color;
        
        // Update dataset colors
        window.trafficChart.data.datasets[0].borderColor = chartTheme.borderColor.download;
        window.trafficChart.data.datasets[0].backgroundColor = chartTheme.backgroundColor.download;
        window.trafficChart.data.datasets[1].borderColor = chartTheme.borderColor.upload;
        window.trafficChart.data.datasets[1].backgroundColor = chartTheme.backgroundColor.upload;
        
        window.trafficChart.update();
    }
    
    // Update card styles
    const cards = document.querySelectorAll('.card');
    cards.forEach(card => {
        if (isDarkMode) {
            card.classList.add('dark-card');
        } else {
            card.classList.remove('dark-card');
        }
    });
    
    // Update status indicators for better dark mode visibility
    const statusIndicators = document.querySelectorAll('.status-indicator');
    statusIndicators.forEach(indicator => {
        if (isDarkMode) {
            indicator.style.backgroundColor = 'rgba(66, 165, 245, 0.15)';
        } else {
            indicator.style.backgroundColor = 'rgba(30, 136, 229, 0.1)';
        }
    });
}

// Initialize the network traffic chart
function initTrafficChart() {
    const ctx = document.getElementById('trafficChart').getContext('2d');
    
    // Generate time labels (last 10 minutes in 1-minute intervals)
    const timeLabels = [];
    const now = new Date();
    for (let i = 9; i >= 0; i--) {
        const time = new Date(now.getTime() - i * 60000);
        timeLabels.push(time.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }));
    }
    
    // Generate sample data
    const downloadData = [2.1, 2.5, 3.2, 5.1, 4.3, 3.8, 3.5, 4.2, 3.9, 4.5];
    const uploadData = [0.8, 0.9, 1.2, 1.5, 1.3, 1.1, 0.9, 1.2, 1.4, 1.1];
    
    // Determine if we're in dark mode
    const isDarkMode = document.body.classList.contains('dark-mode') || 
                        document.body.classList.contains('dashboard-dark-mode') ||
                        localStorage.getItem('theme') === 'dark';
    
    // Set theme-specific colors
    const chartTheme = isDarkMode ? 
        {
            color: '#eceff1',
            gridColor: 'rgba(255, 255, 255, 0.1)',
            backgroundColor: '#1e1e1e',
            borderColor: {
                download: '#42a5f5',  // Adjusted for dark mode
                upload: '#4db6ac'     // Adjusted for dark mode
            },
            backgroundColor: {
                download: 'rgba(66, 165, 245, 0.1)',
                upload: 'rgba(77, 182, 172, 0.1)'
            }
        } : 
        {
            color: '#37474f',
            gridColor: 'rgba(0, 0, 0, 0.1)',
            backgroundColor: '#ffffff',
            borderColor: {
                download: '#1e88e5',
                upload: '#26a69a'
            },
            backgroundColor: {
                download: 'rgba(30, 136, 229, 0.1)',
                upload: 'rgba(38, 166, 154, 0.1)'
            }
        };
    
    // Create the chart
    const trafficChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: timeLabels,
            datasets: [
                {
                    label: 'Download (Mbps)',
                    data: downloadData,
                    borderColor: chartTheme.borderColor.download,
                    backgroundColor: chartTheme.backgroundColor.download,
                    borderWidth: 2,
                    fill: true,
                    tension: 0.4
                },
                {
                    label: 'Upload (Mbps)',
                    data: uploadData,
                    borderColor: chartTheme.borderColor.upload,
                    backgroundColor: chartTheme.backgroundColor.upload,
                    borderWidth: 2,
                    fill: true,
                    tension: 0.4
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    position: 'top',
                    labels: {
                        color: chartTheme.color
                    }
                },
                tooltip: {
                    mode: 'index',
                    intersect: false,
                    backgroundColor: chartTheme.backgroundColor,
                    titleColor: chartTheme.color,
                    bodyColor: chartTheme.color
                }
            },
            scales: {
                y: {
                    beginAtZero: true,
                    title: {
                        display: true,
                        text: 'Mbps',
                        color: chartTheme.color
                    },
                    grid: {
                        color: chartTheme.gridColor
                    },
                    ticks: {
                        color: chartTheme.color
                    }
                },
                x: {
                    title: {
                        display: true,
                        text: 'Time',
                        color: chartTheme.color
                    },
                    grid: {
                        color: chartTheme.gridColor
                    },
                    ticks: {
                        color: chartTheme.color
                    }
                }
            }
        }
    });
    
    // Store chart in window object for later updates
    window.trafficChart = trafficChart;
}

// Simulate a speed test
function runSpeedTest() {
    const button = document.getElementById('runSpeedTestBtn');
    const bandwidth = document.getElementById('bandwidth');
    
    // Disable button and show running state
    button.disabled = true;
    button.innerHTML = '<i class="bi bi-arrow-repeat fa-spin"></i> Running Test...';
    
    // Simulate the test running for 5 seconds
    setTimeout(() => {
        // Update with "results"
        bandwidth.textContent = '94.7 Mbps';
        
        // Re-enable button
        button.disabled = false;
        button.innerHTML = '<i class="bi bi-speedometer2"></i> Run Speed Test';
        
        // Show a toast notification
        showToast('Speed test completed', 'Download: 94.7 Mbps, Upload: 14.2 Mbps');
    }, 5000);
}

// Show a toast notification (would require Bootstrap toast or custom implementation)
function showToast(title, message) {
    // Remove any existing toasts
    const existingToasts = document.querySelectorAll('.toast-notification');
    existingToasts.forEach(toast => toast.remove());
    
    // Create a new toast
    const toast = document.createElement('div');
    toast.className = 'toast-notification';
    toast.innerHTML = `
        <div class="toast-header">
            <strong>${title}</strong>
        </div>
        <div class="toast-body">
            ${message}
        </div>
    `;
    
    // Add to DOM
    document.body.appendChild(toast);
    
    // Position in the bottom right corner
    toast.style.bottom = '20px';
    toast.style.right = '20px';
    
    // Trigger animation
    setTimeout(() => toast.classList.add('show'), 10);
    
    // Auto-hide after delay
    setTimeout(() => {
        toast.classList.remove('show');
        setTimeout(() => toast.remove(), 300);
    }, 5000);
    
    // Add styles for the toast
    toast.style.position = 'fixed';
    toast.style.bottom = '20px';
    toast.style.right = '20px';
    toast.style.backgroundColor = 'white';
    toast.style.color = '#333';
    toast.style.padding = '10px 15px';
    toast.style.borderRadius = '4px';
    toast.style.boxShadow = '0 2px 10px rgba(0,0,0,0.2)';
    toast.style.zIndex = '9999';
    
    // Add to document
    document.body.appendChild(toast);
    
    // Remove after 3 seconds
    setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transition = 'opacity 0.5s ease';
        setTimeout(() => document.body.removeChild(toast), 500);
    }, 3000);
}

// Simulate live updates to various metrics
function startLiveUpdates() {
    // Update uptime every second
    let seconds = 22 * 60 + 37; // 14:22:37
    setInterval(() => {
        seconds++;
        const hours = Math.floor(seconds / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        const secs = seconds % 60;
        
        const timeString = `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
        document.getElementById('uptime').textContent = timeString;
    }, 1000);
    
    // Update latency metrics occasionally
    setInterval(() => {
        // Generate a small random variation for latency
        const currentLatency = parseFloat(document.getElementById('latency').textContent);
        const latencyVariation = (Math.random() * 6) - 3; // -3 to +3 ms
        const newLatency = Math.max(10, Math.round(currentLatency + latencyVariation));
        document.getElementById('latency').textContent = `${newLatency} ms`;
        
        // Occasionally update service latencies
        if (Math.random() < 0.3) { // 30% chance to update each interval
            const services = ['google', 'amazon', 'cloudflare', 'microsoft'];
            const service = services[Math.floor(Math.random() * services.length)];
            const currentValue = parseFloat(document.getElementById(`${service}Latency`).textContent);
            const variation = (Math.random() * 4) - 2; // -2 to +2 ms
            const newValue = Math.max(5, Math.round((currentValue + variation) * 10) / 10);
            document.getElementById(`${service}Latency`).textContent = `${newValue.toFixed(1)} ms`;
        }
    }, 5000);
    
    // Update traffic chart every 60 seconds
    setInterval(() => {
        if (!window.trafficChart) return;
        
        const chart = window.trafficChart;
        
        // Move the time window forward
        const labels = chart.data.labels;
        labels.shift();
        const now = new Date();
        labels.push(now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }));
        
        // Update the data
        for (let dataset of chart.data.datasets) {
            dataset.data.shift();
            if (dataset.label.includes('Download')) {
                // Download data has higher values
                dataset.data.push(Math.round((3 + Math.random() * 2) * 10) / 10);
            } else {
                // Upload data has lower values
                dataset.data.push(Math.round((0.8 + Math.random() * 0.8) * 10) / 10);
            }
        }
        
        // Update the last updated time
        document.getElementById('lastUpdated').textContent = new Date().toLocaleString();
        
        // Update the chart
        chart.update();
    }, 60000);
    
    // Initial set of the last updated time
    document.getElementById('lastUpdated').textContent = new Date().toLocaleString();
}
