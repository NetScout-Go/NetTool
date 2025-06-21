package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/examples/lib/dev"
)

// BLE HTTP Proxy Service UUIDs
const (
	// Service UUID
	HttpProxyServiceUUID = "00001234-0000-1000-8000-00805f9b34fb"

	// Characteristics
	HttpRequestCharUUID  = "00001235-0000-1000-8000-00805f9b34fb"
	HttpResponseCharUUID = "00001236-0000-1000-8000-00805f9b34fb"
)

type BLEHttpClient struct {
	device        ble.Device
	client        ble.Client
	reqChar       *ble.Characteristic
	respChar      *ble.Characteristic
	nextRequestID int
}

func NewBLEHttpClient() (*BLEHttpClient, error) {
	d, err := dev.NewDevice("default")
	if err != nil {
		return nil, fmt.Errorf("can't create BLE device: %v", err)
	}

	ble.SetDefaultDevice(d)

	return &BLEHttpClient{
		device: d,
	}, nil
}

func (c *BLEHttpClient) Connect(deviceName string) error {
	// Set scan filter to find device with specified name
	filter := func(a ble.Advertisement) bool {
		return strings.Contains(a.LocalName(), deviceName)
	}

	// Scan for device
	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), 10*time.Second))
	fmt.Printf("Scanning for %s...\n", deviceName)

	client, err := ble.Connect(ctx, filter)
	if err != nil {
		return fmt.Errorf("can't connect to device: %v", err)
	}

	fmt.Printf("Connected to %s\n", client.Addr())
	c.client = client

	// Discover services
	fmt.Println("Discovering services...")
	services, err := client.DiscoverServices(nil)
	if err != nil {
		client.CancelConnection()
		return fmt.Errorf("can't discover services: %v", err)
	}

	// Find HTTP Proxy service
	var httpService *ble.Service
	for _, s := range services {
		if s.UUID.String() == HttpProxyServiceUUID {
			httpService = &s
			break
		}
	}

	if httpService == nil {
		client.CancelConnection()
		return fmt.Errorf("HTTP Proxy service not found")
	}

	// Discover characteristics
	chars, err := client.DiscoverCharacteristics(nil, *httpService)
	if err != nil {
		client.CancelConnection()
		return fmt.Errorf("can't discover characteristics: %v", err)
	}

	// Find request and response characteristics
	for _, char := range chars {
		switch char.UUID.String() {
		case HttpRequestCharUUID:
			c.reqChar = &char
		case HttpResponseCharUUID:
			c.respChar = &char
		}
	}

	if c.reqChar == nil || c.respChar == nil {
		client.CancelConnection()
		return fmt.Errorf("HTTP Proxy characteristics not found")
	}

	// Subscribe to notifications from response characteristic
	if err := client.Subscribe(c.respChar, false, c.handleNotification); err != nil {
		client.CancelConnection()
		return fmt.Errorf("can't subscribe to notifications: %v", err)
	}

	fmt.Println("Connected to HTTP Proxy service")
	return nil
}

func (c *BLEHttpClient) handleNotification(data []byte) {
	// Process notification data
	if len(data) < 17 {
		fmt.Println("Received invalid notification (too short)")
		return
	}

	// Extract request ID and status
	reqID := fmt.Sprintf("%x", data[0:16])
	status := data[16]

	fmt.Printf("Notification: Request ID %s, Status %d\n", reqID, status)
}

func (c *BLEHttpClient) SendHttpRequest(method, path string, headers map[string]string, body string) (string, error) {
	// Generate a unique request ID
	reqID := fmt.Sprintf("%032x", c.nextRequestID)
	c.nextRequestID++

	// Build HTTP request
	req := fmt.Sprintf("%s %s HTTP/1.1\r\n", method, path)
	for k, v := range headers {
		req += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	req += "\r\n"

	if body != "" {
		req += body
	}

	// Prepare data for BLE transfer
	reqData := []byte(req)

	// Get MTU size
	mtu := 23 // Default minimum MTU
	if c.client != nil {
		// Try to negotiate a larger MTU
		newMTU, err := c.client.ExchangeMTU(512)
		if err == nil && newMTU > mtu {
			mtu = newMTU
		}
	}

	// Max attribute data size
	maxChunkSize := mtu - 3 - 17 // MTU - ATT header - (UUID + flag)

	// Send data in chunks
	totalChunks := (len(reqData) + maxChunkSize - 1) / maxChunkSize
	for i := 0; i < totalChunks; i++ {
		start := i * maxChunkSize
		end := start + maxChunkSize
		if end > len(reqData) {
			end = len(reqData)
		}

		chunk := reqData[start:end]

		// Prepare chunk header:
		// - 16 bytes: request ID
		// - 1 byte: flags
		//   0x01 = new request
		//   0x02 = final chunk
		//   0x03 = new request and final chunk
		var flag byte = 0
		if i == 0 {
			flag |= 0x01 // New request
		}
		if i == totalChunks-1 {
			flag |= 0x02 // Final chunk
		}

		// Convert request ID to bytes
		idBytes := make([]byte, 16)
		fmt.Sscanf(reqID, "%x", &idBytes)

		// Combine header and chunk
		data := append(idBytes, flag)
		data = append(data, chunk...)

		// Write to request characteristic
		if err := c.client.WriteCharacteristic(c.reqChar, data, true); err != nil {
			return "", fmt.Errorf("failed to write request chunk: %v", err)
		}

		// Slight delay to prevent BLE buffer overflow
		time.Sleep(20 * time.Millisecond)
	}

	fmt.Printf("Sent HTTP request: %s %s\n", method, path)

	// Read response
	var response []byte
	offset := uint16(0)

	// Set timeout for reading the full response
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("timeout waiting for response")
		default:
			// Read response characteristic with offset
			data, err := c.client.ReadLongCharacteristic(c.respChar, offset)
			if err != nil {
				return "", fmt.Errorf("failed to read response: %v", err)
			}

			if len(data) == 0 {
				// End of response
				break
			}

			response = append(response, data...)
			offset += uint16(len(data))

			// Short delay to prevent BLE buffer overflow
			time.Sleep(20 * time.Millisecond)
		}
	}

	return string(response), nil
}

func (c *BLEHttpClient) Close() {
	if c.client != nil {
		c.client.CancelConnection()
	}
	if c.device != nil {
		c.device.Stop()
	}
}

// Simple HTTP over BLE proxy handler
func (c *BLEHttpClient) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract path from request
	path := r.URL.Path
	if r.URL.RawQuery != "" {
		path += "?" + r.URL.RawQuery
	}

	// Convert headers
	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	// Read body if present
	var body string
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil && len(bodyBytes) > 0 {
			body = string(bodyBytes)
		}
	}

	// Send request over BLE
	response, err := c.SendHttpRequest(r.Method, path, headers, body)
	if err != nil {
		http.Error(w, fmt.Sprintf("BLE request failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse HTTP response
	parts := strings.SplitN(response, "\r\n\r\n", 2)
	if len(parts) != 2 {
		http.Error(w, "Invalid response format", http.StatusInternalServerError)
		return
	}

	headerSection := parts[0]
	responseBody := parts[1]

	// Parse status line and headers
	headerLines := strings.Split(headerSection, "\r\n")
	if len(headerLines) < 1 {
		http.Error(w, "Invalid response format", http.StatusInternalServerError)
		return
	}

	// Parse status line
	statusLine := headerLines[0]
	statusParts := strings.SplitN(statusLine, " ", 3)
	if len(statusParts) < 3 {
		http.Error(w, "Invalid status line", http.StatusInternalServerError)
		return
	}

	statusCode := 200
	fmt.Sscanf(statusParts[1], "%d", &statusCode)

	// Parse headers
	for i := 1; i < len(headerLines); i++ {
		headerLine := headerLines[i]
		headerParts := strings.SplitN(headerLine, ":", 2)
		if len(headerParts) == 2 {
			key := strings.TrimSpace(headerParts[0])
			value := strings.TrimSpace(headerParts[1])
			w.Header().Add(key, value)
		}
	}

	// Write status code and body
	w.WriteHeader(statusCode)
	w.Write([]byte(responseBody))
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ble_http_client <device_name>")
		os.Exit(1)
	}

	deviceName := os.Args[1]

	client, err := NewBLEHttpClient()
	if err != nil {
		fmt.Printf("Error creating BLE client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	err = client.Connect(deviceName)
	if err != nil {
		fmt.Printf("Error connecting to device: %v\n", err)
		os.Exit(1)
	}

	// Start local HTTP server
	http.Handle("/", client)

	fmt.Println("Starting HTTP server on localhost:8000")
	fmt.Println("Access NetTool through http://localhost:8000/")
	fmt.Println("Press Ctrl+C to quit")

	err = http.ListenAndServe(":8000", nil)
	if err != nil {
		fmt.Printf("HTTP server error: %v\n", err)
	}
}
