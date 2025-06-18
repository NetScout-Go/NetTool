/**
 * NetScout-Pi Plugin Manager
 * Handles plugin navigation, execution and results display
 */

// Plugin manager object to handle all plugin interactions
const PluginManager = {
    activePluginId: null,
    pluginHistory: {},
    customDisplayFunctions: {}, // Store custom display functions
    
    // Initialize plugin manager
    init: function() {
        console.log('Plugin Manager initialized');
        
        // Store current plugin ID if on a plugin page
        const pluginLinks = document.querySelectorAll('.plugin-nav-link');
        pluginLinks.forEach(link => {
            const pluginId = link.getAttribute('data-plugin-id');
            if (link.classList.contains('active')) {
                this.activePluginId = pluginId;
            }
            
            // Add click event for analytics
            link.addEventListener('click', () => {
                console.log(`Navigating to plugin: ${pluginId}`);
            });
        });
        
        // Set up plugin execution handlers
        this.setupPluginForm();
        
        // Set up result caching
        this.setupResultCaching();
    },
    
    // Set up the plugin form submission handling
    setupPluginForm: function() {
        const pluginForm = document.getElementById('pluginForm');
        if (pluginForm) {
            pluginForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.runPlugin();
            });
            
            // Handle refresh results button
            const refreshBtn = document.getElementById('refreshResultsBtn');
            if (refreshBtn) {
                refreshBtn.addEventListener('click', () => {
                    this.refreshResults();
                });
            }
            
            // Handle export results button
            const exportBtn = document.getElementById('exportResultsBtn');
            if (exportBtn) {
                exportBtn.addEventListener('click', () => {
                    this.exportResults();
                });
            }
        }
    },
    
    // Set up result caching to persist results between page refreshes
    setupResultCaching: function() {
        // Check if we have cached results for this plugin
        if (this.activePluginId) {
            const cachedResults = localStorage.getItem(`plugin_results_${this.activePluginId}`);
            if (cachedResults) {
                try {
                    const results = JSON.parse(cachedResults);
                    const timestamp = localStorage.getItem(`plugin_timestamp_${this.activePluginId}`);
                    
                    // Display cached results
                    this.displayResults(results, timestamp);
                    
                    console.log('Loaded cached results for plugin:', this.activePluginId);
                } catch (e) {
                    console.error('Error loading cached results:', e);
                    localStorage.removeItem(`plugin_results_${this.activePluginId}`);
                }
            }
        }
    },
    
    // Run the current plugin with form parameters
    runPlugin: function() {
        if (!this.activePluginId) return;
        
        // Show loading indicator with message
        document.getElementById('pluginResults').classList.add('d-none');
        document.getElementById('resultsLoading').classList.remove('d-none');
        document.getElementById('loadingMessage').textContent = 'Running plugin...';
        
        // Get form parameters
        const form = document.getElementById('pluginForm');
        const formData = new FormData(form);
        const params = {};
        
        // Convert form data to JSON object
        formData.forEach((value, key) => {
            // Handle different input types
            if (key.startsWith('param-')) {
                const paramName = key.substring(6); // Remove 'param-' prefix
                
                // Convert numeric values
                if (!isNaN(value) && value !== '') {
                    params[paramName] = parseFloat(value);
                } else if (value === 'true' || value === 'false') {
                    params[paramName] = value === 'true';
                } else {
                    params[paramName] = value;
                }
            }
        });
        
        // Save parameters for refresh
        this.lastParams = params;
        
        // For external plugins, give a specific message
        if (this.activePluginId === 'external_plugin') {
            document.getElementById('loadingMessage').textContent = 'Running external plugin... This may take a moment.';
        }
        
        // Set a timeout to update the loading message if it takes longer than expected
        const loadingTimeout = setTimeout(() => {
            document.getElementById('loadingMessage').textContent = 'Plugin is still running... Please wait.';
        }, 5000);
        
        // Call API to run plugin
        fetch(`/api/plugins/${this.activePluginId}/run`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(params)
        })
        .then(response => {
            clearTimeout(loadingTimeout);
            if (!response.ok) {
                if (response.status === 500) {
                    // Instead of throwing an error, update the loading message
                    document.getElementById('loadingMessage').textContent = 'Processing results...';
                    // Try again after a short delay
                    setTimeout(() => this.checkPluginStatus(params), 1500);
                    return null;
                }
                throw new Error(`Server responded with status: ${response.status}`);
            }
            return response.json();
        })
        .then(data => {
            if (data === null) return; // Handled by retry mechanism
            
            // Hide loading indicator
            document.getElementById('resultsLoading').classList.add('d-none');
            document.getElementById('pluginResults').classList.remove('d-none');
            
            // Save result for export and caching
            this.lastResult = data;
            
            // Cache results for this plugin
            try {
                localStorage.setItem(`plugin_results_${this.activePluginId}`, JSON.stringify(data));
                const timestamp = new Date().toLocaleString();
                localStorage.setItem(`plugin_timestamp_${this.activePluginId}`, timestamp);
                
                // Display results
                this.displayResults(data, timestamp);
            } catch (e) {
                console.error('Error caching results:', e);
            }
        })
        .catch(error => {
            console.error('Error running plugin:', error);
            document.getElementById('resultsLoading').classList.add('d-none');
            document.getElementById('pluginResults').classList.remove('d-none');
            document.getElementById('pluginResults').innerHTML = `
                <div class="alert alert-danger">
                    <i class="bi bi-exclamation-triangle-fill"></i>
                    Error running plugin: ${error.message}
                </div>
            `;
        });
    },
    
    // Check plugin status and retry if necessary
    checkPluginStatus: function(params) {
        console.log('Checking plugin status and retrying...');
        
        fetch(`/api/plugins/${this.activePluginId}/run`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(params)
        })
        .then(response => {
            if (!response.ok) {
                if (response.status === 500) {
                    // If it's still processing, try again after a delay
                    document.getElementById('loadingMessage').textContent = 'Plugin is still running... Please be patient.';
                    setTimeout(() => this.checkPluginStatus(params), 2000);
                    return null;
                }
                throw new Error(`Server responded with status: ${response.status}`);
            }
            return response.json();
        })
        .then(data => {
            if (data === null) return; // Handled by retry mechanism
            
            // Hide loading indicator
            document.getElementById('resultsLoading').classList.add('d-none');
            document.getElementById('pluginResults').classList.remove('d-none');
            
            // Save result for export and caching
            this.lastResult = data;
            
            // Cache results for this plugin
            try {
                localStorage.setItem(`plugin_results_${this.activePluginId}`, JSON.stringify(data));
                const timestamp = new Date().toLocaleString();
                localStorage.setItem(`plugin_timestamp_${this.activePluginId}`, timestamp);
                
                // Display results
                this.displayResults(data, timestamp);
            } catch (e) {
                console.error('Error caching results:', e);
            }
        })
        .catch(error => {
            console.error('Error checking plugin status:', error);
            // After several retries, give up and show the error
            document.getElementById('resultsLoading').classList.add('d-none');
            document.getElementById('pluginResults').classList.remove('d-none');
            document.getElementById('pluginResults').innerHTML = `
                <div class="alert alert-danger">
                    <i class="bi bi-exclamation-triangle-fill"></i>
                    Error running plugin: ${error.message}
                </div>
            `;
        });
    },
    
    // Refresh the current plugin results
    refreshResults: function() {
        if (this.lastParams) {
            this.runPlugin();
        } else {
            // Run with default parameters
            this.runPlugin();
        }
    },
    
    // Export results to a file
    exportResults: function() {
        if (!this.lastResult) {
            alert('No results to export');
            return;
        }

        // Create a JSON file for download
        const dataStr = "data:text/json;charset=utf-8," + encodeURIComponent(JSON.stringify(this.lastResult, null, 2));
        const downloadAnchorNode = document.createElement('a');
        downloadAnchorNode.setAttribute("href", dataStr);
        downloadAnchorNode.setAttribute("download", `netscout_${this.activePluginId}_${new Date().toISOString()}.json`);
        document.body.appendChild(downloadAnchorNode);
        downloadAnchorNode.click();
        downloadAnchorNode.remove();
    },
    
    // Display plugin results
    displayResults: function(data, timestamp) {
        const resultsElement = document.getElementById('pluginResults');
        if (!resultsElement) return;
        
        // Update timestamp
        document.getElementById('resultTimestamp').textContent = `Last run: ${timestamp}`;
        
        // Check if there's a custom display function for this plugin
        if (this.customDisplayFunctions[this.activePluginId]) {
            resultsElement.innerHTML = '';
            this.customDisplayFunctions[this.activePluginId](data, resultsElement);
            return;
        }
        
        // Different display formats based on plugin ID
        switch (this.activePluginId) {
            case 'ping':
                this.displayPingResults(data, resultsElement);
                break;
            case 'traceroute':
                this.displayTracerouteResults(data, resultsElement);
                break;
            case 'port_scanner':
                this.displayPortScannerResults(data, resultsElement);
                break;
            case 'dns_lookup':
                this.displayDNSLookupResults(data, resultsElement);
                break;
            case 'bandwidth_test':
                this.displayBandwidthResults(data, resultsElement);
                break;
            case 'external_plugin':
                this.displayExternalPluginResults(data, resultsElement);
                break;
            case 'wifi_device_locator':
                this.displayWifiDeviceLocatorResults(data, resultsElement);
                break;
            default:
                // Generic JSON display
                resultsElement.innerHTML = `<pre class="json-result">${JSON.stringify(data, null, 2)}</pre>`;
        }
    },
    
    // Display ping results
    displayPingResults: function(data, element) {
        // Implementation for ping results display
        element.innerHTML = `
            <div class="ping-results">
                <div class="row mb-4">
                    <div class="col-md-6">
                        <div class="result-card">
                            <div class="result-header">Summary</div>
                            <div class="result-body">
                                <div class="result-row">
                                    <div class="result-label">Host</div>
                                    <div class="result-value">${data.host}</div>
                                </div>
                                <div class="result-row">
                                    <div class="result-label">Packets</div>
                                    <div class="result-value">${data.transmitted} sent, ${data.received} received</div>
                                </div>
                                <div class="result-row">
                                    <div class="result-label">Packet Loss</div>
                                    <div class="result-value">${data.packetLoss}%</div>
                                </div>
                                <div class="result-row">
                                    <div class="result-label">Round Trip Time</div>
                                    <div class="result-value">
                                        min: ${data.timeMin.toFixed(3)} ms<br>
                                        avg: ${data.timeAvg.toFixed(3)} ms<br>
                                        max: ${data.timeMax.toFixed(3)} ms<br>
                                        stddev: ${data.timeStdDev.toFixed(3)} ms
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div class="col-md-6">
                        <canvas id="pingChart" height="200"></canvas>
                    </div>
                </div>
                <div class="result-card">
                    <div class="result-header">Raw Output</div>
                    <div class="result-body">
                        <pre class="raw-output">${data.rawOutput || 'No raw output available'}</pre>
                    </div>
                </div>
            </div>
        `;
        
        // Create ping time chart
        setTimeout(() => {
            const ctx = document.getElementById('pingChart').getContext('2d');
            new Chart(ctx, {
                type: 'bar',
                data: {
                    labels: ['Min', 'Avg', 'Max'],
                    datasets: [{
                        label: 'Round Trip Time (ms)',
                        data: [data.timeMin, data.timeAvg, data.timeMax],
                        backgroundColor: [
                            'rgba(75, 192, 192, 0.2)',
                            'rgba(54, 162, 235, 0.2)',
                            'rgba(255, 99, 132, 0.2)'
                        ],
                        borderColor: [
                            'rgba(75, 192, 192, 1)',
                            'rgba(54, 162, 235, 1)',
                            'rgba(255, 99, 132, 1)'
                        ],
                        borderWidth: 1
                    }]
                },
                options: {
                    scales: {
                        y: {
                            beginAtZero: true,
                            title: {
                                display: true,
                                text: 'Time (ms)'
                            }
                        }
                    }
                }
            });
        }, 100);
    },
    
    // Display traceroute results
    displayTracerouteResults: function(data, element) {
        // Implementation for traceroute results display
        let hopsHtml = '';
        data.hops.forEach(hop => {
            hopsHtml += `
                <tr>
                    <td>${hop.hop}</td>
                    <td>${hop.host}</td>
                    <td>${hop.name || 'N/A'}</td>
                    <td>${hop.rtt.toFixed(3)} ms</td>
                    <td><span class="badge ${hop.status === 'OK' ? 'bg-success' : 'bg-warning'}">${hop.status}</span></td>
                </tr>
            `;
        });
        
        element.innerHTML = `
            <div class="traceroute-results">
                <div class="row mb-4">
                    <div class="col-md-6">
                        <div class="result-card">
                            <div class="result-header">Target Information</div>
                            <div class="result-body">
                                <div class="result-row">
                                    <div class="result-label">Target Host</div>
                                    <div class="result-value">${data.host}</div>
                                </div>
                                <div class="result-row">
                                    <div class="result-label">Hops</div>
                                    <div class="result-value">${data.hops.length}</div>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div class="col-md-6">
                        <canvas id="hopChart" height="200"></canvas>
                    </div>
                </div>
                <div class="result-card">
                    <div class="result-header">Route</div>
                    <div class="result-body">
                        <div class="table-responsive">
                            <table class="table table-striped table-hover">
                                <thead>
                                    <tr>
                                        <th>Hop</th>
                                        <th>IP</th>
                                        <th>Hostname</th>
                                        <th>RTT</th>
                                        <th>Status</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    ${hopsHtml}
                                </tbody>
                            </table>
                        </div>
                    </div>
                </div>
                <div class="result-card">
                    <div class="result-header">Raw Output</div>
                    <div class="result-body">
                        <pre class="raw-output">${data.rawOutput || 'No raw output available'}</pre>
                    </div>
                </div>
            </div>
        `;
        
        // Create RTT chart
        setTimeout(() => {
            const ctx = document.getElementById('hopChart').getContext('2d');
            new Chart(ctx, {
                type: 'line',
                data: {
                    labels: data.hops.map(hop => `Hop ${hop.hop}`),
                    datasets: [{
                        label: 'Round Trip Time (ms)',
                        data: data.hops.map(hop => hop.rtt),
                        backgroundColor: 'rgba(54, 162, 235, 0.2)',
                        borderColor: 'rgba(54, 162, 235, 1)',
                        borderWidth: 2,
                        pointRadius: 5,
                        pointBackgroundColor: 'rgba(54, 162, 235, 1)'
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    scales: {
                        y: {
                            beginAtZero: true,
                            title: {
                                display: true,
                                text: 'Time (ms)'
                            }
                        }
                    }
                }
            });
        }, 100);
    },
    
    // Display port scanner results
    displayPortScannerResults: function(data, element) {
        // Implementation for port scanner results display
        let portsHtml = '';
        data.openPorts.forEach(port => {
            portsHtml += `
                <tr>
                    <td>${port.port}</td>
                    <td>${port.service}</td>
                    <td><span class="badge bg-success">${port.status}</span></td>
                </tr>
            `;
        });
        
        element.innerHTML = `
            <div class="port-scanner-results">
                <div class="row mb-4">
                    <div class="col-md-6">
                        <div class="result-card">
                            <div class="result-header">Scan Information</div>
                            <div class="result-body">
                                <div class="result-row">
                                    <div class="result-label">Host</div>
                                    <div class="result-value">${data.host}</div>
                                </div>
                                <div class="result-row">
                                    <div class="result-label">Port Range</div>
                                    <div class="result-value">${data.portRange}</div>
                                </div>
                                <div class="result-row">
                                    <div class="result-label">Open Ports</div>
                                    <div class="result-value">${data.openPorts.length}</div>
                                </div>
                                <div class="result-row">
                                    <div class="result-label">Closed Ports</div>
                                    <div class="result-value">${data.closedPorts}</div>
                                </div>
                                <div class="result-row">
                                    <div class="result-label">Scan Time</div>
                                    <div class="result-value">${data.scanTime.toFixed(3)} seconds</div>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div class="col-md-6">
                        <canvas id="portChart" height="200"></canvas>
                    </div>
                </div>
                <div class="result-card">
                    <div class="result-header">Open Ports</div>
                    <div class="result-body">
                        <div class="table-responsive">
                            <table class="table table-striped table-hover">
                                <thead>
                                    <tr>
                                        <th>Port</th>
                                        <th>Service</th>
                                        <th>Status</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    ${portsHtml}
                                </tbody>
                            </table>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        // Create port distribution chart
        setTimeout(() => {
            const ctx = document.getElementById('portChart').getContext('2d');
            new Chart(ctx, {
                type: 'pie',
                data: {
                    labels: ['Open Ports', 'Closed Ports'],
                    datasets: [{
                        label: 'Ports',
                        data: [data.openPorts.length, data.closedPorts],
                        backgroundColor: [
                            'rgba(75, 192, 192, 0.2)',
                            'rgba(255, 99, 132, 0.2)'
                        ],
                        borderColor: [
                            'rgba(75, 192, 192, 1)',
                            'rgba(255, 99, 132, 1)'
                        ],
                        borderWidth: 1
                    }]
                }
            });
        }, 100);
    },
    
    // Display DNS lookup results
    displayDNSLookupResults: function(data, element) {
        // Implementation for DNS lookup results display
        let recordsHtml = '';
        Object.entries(data.results).forEach(([type, records]) => {
            if (records && records.length > 0) {
                recordsHtml += `
                    <div class="dns-record-type">
                        <h5 class="record-type">${type} Records</h5>
                        <ul class="record-list">
                `;
                
                records.forEach(record => {
                    recordsHtml += `<li>${record}</li>`;
                });
                
                recordsHtml += `
                        </ul>
                    </div>
                `;
            }
        });
        
        element.innerHTML = `
            <div class="dns-lookup-results">
                <div class="result-card">
                    <div class="result-header">Domain Information</div>
                    <div class="result-body">
                        <div class="result-row">
                            <div class="result-label">Domain</div>
                            <div class="result-value">${data.domain}</div>
                        </div>
                        <div class="result-row">
                            <div class="result-label">Record Type</div>
                            <div class="result-value">${data.recordType}</div>
                        </div>
                    </div>
                </div>
                
                <div class="result-card">
                    <div class="result-header">DNS Records</div>
                    <div class="result-body">
                        <div class="dns-records">
                            ${recordsHtml || '<p class="text-muted">No records found</p>'}
                        </div>
                    </div>
                </div>
            </div>
        `;
    },
    
    // Display bandwidth test results
    displayBandwidthResults: function(data, element) {
        // Implementation for bandwidth test results display
        element.innerHTML = `
            <div class="bandwidth-results">
                <div class="row mb-4">
                    <div class="col-md-6">
                        <div class="result-card">
                            <div class="result-header">Test Information</div>
                            <div class="result-body">
                                <div class="result-row">
                                    <div class="result-label">Server</div>
                                    <div class="result-value">${data.server}</div>
                                </div>
                                <div class="result-row">
                                    <div class="result-label">Download Speed</div>
                                    <div class="result-value">${data.downloadSpeed.toFixed(2)} Mbps</div>
                                </div>
                                <div class="result-row">
                                    <div class="result-label">Upload Speed</div>
                                    <div class="result-value">${data.uploadSpeed.toFixed(2)} Mbps</div>
                                </div>
                                <div class="result-row">
                                    <div class="result-label">Latency</div>
                                    <div class="result-value">${data.latency} ms</div>
                                </div>
                                <div class="result-row">
                                    <div class="result-label">Jitter</div>
                                    <div class="result-value">${data.jitter} ms</div>
                                </div>
                                <div class="result-row">
                                    <div class="result-label">Packet Loss</div>
                                    <div class="result-value">${data.packetLoss}%</div>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div class="col-md-6">
                        <div class="speed-gauge">
                            <div class="gauge-label">Download</div>
                            <div class="gauge-value">${data.downloadSpeed.toFixed(2)}<span class="unit">Mbps</span></div>
                            <progress class="gauge-progress download" value="${data.downloadSpeed}" max="100"></progress>
                            
                            <div class="gauge-label mt-4">Upload</div>
                            <div class="gauge-value">${data.uploadSpeed.toFixed(2)}<span class="unit">Mbps</span></div>
                            <progress class="gauge-progress upload" value="${data.uploadSpeed}" max="100"></progress>
                        </div>
                    </div>
                </div>
                
                <div class="result-card">
                    <div class="result-header">Speed Over Time</div>
                    <div class="result-body">
                        <canvas id="speedChart" height="200"></canvas>
                    </div>
                </div>
            </div>
        `;
        
        // Create speed chart
        setTimeout(() => {
            if (data.chart) {
                const ctx = document.getElementById('speedChart').getContext('2d');
                new Chart(ctx, {
                    type: 'line',
                    data: {
                        labels: data.chart.time.map(t => `${t}s`),
                        datasets: [
                            {
                                label: 'Download (Mbps)',
                                data: data.chart.download,
                                borderColor: 'rgba(75, 192, 192, 1)',
                                backgroundColor: 'rgba(75, 192, 192, 0.2)',
                                borderWidth: 2,
                                fill: true
                            },
                            {
                                label: 'Upload (Mbps)',
                                data: data.chart.upload,
                                borderColor: 'rgba(54, 162, 235, 1)',
                                backgroundColor: 'rgba(54, 162, 235, 0.2)',
                                borderWidth: 2,
                                fill: true
                            }
                        ]
                    },
                    options: {
                        responsive: true,
                        scales: {
                            y: {
                                beginAtZero: true,
                                title: {
                                    display: true,
                                    text: 'Speed (Mbps)'
                                }
                            }
                        }
                    }
                });
            }
        }, 100);
    },
    
    // Display external plugin results
    displayExternalPluginResults: function(data, element) {
        // Check if we have raw output or structured data
        if (data.raw_output) {
            element.innerHTML = `
                <div class="external-plugin-results">
                    <div class="result-card">
                        <div class="result-header">External Plugin Output</div>
                        <div class="result-body">
                            <pre class="raw-output">${data.raw_output}</pre>
                        </div>
                    </div>
                </div>
            `;
            return;
        }
        
        // If we have structured data, display it nicely
        let resultsHtml = '<div class="external-plugin-results">';
        
        // Check if we have a custom title
        if (data.title) {
            resultsHtml += `
                <div class="alert alert-info">
                    <h5>${data.title}</h5>
                    ${data.description ? `<p>${data.description}</p>` : ''}
                </div>
            `;
        }
        
        // Add key-value pairs in a card
        resultsHtml += '<div class="result-card"><div class="result-header">Results</div><div class="result-body">';
        
        Object.entries(data).forEach(([key, value]) => {
            // Skip special keys used for display purposes
            if (key === 'title' || key === 'description') return;
            
            // Format different value types
            let displayValue = '';
            if (typeof value === 'object' && value !== null) {
                displayValue = `<pre class="json-result">${JSON.stringify(value, null, 2)}</pre>`;
            } else {
                displayValue = value;
            }
            
            resultsHtml += `
                <div class="result-row">
                    <div class="result-label">${key}</div>
                    <div class="result-value">${displayValue}</div>
                </div>
            `;
        });
        
        resultsHtml += '</div></div></div>';
        element.innerHTML = resultsHtml;
    },
    
    // Display Wi-Fi Device Locator results
    displayWifiDeviceLocatorResults: function(data, element) {
        if (data.error) {
            element.innerHTML = `<div class="alert alert-danger">${data.error}</div>`;
            return;
        }
        
        let resultsHtml = `
            <div class="card mb-4">
                <div class="card-header">
                    <h5 class="mb-0">Wi-Fi Device Locator Results</h5>
                </div>
                <div class="card-body">
                    <div class="result-summary">
                        <p><strong>Interface:</strong> ${data.interface}</p>
                        <p><strong>Scan Time:</strong> ${data.scan_time} seconds</p>
                        <p><strong>Scan Mode:</strong> ${data.scan_mode === 'all' ? 'All Devices' : 
                                                        data.scan_mode === 'ap' ? 'Access Points Only' : 
                                                        'Client Devices Only'}</p>
                        <p><strong>Min Signal Strength:</strong> ${data.min_signal_strength} dBm</p>
                        <p><strong>Devices Found:</strong> ${data.device_count} (${data.ap_count} APs, ${data.client_count} Clients)</p>
                    </div>
                    
                    <div class="device-list mt-4">
                        <h5 class="mb-3">Devices by Signal Strength</h5>
                        <div class="table-responsive">
                            <table class="table table-striped table-bordered">
                                <thead>
                                    <tr>
                                        <th>Type</th>
                                        <th>BSSID (MAC)</th>
                                        <th>SSID</th>
                                        <th>Signal</th>
                                        <th>Quality</th>
                                        <th>Channel</th>
                                        <th>Last Seen</th>
                                    </tr>
                                </thead>
                                <tbody>`;
        
        // Add each device to the table
        data.devices.forEach(device => {
            // Create color-coded signal indicator
            const signalQuality = device.signal_quality || 0;
            let signalClass = '';
            
            if (signalQuality >= 80) {
                signalClass = 'bg-success';
            } else if (signalQuality >= 60) {
                signalClass = 'bg-info';
            } else if (signalQuality >= 40) {
                signalClass = 'bg-warning';
            } else {
                signalClass = 'bg-danger';
            }
            
            const signalBar = `
                <div class="signal-bar-container">
                    <div class="signal-bar ${signalClass}" style="width: ${signalQuality}%"></div>
                    <span class="signal-text">${device.signal_strength} dBm (${signalQuality}%)</span>
                </div>
            `;
            
            // Get device type icon
            const deviceIcon = device.device_type === 'AP' ? 
                '<i class="fas fa-wifi" title="Access Point"></i>' : 
                '<i class="fas fa-laptop" title="Client Device"></i>';
            
            // Format signal trend if available
            let trendIndicator = '';
            if (device.signal_trend) {
                if (device.signal_trend === 'increasing') {
                    trendIndicator = ' <i class="fas fa-arrow-up text-success" title="Signal increasing (getting closer)"></i>';
                } else if (device.signal_trend === 'decreasing') {
                    trendIndicator = ' <i class="fas fa-arrow-down text-danger" title="Signal decreasing (getting farther)"></i>';
                } else {
                    trendIndicator = ' <i class="fas fa-minus text-secondary" title="Signal stable"></i>';
                }
            }
            
            resultsHtml += `
                <tr>
                    <td>${deviceIcon} ${device.device_type}</td>
                    <td>${device.bssid}</td>
                    <td>${device.ssid || '<em>Hidden</em>'}</td>
                    <td>${signalBar}${trendIndicator}</td>
                    <td>${device.signal_quality}%</td>
                    <td>${device.channel || 'N/A'}</td>
                    <td>${this.formatTimestamp(device.last_seen)}</td>
                </tr>`;
        });
        
        resultsHtml += `
                                </tbody>
                            </table>
                        </div>
                    </div>
                </div>
            </div>
            
            <div class="card mb-4">
                <div class="card-header">
                    <h5 class="mb-0">Device Location Tips</h5>
                </div>
                <div class="card-body">
                    <ul>
                        <li><strong>Signal Strength:</strong> The higher the dBm value (closer to 0), the stronger the signal and the closer the device</li>
                        <li><strong>Tracking:</strong> For continuous tracking, enable "Continuous Scan" option and watch for signal trend indicators</li>
                        <li><strong>Multiple Scans:</strong> Move around and run multiple scans to triangulate the device location</li>
                        <li><strong>Best Practices:</strong> Use in open areas for better accuracy, walls and obstacles can affect signal</li>
                    </ul>
                </div>
            </div>`;
        
        // Add CSS for signal bars
        resultsHtml += `
            <style>
                .signal-bar-container {
                    width: 100%;
                    background-color: #f5f5f5;
                    border-radius: 4px;
                    height: 20px;
                    position: relative;
                    overflow: hidden;
                }
                .signal-bar {
                    height: 100%;
                    border-radius: 3px;
                }
                .signal-text {
                    position: absolute;
                    left: 5px;
                    top: 0;
                    font-size: 12px;
                    color: #333;
                    line-height: 20px;
                    text-shadow: 0 0 2px #fff;
                }
            </style>`;
        
        element.innerHTML = resultsHtml;
        
        // Add auto-refresh for continuous mode
        if (data.continuous_scan) {
            setTimeout(() => {
                document.getElementById('runPluginBtn').click();
            }, 3000); // Refresh every 3 seconds
        }
    },
    
    // Helper to format timestamp
    formatTimestamp: function(isoTimestamp) {
        const date = new Date(isoTimestamp);
        return date.toLocaleTimeString();
    },
    
    // Register a custom display function for a specific plugin
    registerCustomDisplay: function(pluginId, displayFunction) {
        this.customDisplayFunctions[pluginId] = displayFunction;
    }
};

// Initialize the plugin manager when DOM is loaded
document.addEventListener('DOMContentLoaded', function() {
    PluginManager.init();
});
