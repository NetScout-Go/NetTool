/**
 * NetTool Dashboard JavaScript
 * Handles main dashboard functionality and WebSocket communications
 */

// Traffic data history for charts
const trafficHistory = {
    timestamps: [],
    download: [],
    upload: [],
    maxDataPoints: 20 // Maximum number of data points to show
};

// Dashboard charts
let trafficChart;
let topologyChart;

// Network data cache
let lastNetworkData = null;

// Speed test variables
let lastSpeedTest = null;
let speedTestInProgress = false;
let speedTestTimer = null;

// Initialize WebSocket connection
let socket = null;
let reconnectTimeout = null;

// Initialize dashboard on document load
document.addEventListener('DOMContentLoaded', function() {
    initializeCharts();
    initializeWebSocket();
    fetchNetworkInfo();
    
    // Refresh data periodically in case WebSocket fails
    setInterval(fetchNetworkInfo, 10000);
    
    // Set status indicators pulse effect
    setInterval(() => {
        const indicators = document.querySelectorAll('.realtime-indicator');
        indicators.forEach(ind => {
            ind.classList.add('pulse');
            setTimeout(() => ind.classList.remove('pulse'), 1000);
        });
    }, 2000);

    // Add listeners to run speed test when connection is detected
    document.addEventListener('connection-status-change', checkAndRunSpeedTest);
    
    // Add event listener for manual speed test button
    const speedTestButton = document.getElementById('runSpeedTestBtn');
    if (speedTestButton) {
        speedTestButton.addEventListener('click', function() {
            runSpeedTest();
            this.disabled = true;
            this.innerHTML = '<span class="spinner-border spinner-border-sm" role="status"></span> Testing...';
            
            // Re-enable button after 15 seconds (test duration + buffer)
            setTimeout(() => {
                this.disabled = false;
                this.innerHTML = '<i class="bi bi-speedometer2"></i> Run Speed Test';
            }, 15000);
        });
    }
});

// Function to add data to traffic history
function addTrafficData(timestamp, downloadMbps, uploadMbps) {
    trafficHistory.timestamps.push(timestamp);
    trafficHistory.download.push(downloadMbps);
    trafficHistory.upload.push(uploadMbps);
    
    // Keep only the last maxDataPoints
    if (trafficHistory.timestamps.length > trafficHistory.maxDataPoints) {
        trafficHistory.timestamps.shift();
        trafficHistory.download.shift();
        trafficHistory.upload.shift();
    }
    
    // Update traffic chart if it exists
    if (trafficChart) {
        trafficChart.data.labels = trafficHistory.timestamps;
        trafficChart.data.datasets[0].data = trafficHistory.download;
        trafficChart.data.datasets[1].data = trafficHistory.upload;
        trafficChart.update();
    }
}

// Format bytes to human-readable format
function formatBytes(bytes, decimals = 2) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
}

// Format uptime to human-readable format
function formatUptime(seconds) {
    if (isNaN(seconds)) return '--:--:--';
    
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = Math.floor(seconds % 60);
    
    return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
}

// Format a MAC address with colons and uppercase
function formatMacAddress(mac) {
    if (!mac) return '--';
    return mac.toUpperCase();
}

// Get appropriate color for the ARP state
function getStateColor(state) {
    if (!state) return '';
    
    switch(state) {
        case 'REACHABLE':
            return 'text-success';
        case 'STALE':
        case 'DELAY':
        case 'PROBE':
            return 'text-warning';
        case 'FAILED':
        case 'INCOMPLETE': 
        case 'NOARP':
            return 'text-danger';
        case 'PERMANENT':
            return 'text-info';
        default:
            return '';
    }
}

// Initialize network topology visualization
function initializeTopology(data) {
    const container = document.getElementById('networkTopology');
    if (!container) return;
    
    // Check if we have valid connection data and traffic data
    const latency = (data.connection && typeof data.connection.latencyMS !== 'undefined') 
        ? `${data.connection.latencyMS.toFixed(1)} ms` 
        : '-- ms';
        
    const bandwidth = (data.traffic && typeof data.traffic.currentBandwidth !== 'undefined') 
        ? `${data.traffic.currentBandwidth.toFixed(1)} Mbps` 
        : '-- Mbps';
    
    const gateway = data.gateway || '--';
    const ipAddress = data.ipv4Address || '--';
    const deviceType = data.ssid ? 'laptop' : 'pc-display';
    const connectionType = data.ssid ? `Connected to ${data.ssid}` : 'Ethernet';
    
    // Basic topology visualization
    let html = `
        <div class="network-topology">
            <div class="device device-internet">
                <i class="bi bi-globe"></i>
                <div class="device-name">Internet</div>
            </div>
            <div class="connection-line">
                <div class="connection-speed">${latency}</div>
            </div>
            <div class="device device-gateway">
                <i class="bi bi-router"></i>
                <div class="device-name">Gateway (${gateway})</div>
            </div>
            <div class="connection-line">
                <div class="connection-speed">${bandwidth}</div>
            </div>
            <div class="device device-current">
                <i class="bi bi-${deviceType}"></i>
                <div class="device-name">NetScout-Go (${ipAddress})</div>
                <div class="device-detail">${connectionType}</div>
            </div>
        </div>
    `;
    
    container.innerHTML = html;
}

// Initialize traffic chart
function initializeTrafficChart() {
    const ctx = document.getElementById('trafficChart');
    if (!ctx) return;
    
    trafficChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: trafficHistory.timestamps,
            datasets: [
                {
                    label: 'Download (Mbps)',
                    borderColor: 'rgba(75, 192, 192, 1)',
                    backgroundColor: 'rgba(75, 192, 192, 0.2)',
                    data: trafficHistory.download,
                    fill: true
                },
                {
                    label: 'Upload (Mbps)',
                    borderColor: 'rgba(54, 162, 235, 1)',
                    backgroundColor: 'rgba(54, 162, 235, 0.2)',
                    data: trafficHistory.upload,
                    fill: true
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            animation: {
                duration: 500
            },
            scales: {
                x: {
                    grid: {
                        display: false
                    }
                },
                y: {
                    beginAtZero: true,
                    title: {
                        display: true,
                        text: 'Speed (Mbps)'
                    }
                }
            },
            plugins: {
                legend: {
                    position: 'top',
                },
                tooltip: {
                    mode: 'index',
                    intersect: false
                }
            }
        
        }
        });
}

// Initialize the WebSocket connection
function initializeWebSocket() {
    // Close any existing connection
    if (socket) {
        socket.close();
    }
    
    // Clear any pending reconnect
    if (reconnectTimeout) {
        clearTimeout(reconnectTimeout);
    }
    
    // Create new WebSocket connection
    socket = new WebSocket(`ws://${window.location.host}/ws`);
    
    socket.onopen = function(e) {
        console.log('WebSocket connection established');
        
        // Set all indicators to active
        document.querySelectorAll('.realtime-indicator').forEach(ind => {
            ind.classList.add('active');
        });
    };
    
    socket.onmessage = function(event) {
        try {
            const data = JSON.parse(event.data);
            if (data.type === 'network_update') {
                updateDashboard(data.data);
            }
        } catch (error) {
            console.error('Error processing WebSocket message:', error);
        }
    };
    
    socket.onclose = function(event) {
        console.log('WebSocket connection closed');
        
        // Set all indicators to inactive
        document.querySelectorAll('.realtime-indicator').forEach(ind => {
            ind.classList.remove('active');
        });
        
        // Attempt to reconnect after 5 seconds
        reconnectTimeout = setTimeout(() => {
            console.log('Attempting to reconnect WebSocket...');
            initializeWebSocket();
        }, 5000);
    };
    
    socket.onerror = function(error) {
        console.error('WebSocket error:', error);
    };
}

// Initialize charts
function initializeCharts() {
    initializeTrafficChart();
}

// Fetch network information from API
function fetchNetworkInfo() {
    console.log("Fetching network info from API...");
    fetch('/api/network-info')
        .then(response => {
            console.log("API Response status:", response.status);
            if (!response.ok) {
                throw new Error(`Network response was not ok: ${response.status}`);
            }
            return response.json();
        })
        .then(data => {
            console.log("Network data received:", JSON.stringify(data));
            if (!data) {
                console.error("Empty data received from API");
                return;
            }
            
            // Log service latency data specifically
            console.log("Service Latency Data:", JSON.stringify(data.serviceLatency));
            
            // Update the dashboard with the data
            updateDashboard(data);
        })
        .catch(error => {
            console.error('Error fetching network info:', error);
        });
}

// Update dashboard with network info
function updateDashboard(data) {
    // Cache the network data
    lastNetworkData = data;
    
    // Update connection status
    updateConnectionStatus(data);
    
    // Update IP configuration
    updateIPConfiguration(data);
    
    // Update connection metrics
    updateConnectionMetrics(data);
    
    // Update DNS servers
    updateDNSServers(data);
    
    // Update interface details
    updateInterfaceDetails(data);
    
    // Update traffic statistics
    updateTrafficStatistics(data);
    
    // Update service latency (try both methods)
    updateServiceLatency(data);
    setServiceLatencies(data); // Direct update as fallback
    
    // Update ARP table
    updateARPTable(data);
    
    // Update network topology
    updateNetworkTopology(data);
    
    // Add data point to traffic chart
    addTrafficDataPoint(data);
    
    // Update last updated timestamp
    updateTimestamp(data);
}

// Update service latency values
function updateServiceLatency(data) {
    console.log("Updating service latency with data:", data.serviceLatency);
    
    // Check DOM elements directly
    console.log("googleLatency element exists:", document.getElementById('googleLatency') !== null);
    console.log("amazonLatency element exists:", document.getElementById('amazonLatency') !== null);
    console.log("cloudflareLatency element exists:", document.getElementById('cloudflareLatency') !== null);
    console.log("microsoftLatency element exists:", document.getElementById('microsoftLatency') !== null);
    
    if (!data.serviceLatency) {
        console.warn("No serviceLatency data available");
        return;
    }
    
    // Update major service latencies
    const services = {
        'googleLatency': data.serviceLatency.google,
        'amazonLatency': data.serviceLatency.amazon,
        'cloudflareLatency': data.serviceLatency.cloudflare,
        'microsoftLatency': data.serviceLatency.microsoft
    };
    
    console.log("Service latencies:", services);
    
    // Update each service latency value directly without class manipulation first
    for (const [elementId, value] of Object.entries(services)) {
        const element = document.getElementById(elementId);
        if (element) {
            console.log(`Updating ${elementId} with value ${value}`);
            element.textContent = value > 0 ? `${value.toFixed(1)} ms` : '-- ms';
        } else {
            console.warn(`Element with id ${elementId} not found in the DOM`);
        }
    }
    
    // Update network service latencies (DNS and HTTP)
    if (data.serviceLatency.dns !== undefined) {
        updateElementText('dnsLatency', data.serviceLatency.dns > 0 ? 
            `${data.serviceLatency.dns.toFixed(1)} ms` : '-- ms');
            
        // Update progress bar
        const dnsBar = document.getElementById('dnsLatencyBar');
        if (dnsBar) {
            const percentage = Math.min(data.serviceLatency.dns / 2, 100);
            dnsBar.style.width = `${percentage}%`;
            dnsBar.className = 'progress-bar';
            if (data.serviceLatency.dns < 50) {
                dnsBar.classList.add('bg-success');
            } else if (data.serviceLatency.dns < 100) {
                dnsBar.classList.add('bg-warning');
            } else {
                dnsBar.classList.add('bg-danger');
            }
        }
    }
    
    if (data.serviceLatency.http !== undefined) {
        updateElementText('httpLatency', data.serviceLatency.http > 0 ? 
            `${data.serviceLatency.http.toFixed(1)} ms` : '-- ms');
            
        // Update progress bar
        const httpBar = document.getElementById('httpLatencyBar');
        if (httpBar) {
            const percentage = Math.min(data.serviceLatency.http / 2, 100);
            httpBar.style.width = `${percentage}%`;
            httpBar.className = 'progress-bar';
            if (data.serviceLatency.http < 100) {
                httpBar.classList.add('bg-success');
            } else if (data.serviceLatency.http < 200) {
                httpBar.classList.add('bg-warning');
            } else {
                httpBar.classList.add('bg-danger');
            }
        }
    }
}

// Direct function to update service latencies (simplified for debugging)
function setServiceLatencies(data) {
    if (!data || !data.serviceLatency) return;
    
    // Get elements
    const googleEl = document.getElementById('googleLatency');
    const amazonEl = document.getElementById('amazonLatency');
    const cloudflareEl = document.getElementById('cloudflareLatency');
    const microsoftEl = document.getElementById('microsoftLatency');
    
    // Update Google
    if (googleEl && data.serviceLatency.google) {
        googleEl.textContent = `${data.serviceLatency.google.toFixed(1)} ms`;
    }
    
    // Update Amazon
    if (amazonEl && data.serviceLatency.amazon) {
        amazonEl.textContent = `${data.serviceLatency.amazon.toFixed(1)} ms`;
    }
    
    // Update Cloudflare
    if (cloudflareEl && data.serviceLatency.cloudflare) {
        cloudflareEl.textContent = `${data.serviceLatency.cloudflare.toFixed(1)} ms`;
    }
    
    // Update Microsoft
    if (microsoftEl && data.serviceLatency.microsoft) {
        microsoftEl.textContent = `${data.serviceLatency.microsoft.toFixed(1)} ms`;
    }
    
    // Also update the network service latencies
    const dnsLatencyEl = document.getElementById('dnsLatency');
    const httpLatencyEl = document.getElementById('httpLatency');
    
    if (dnsLatencyEl && data.serviceLatency.dns) {
        dnsLatencyEl.textContent = `${data.serviceLatency.dns.toFixed(1)} ms`;
    }
    
    if (httpLatencyEl && data.serviceLatency.http) {
        httpLatencyEl.textContent = `${data.serviceLatency.http.toFixed(1)} ms`;
    }
}

// Update connection status
function updateConnectionStatus(data) {
    const statusElement = document.getElementById('connectionStatus');
    if (!statusElement) return;
    
    // Keep track of previous connection status to detect changes
    const previousStatus = lastNetworkData && lastNetworkData.connection ? lastNetworkData.connection.status : null;
    const currentStatus = data.connection.status;
    
    if (currentStatus === 'connected') {
        statusElement.innerHTML = '<i class="bi bi-wifi status-icon connected"></i><span class="status-text">Connected</span>';
    } else if (currentStatus === 'limited') {
        statusElement.innerHTML = '<i class="bi bi-wifi-1 status-icon limited"></i><span class="status-text">Limited</span>';
    } else {
        statusElement.innerHTML = '<i class="bi bi-wifi-off status-icon disconnected"></i><span class="status-text">Disconnected</span>';
    }
    
    // Update uptime
    const uptimeElement = document.getElementById('uptime');
    if (uptimeElement) {
        uptimeElement.textContent = formatUptime(data.connection.uptime);
    }
    
    // Update connection type
    const connectionTypeElement = document.getElementById('connectionType');
    if (connectionTypeElement) {
        const connectionType = data.ssid ? `Wi-Fi (${data.ssid})` : 'Ethernet';
        connectionTypeElement.textContent = connectionType;
    }
    
    // Emit an event if connection status changed
    if (previousStatus !== currentStatus) {
        console.log(`Connection status changed from ${previousStatus} to ${currentStatus}`);
        document.dispatchEvent(new CustomEvent('connection-status-change', {
            detail: {
                previousStatus: previousStatus,
                status: currentStatus
            }
        }));
    }
}

// Update IP configuration
function updateIPConfiguration(data) {
    updateElementText('ipv4Address', data.ipv4Address || '--');
    updateElementText('subnetMask', data.subnetMask || '--');
    updateElementText('gateway', data.gateway || '--');
    updateElementText('ipv6Address', data.ipv6Address || '--');
}

// Update connection metrics
function updateConnectionMetrics(data) {
    console.log("Connection metrics:", data.connection); // Debug log
    console.log("Traffic data:", data.traffic); // Debug log
    
    // Safely update metrics with null checks
    if (data.connection) {
        // Note camelCase difference: JSON has latencyMs but code was looking for latencyMS
        const latency = data.connection.latencyMs !== undefined ? data.connection.latencyMs : 
                       (data.connection.latencyMS !== undefined ? data.connection.latencyMS : 0);
        
        updateElementText('latency', latency > 0 ? `${latency.toFixed(1)} ms` : '-- ms');
        updateElementText('packetLoss', data.connection.packetLoss !== undefined ? `${data.connection.packetLoss.toFixed(1)}%` : '--%');
        
        if (data.connection.signalStrength) {
            updateElementText('signalStrength', `${data.connection.signalStrength} dBm`);
        } else {
            updateElementText('signalStrength', 'N/A');
        }
    } else {
        updateElementText('latency', '-- ms');
        updateElementText('packetLoss', '--%');
        updateElementText('signalStrength', 'N/A');
    }
    
    // Safe bandwidth update
    if (data.traffic && typeof data.traffic.currentBandwidth !== 'undefined') {
        updateElementText('bandwidth', `${data.traffic.currentBandwidth.toFixed(1)} Mbps`);
    } else {
        updateElementText('bandwidth', '-- Mbps');
    }
}

// Update DNS servers
function updateDNSServers(data) {
    const dnsElement = document.getElementById('dnsServers');
    if (!dnsElement) {
        console.error("DNS Servers element not found in DOM!");
        return;
    }
    
    dnsElement.innerHTML = '';
    console.log("DNS Servers data:", data.dnsServers); // Debug log
    
    if (data.dnsServers && Array.isArray(data.dnsServers) && data.dnsServers.length > 0) {
        data.dnsServers.forEach((server, index) => {
            dnsElement.innerHTML += `<div>DNS ${index + 1}: ${server}</div>`;
        });
    } else {
        dnsElement.innerHTML = '<div>No DNS servers configured</div>';
    }
}

// Update interface details
function updateInterfaceDetails(data) {
    console.log("Interface details:", data.ethernetInfo); // Debug log
    
    // Check if DOM elements exist
    const interfaceNameEl = document.getElementById('interfaceName');
    const macAddressEl = document.getElementById('macAddress');
    const linkSpeedEl = document.getElementById('linkSpeed');
    const duplexEl = document.getElementById('duplex');
    const ssidEl = document.getElementById('ssid');
    const vlanInfoEl = document.getElementById('vlanInfo');
    
    if (!interfaceNameEl) console.error("Element 'interfaceName' not found in DOM!");
    if (!macAddressEl) console.error("Element 'macAddress' not found in DOM!");
    if (!linkSpeedEl) console.error("Element 'linkSpeed' not found in DOM!");
    if (!duplexEl) console.error("Element 'duplex' not found in DOM!");
    if (!ssidEl) console.error("Element 'ssid' not found in DOM!");
    if (!vlanInfoEl) console.error("Element 'vlanInfo' not found in DOM!");
    
    // Safely update interface details with null checks
    if (data.ethernetInfo) {
        updateElementText('interfaceName', data.ethernetInfo.interfaceName || '--');
        updateElementText('macAddress', data.ethernetInfo.macAddress || '--');
        updateElementText('linkSpeed', data.ethernetInfo.speed || '--');
        updateElementText('duplex', data.ethernetInfo.duplex || '--');
    } else {
        // Set default values if ethernetInfo is missing
        updateElementText('interfaceName', '--');
        updateElementText('macAddress', '--');
        updateElementText('linkSpeed', '--');
        updateElementText('duplex', '--');
    }
    
    updateElementText('ssid', data.ssid || 'N/A');
    
    // Format VLAN info
    if (vlanInfoEl) {
        if (data.vlanInfo && data.vlanInfo.enabled) {
            // Check if we're dealing with vlanId (camelCase issue)
            const vlanId = data.vlanInfo.vlanId !== undefined ? data.vlanInfo.vlanId : 
                         (data.vlanInfo.VLANID !== undefined ? data.vlanInfo.VLANID : 0);
            
            vlanInfoEl.innerHTML = `ID: ${vlanId}<br>Name: ${data.vlanInfo.name || ''}`;
        } else {
            vlanInfoEl.textContent = 'Not configured';
        }
    }
}

// Update traffic statistics
function updateTrafficStatistics(data) {
    console.log("Traffic statistics:", data.traffic); // Debug log
    console.log("DHCP info:", data.dhcpInfo); // Debug log
    
    // Check if DOM elements exist
    const bytesReceivedEl = document.getElementById('bytesReceived');
    const bytesSentEl = document.getElementById('bytesSent');
    const packetsReceivedEl = document.getElementById('packetsReceived');
    const packetsSentEl = document.getElementById('packetsSent');
    const dhcpStatusEl = document.getElementById('dhcpStatus');
    const dhcpInfoEl = document.getElementById('dhcpInfo');
    
    if (!bytesReceivedEl) console.error("Element 'bytesReceived' not found in DOM!");
    if (!bytesSentEl) console.error("Element 'bytesSent' not found in DOM!");
    if (!packetsReceivedEl) console.error("Element 'packetsReceived' not found in DOM!");
    if (!packetsSentEl) console.error("Element 'packetsSent' not found in DOM!");
    if (!dhcpStatusEl) console.error("Element 'dhcpStatus' not found in DOM!");
    if (!dhcpInfoEl) console.error("Element 'dhcpInfo' not found in DOM!");
    
    // Safely update traffic statistics with null checks
    if (data.traffic) {
        updateElementText('bytesReceived', typeof data.traffic.bytesReceived !== 'undefined' ? formatBytes(data.traffic.bytesReceived) : '--');
        updateElementText('bytesSent', typeof data.traffic.bytesSent !== 'undefined' ? formatBytes(data.traffic.bytesSent) : '--');
        updateElementText('packetsReceived', typeof data.traffic.packetsReceived !== 'undefined' ? data.traffic.packetsReceived.toLocaleString() : '--');
        updateElementText('packetsSent', typeof data.traffic.packetsSent !== 'undefined' ? data.traffic.packetsSent.toLocaleString() : '--');
    } else {
        updateElementText('bytesReceived', '--');
        updateElementText('bytesSent', '--');
        updateElementText('packetsReceived', '--');
        updateElementText('packetsSent', '--');
    }
    
    // Safely update DHCP info
    if (data.dhcpInfo) {
        updateElementText('dhcpStatus', data.dhcpInfo.enabled ? 'Enabled' : 'Disabled');
        
        // Handle DHCP info
        if (dhcpInfoEl) {
            if (data.dhcpInfo.enabled) {
                let dhcpInfoText = `Server: ${data.dhcpInfo.dhcpServer || data.dhcpInfo.DHCPServer || 'Unknown'}`;
                
                if (data.dhcpInfo.leaseObtained && data.dhcpInfo.leaseObtained !== "0001-01-01T00:00:00Z") {
                    const leaseObtained = new Date(data.dhcpInfo.leaseObtained);
                    dhcpInfoText += `<br>Obtained: ${leaseObtained.toLocaleString()}`;
                }
                
                if (data.dhcpInfo.leaseExpires && data.dhcpInfo.leaseExpires !== "0001-01-01T00:00:00Z") {
                    const leaseExpires = new Date(data.dhcpInfo.leaseExpires);
                    dhcpInfoText += `<br>Expires: ${leaseExpires.toLocaleString()}`;
                }
                
                dhcpInfoEl.innerHTML = dhcpInfoText;
            } else {
                dhcpInfoEl.textContent = 'Static Configuration';
            }
        }
    } else {
        updateElementText('dhcpStatus', '--');
        if (dhcpInfoEl) {
            dhcpInfoEl.textContent = '--';
        }
    }
}

// Update ARP table
function updateARPTable(data) {
    const arpTableBody = document.getElementById('arpTable');
    if (!arpTableBody) return;
    
    arpTableBody.innerHTML = '';
    
    if (data.arpEntries && data.arpEntries.length > 0) {
        data.arpEntries.forEach(entry => {
            const row = document.createElement('tr');
            
            // IP Address cell
            const ipCell = document.createElement('td');
            ipCell.textContent = entry.ipAddress;
            row.appendChild(ipCell);
            
            // MAC Address cell
            const macCell = document.createElement('td');
            macCell.textContent = formatMacAddress(entry.macAddress);
            row.appendChild(macCell);
            
            // Interface cell
            const interfaceCell = document.createElement('td');
            interfaceCell.textContent = entry.device;
            row.appendChild(interfaceCell);
            
            // State cell
            const stateCell = document.createElement('td');
            stateCell.textContent = entry.state || 'UNKNOWN';
            stateCell.classList.add(getStateColor(entry.state));
            row.appendChild(stateCell);
            
            arpTableBody.appendChild(row);
        });
    } else {
        arpTableBody.innerHTML = '<tr><td colspan="4" class="text-center">No ARP entries found</td></tr>';
    }
}

// Update network topology
function updateNetworkTopology(data) {
    const topologyElement = document.getElementById('networkTopology');
    if (!topologyElement) return;
    
    // Generate simple topology HTML
    topologyElement.innerHTML = generateSimpleTopologyHTML(data);
}

// Generate a simple topology HTML visualization
function generateSimpleTopologyHTML(data) {
    // Find gateway in ARP entries for better visualization
    let gatewayMac = '';
    if (data.arpEntries && data.arpEntries.length > 0) {
        const gatewayEntry = data.arpEntries.find(entry => entry.ipAddress === data.gateway);
        if (gatewayEntry) {
            gatewayMac = gatewayEntry.macAddress;
        }
    }

    // Display neighboring devices from ARP table
    let neighborDevices = '';
    if (data.arpEntries && data.arpEntries.length > 0) {
        // Limit to 5 neighbors for visual clarity
        const neighbors = data.arpEntries.filter(entry => entry.ipAddress !== data.gateway).slice(0, 5);
        
        if (neighbors.length > 0) {
            neighbors.forEach(neighbor => {
                neighborDevices += `
                    <div class="topology-device neighbor">
                        <i class="bi bi-pc"></i>
                        <div>${neighbor.ipAddress}</div>
                        <div class="small text-muted">${neighbor.macAddress}</div>
                    </div>
                `;
            });
        }
    }

    return `
        <div class="topology-map">
            <div class="topology-internet">
                <i class="bi bi-globe"></i>
                <div>Internet</div>
            </div>
            <div class="topology-line"></div>
            <div class="topology-router">
                <i class="bi bi-router"></i>
                <div>Router (${data.gateway})</div>
                ${gatewayMac ? `<div class="small text-muted">${gatewayMac}</div>` : ''}
            </div>
            <div class="topology-line"></div>
            <div class="topology-device active">
                <i class="bi bi-${data.ssid ? 'laptop' : 'pc-display'}"></i>
                <div>NetScout-Go (${data.ipv4Address})</div>
                <div class="small text-muted">${data.ethernetInfo.macAddress}</div>
            </div>
            ${data.ssid ? `
            <div class="topology-wifi-indicator">
                <i class="bi bi-wifi"></i>
                <div>${data.ssid}</div>
            </div>
            ` : ''}
            
            ${neighborDevices ? `
            <div class="topology-neighbors">
                <div class="small text-muted mb-2">Other Devices on Network:</div>
                ${neighborDevices}
            </div>
            ` : ''}
        </div>
    `;
}

// Add traffic data point to chart
function addTrafficDataPoint(data) {
    const now = new Date();
    const timeLabel = now.toLocaleTimeString();
    
    // Safe check if traffic data exists
    if (data.traffic && typeof data.traffic.currentBandwidth !== 'undefined') {
        // Calculate download/upload split (for demonstration - in real app would come from actual measurements)
        // This is just for visual effect on the chart since we only have total bandwidth
        const downloadBandwidth = data.traffic.currentBandwidth * 0.7; // 70% of bandwidth
        const uploadBandwidth = data.traffic.currentBandwidth * 0.3;   // 30% of bandwidth
        
        // Add data to traffic history
        addTrafficData(timeLabel, downloadBandwidth, uploadBandwidth);
    } else {
        // Add zero values if no data is available
        addTrafficData(timeLabel, 0, 0);
    }
}

// Update timestamp
function updateTimestamp(data) {
    const lastUpdatedElement = document.getElementById('lastUpdated');
    if (lastUpdatedElement) {
        const timestamp = new Date(data.timestamp);
        lastUpdatedElement.textContent = timestamp.toLocaleString();
    }
}

// Function to run a bandwidth test
function runSpeedTest() {
    if (speedTestInProgress) return; // Don't run if test is already in progress
    
    // Set the test to in progress
    speedTestInProgress = true;
    
    // Update UI to show test is running
    const bandwidthElement = document.getElementById('bandwidth');
    if (bandwidthElement) {
        bandwidthElement.innerHTML = '<span class="spinner-border spinner-border-sm" role="status"></span> Testing...';
    }
    
    // Add status indicator to bandwidth row
    const bandwidthRow = document.querySelector('.row:has(#bandwidth)');
    if (bandwidthRow) {
        // Check if status badge already exists
        let statusBadge = document.getElementById('speedTestStatus');
        if (!statusBadge) {
            // Create a new status badge if it doesn't exist
            statusBadge = document.createElement('div');
            statusBadge.id = 'speedTestStatus';
            statusBadge.className = 'badge bg-warning ms-2';
            statusBadge.style.verticalAlign = 'middle';
            bandwidthRow.querySelector('.col-6:first-child').appendChild(statusBadge);
        }
        statusBadge.innerHTML = 'Testing...';
        statusBadge.className = 'badge bg-warning ms-2';
    }
    
    // Disable the speed test button and show testing status
    const speedTestButton = document.getElementById('runSpeedTestBtn');
    if (speedTestButton) {
        speedTestButton.disabled = true;
        speedTestButton.innerHTML = '<span class="spinner-border spinner-border-sm" role="status"></span> Testing...';
    }
    
    // Add a testing indicator to the Connection Metrics card
    const cardFooter = document.querySelector('.card:has(#bandwidth) .card-footer .small.text-muted');
    if (cardFooter) {
        cardFooter.innerHTML = `<i class="bi bi-arrow-repeat spin"></i> Speed test running...`;
    }
    
    console.log("Running automatic bandwidth test...");
    
    // Call the bandwidth_test plugin API
    const requestBody = {
        id: 'bandwidth_test',
        params: {
            server: 'auto',
            duration: 10
        }
    };
    
    console.log('Sending speed test request:', requestBody);
    
    fetch('/api/run-plugin', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestBody)
    })
    .then(response => {
        if (!response.ok) {
            // Log more detailed information about the error
            console.error(`API error: ${response.status} ${response.statusText}`);
            // Try to get more information from the response body
            return response.text().then(text => {
                try {
                    // Try to parse as JSON
                    const errorData = JSON.parse(text);
                    throw new Error(`Server error: ${errorData.error || response.status}`);
                } catch (parseError) {
                    // If it's not JSON, just use the text
                    throw new Error(`Network response was not ok: ${response.status} - ${text || response.statusText}`);
                }
            });
        }
        return response.json();
    })
    .then(data => {
        console.log('Speed test response:', data);
        
        // Check if we have the expected data structure
        if (!data.downloadSpeed && !data.downloadSpeed !== 0) {
            console.warn('Speed test response missing downloadSpeed:', data);
        }
        
        // Store the test results
        lastSpeedTest = {
            timestamp: new Date(),
            downloadSpeed: data.downloadSpeed || 0,
            uploadSpeed: data.uploadSpeed || 0,
            latency: data.latency || 0
        };
        
        // Update UI with test results
        updateSpeedTestResults(data);
        
        // Set speed test to not in progress
        speedTestInProgress = false;
        
        console.log("Speed test complete:", data);
    })
    .catch(error => {
        console.error('Error running speed test:', error);
        
        // Reset UI
        if (bandwidthElement) {
            bandwidthElement.innerHTML = '<span class="text-danger">Test failed</span>';
            setTimeout(() => {
                bandwidthElement.textContent = '-- Mbps';
            }, 3000);
        }
        
        // Update status indicator to show failure
        const statusBadge = document.getElementById('speedTestStatus');
        if (statusBadge) {
            statusBadge.className = 'speed-test-status badge bg-danger position-absolute end-0 top-0 mt-1 me-1';
            statusBadge.innerHTML = '<i class="bi bi-x-circle"></i> Failed';
            
            // Fade out the status badge after 3 seconds
            setTimeout(() => {
                if (statusBadge) {
                    statusBadge.style.opacity = '0';
                    statusBadge.style.transition = 'opacity 1s';
                    setTimeout(() => {
                        if (statusBadge) {
                            statusBadge.style.display = 'none';
                        }
                    }, 1000);
                }
            }, 3000);
        }
        
        // Update the footer to indicate test failure
        const cardFooter = document.querySelector('.card:has(#bandwidth) .card-footer .small.text-muted');
        if (cardFooter) {
            cardFooter.innerHTML = '<i class="bi bi-info-circle"></i> Performance metrics <span class="ms-1 text-danger">(Test failed)</span>';
        }
        
        // Re-enable the button if present
        const speedTestButton = document.getElementById('runSpeedTestBtn');
        if (speedTestButton) {
            speedTestButton.disabled = false;
            speedTestButton.innerHTML = '<i class="bi bi-speedometer2"></i> Run Speed Test';
        }
        
        // Set speed test to not in progress
        speedTestInProgress = false;
    });
}

// Function to update UI with speed test results
function updateSpeedTestResults(data) {
    // Update bandwidth display
    const bandwidthElement = document.getElementById('bandwidth');
    if (bandwidthElement && data.downloadSpeed) {
        bandwidthElement.textContent = `${data.downloadSpeed.toFixed(2)} Mbps`;
    }
    
    // If we have connection metrics, update latency too
    const latencyElement = document.getElementById('latency');
    if (latencyElement && data.latency) {
        latencyElement.textContent = `${data.latency} ms`;
    }
    
    // Update packet loss if available
    const packetLossElement = document.getElementById('packetLoss');
    if (packetLossElement && data.packetLoss !== undefined) {
        packetLossElement.textContent = `${data.packetLoss}%`;
    }
    
    // Hide or update status indicator
    const statusBadge = document.getElementById('speedTestStatus');
    if (statusBadge) {
        statusBadge.innerHTML = 'Complete';
        statusBadge.className = 'badge bg-success ms-2';
        
        // Fade out the badge after 5 seconds
        setTimeout(() => {
            statusBadge.style.transition = 'opacity 1s';
            statusBadge.style.opacity = '0';
            // Remove the badge after fade out
            setTimeout(() => {
                statusBadge.remove();
            }, 1000);
        }, 5000);
    }
    
    // Update the footer with last test time in the Connection Metrics card
    const cardFooter = document.querySelector('.card:has(#bandwidth) .card-footer .small.text-muted');
    if (cardFooter) {
        const timestamp = new Date().toLocaleTimeString();
        cardFooter.innerHTML = `<i class="bi bi-info-circle"></i> Performance metrics <span class="ms-1">(Last test: ${timestamp})</span>`;
    }
    
    // Reset the speed test button
    const speedTestButton = document.getElementById('runSpeedTestBtn');
    if (speedTestButton) {
        speedTestButton.disabled = false;
        speedTestButton.innerHTML = '<i class="bi bi-speedometer2"></i> Run Speed Test';
    }
}

// Check connection status and run speed test if appropriate
function checkAndRunSpeedTest(event) {
    // If the connection just became active and we haven't run a test recently
    if (event.detail.status === 'connected') {
        const now = new Date();
        // Check if we've switched from disconnected or limited to connected
        const connectionChanged = event.detail.previousStatus !== 'connected';
        
        // Check if connection is Ethernet (not Wi-Fi)
        const isEthernet = lastNetworkData && !lastNetworkData.ssid;
        
        // If the connection just changed to connected and it's Ethernet, or
        // if we haven't run a test before, or it's been more than 30 minutes
        const shouldRunTest = (connectionChanged && isEthernet) || 
            !lastSpeedTest || 
            ((now - lastSpeedTest.timestamp) > (30 * 60 * 1000));
            
        if (shouldRunTest) {
            // Show notification that automatic test will run
            const speedTestButton = document.getElementById('runSpeedTestBtn');
            if (speedTestButton) {
                speedTestButton.innerHTML = '<i class="bi bi-clock-history"></i> Auto-test in 5s...';
                speedTestButton.disabled = true;
            }
            
            // Wait 5 seconds to ensure connection is stable
            clearTimeout(speedTestTimer);
            console.log("Scheduling speed test after connection detected" + 
                        (isEthernet ? " (Ethernet connection)" : ""));
            
            speedTestTimer = setTimeout(() => {
                runSpeedTest();
            }, 5000);
        }
    }
}

// Helper function to update element text
function updateElementText(elementId, text) {
    const element = document.getElementById(elementId);
    if (element) {
        element.textContent = text;
    } else {
        console.error(`Failed to update element: '${elementId}' not found in DOM!`);
    }
}
