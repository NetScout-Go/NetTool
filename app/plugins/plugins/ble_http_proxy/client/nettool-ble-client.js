/**
 * NetTool BLE HTTP Proxy Client Library
 * 
 * This JavaScript library provides a client-side implementation for connecting to 
 * a NetTool device via Bluetooth Low Energy and proxying HTTP requests through it.
 * 
 * Usage:
 * 1. Include this file in your web application
 * 2. Create a new NetToolBLEClient instance
 * 3. Call connect() to connect to a NetTool device
 * 4. Use the fetch() method to make HTTP requests through the BLE connection
 */

class NetToolBLEClient {
    constructor() {
        // BLE Service and Characteristic UUIDs
        this.SERVICE_UUID = '00001234-0000-1000-8000-00805f9b34fb';
        this.REQUEST_CHAR_UUID = '00001235-0000-1000-8000-00805f9b34fb';
        this.RESPONSE_CHAR_UUID = '00001236-0000-1000-8000-00805f9b34fb';
        
        // Internal state
        this.device = null;
        this.server = null;
        this.service = null;
        this.requestChar = null;
        this.responseChar = null;
        this.connected = false;
        this.pendingRequests = new Map();
        
        // Maximum size for BLE packets (MTU - 3)
        this.maxPacketSize = 509;
    }
    
    /**
     * Connect to a NetTool device via Bluetooth LE
     * @param {Object} options - Connection options
     * @param {string} options.deviceName - Optional device name filter
     * @param {Function} options.onDisconnect - Callback for disconnection events
     * @returns {Promise} - Resolves when connected
     */
    async connect(options = {}) {
        if (!navigator.bluetooth) {
            throw new Error('Web Bluetooth API is not available in this browser.');
        }
        
        const deviceName = options.deviceName || 'NetTool';
        
        try {
            // Request the device
            this.device = await navigator.bluetooth.requestDevice({
                filters: [
                    { name: deviceName },
                    { services: [this.SERVICE_UUID] }
                ]
            });
            
            // Setup disconnect listener
            this.device.addEventListener('gattserverdisconnected', () => {
                this.connected = false;
                if (options.onDisconnect) {
                    options.onDisconnect();
                }
            });
            
            // Connect to the GATT server
            this.server = await this.device.gatt.connect();
            
            // Get the HTTP Proxy service
            this.service = await this.server.getPrimaryService(this.SERVICE_UUID);
            
            // Get the characteristics
            this.requestChar = await this.service.getCharacteristic(this.REQUEST_CHAR_UUID);
            this.responseChar = await this.service.getCharacteristic(this.RESPONSE_CHAR_UUID);
            
            // Set up notifications for the response characteristic
            await this.responseChar.startNotifications();
            this.responseChar.addEventListener('characteristicvaluechanged', 
                this._handleResponseNotification.bind(this));
            
            this.connected = true;
            return true;
        } catch (error) {
            console.error('Connection error:', error);
            throw error;
        }
    }
    
    /**
     * Disconnect from the NetTool device
     */
    disconnect() {
        if (this.device && this.device.gatt.connected) {
            this.device.gatt.disconnect();
        }
        this.connected = false;
    }
    
    /**
     * Check if connected to a NetTool device
     * @returns {boolean} - True if connected
     */
    isConnected() {
        return this.connected && 
               this.device && 
               this.device.gatt.connected;
    }
    
    /**
     * Make an HTTP request through the BLE connection
     * @param {string} url - The URL to fetch
     * @param {Object} options - Fetch options (similar to fetch API)
     * @returns {Promise} - Resolves with the response
     */
    async fetch(url, options = {}) {
        if (!this.isConnected()) {
            throw new Error('Not connected to a NetTool device');
        }
        
        // Create a unique request ID
        const requestId = this._generateRequestId();
        
        // Build the HTTP request
        let httpRequest = `${options.method || 'GET'} ${url} HTTP/1.1\r\n`;
        
        // Add headers
        const headers = options.headers || {};
        for (const [key, value] of Object.entries(headers)) {
            httpRequest += `${key}: ${value}\r\n`;
        }
        
        // Add body if present
        let body = '';
        if (options.body) {
            body = typeof options.body === 'string' 
                ? options.body 
                : JSON.stringify(options.body);
            
            httpRequest += `Content-Length: ${body.length}\r\n`;
        }
        
        // Finish headers
        httpRequest += '\r\n';
        
        // Add body
        if (body) {
            httpRequest += body;
        }
        
        // Create a promise that will resolve when we get a response
        const responsePromise = new Promise((resolve, reject) => {
            this.pendingRequests.set(requestId, { resolve, reject });
            
            // Set a timeout to reject the promise if we don't get a response
            setTimeout(() => {
                if (this.pendingRequests.has(requestId)) {
                    this.pendingRequests.delete(requestId);
                    reject(new Error('Request timed out'));
                }
            }, 30000); // 30 second timeout
        });
        
        // Send the request
        await this._sendHttpRequest(requestId, httpRequest);
        
        // Wait for the response
        return responsePromise;
    }
    
    /**
     * Generate a unique request ID
     * @returns {string} - A unique ID
     */
    _generateRequestId() {
        return 'req_' + Math.random().toString(36).substr(2, 9);
    }
    
    /**
     * Send an HTTP request over BLE
     * @private
     * @param {string} requestId - The request ID
     * @param {string} httpRequest - The HTTP request string
     */
    async _sendHttpRequest(requestId, httpRequest) {
        // Convert the request string to bytes
        const encoder = new TextEncoder();
        const requestBytes = encoder.encode(httpRequest);
        
        // Convert the request ID to bytes
        const requestIdBytes = encoder.encode(requestId);
        
        // Calculate how many chunks we need to send
        const maxDataSize = this.maxPacketSize - 17; // 16 bytes for ID + 1 byte for flags
        const numChunks = Math.ceil(requestBytes.length / maxDataSize);
        
        // Send the request in chunks
        for (let i = 0; i < numChunks; i++) {
            // Calculate the start and end positions for this chunk
            const start = i * maxDataSize;
            const end = Math.min(start + maxDataSize, requestBytes.length);
            
            // Create a buffer for this chunk
            const chunk = new Uint8Array(17 + (end - start));
            
            // Add the request ID (first 16 bytes)
            chunk.set(requestIdBytes.slice(0, 16), 0);
            
            // Add the chunk flag (1 byte)
            // 1 = first chunk, 2 = last chunk, 3 = first and last (single chunk), 0 = middle chunk
            let flag = 0;
            if (i === 0) flag |= 1;
            if (i === numChunks - 1) flag |= 2;
            chunk[16] = flag;
            
            // Add the data
            chunk.set(requestBytes.slice(start, end), 17);
            
            // Send the chunk
            await this.requestChar.writeValue(chunk);
        }
    }
    
    /**
     * Handle a notification from the response characteristic
     * @private
     * @param {Event} event - The notification event
     */
    _handleResponseNotification(event) {
        const value = event.target.value;
        if (!value || value.byteLength < 17) {
            console.error('Invalid response format');
            return;
        }
        
        // Extract the request ID
        const requestIdBytes = new Uint8Array(value.buffer, 0, 16);
        const decoder = new TextDecoder();
        const requestId = decoder.decode(requestIdBytes);
        
        // Extract the chunk flag
        const flag = value.getUint8(16);
        
        // Extract the data
        const dataBytes = new Uint8Array(value.buffer, 17);
        
        // Find the pending request
        const request = this.pendingRequests.get(requestId);
        if (!request) {
            console.warn(`No pending request found for ID: ${requestId}`);
            return;
        }
        
        // Process the response based on the flag
        if (flag & 2) { // Last chunk or only chunk
            // This is the last or only chunk, resolve the promise
            this.pendingRequests.delete(requestId);
            request.resolve(decoder.decode(dataBytes));
        } else {
            // This is not the last chunk, we need to buffer the data
            if (!request.buffer) {
                request.buffer = dataBytes;
            } else {
                // Concatenate the buffers
                const newBuffer = new Uint8Array(request.buffer.length + dataBytes.length);
                newBuffer.set(request.buffer, 0);
                newBuffer.set(dataBytes, request.buffer.length);
                request.buffer = newBuffer;
            }
        }
    }
}

// Export the class for use in modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = NetToolBLEClient;
}

// Also make it available globally
if (typeof window !== 'undefined') {
    window.NetToolBLEClient = NetToolBLEClient;
}
