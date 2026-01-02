// Package main is the entry point for the NetTool application.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/NetScout-Go/NetTool/app/core"
	"github.com/NetScout-Go/NetTool/app/plugins"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Build-time variables (set via ldflags during build)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(_ *http.Request) bool {
		return true // Allow all connections for development
	},
}

// Frontend directory for the React SPA
const frontendDir = "frontend/dist"

func main() {
	// Parse command line flags
	port := flag.Int("port", 8080, "Port to run the server on")
	version := flag.Bool("version", false, "Show version information")
	showIntegrity := flag.Bool("integrity", false, "Show integrity verification status and exit")
	flag.Parse()

	// Show version if requested
	if *version {
		fmt.Printf("NetTool %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		os.Exit(0)
	}

	// MANDATORY: Perform integrity verification - cannot be skipped
	performIntegrityCheck(*showIntegrity)

	// If only showing integrity status, exit
	if *showIntegrity {
		os.Exit(0)
	}

	// Apply runtime protection measures
	core.RuntimeProtection()

	// Perform comprehensive security checks
	securityResult := core.PerformSecurityChecks()
	if !securityResult.Passed {
		log.Println("⚠️  Security check warnings detected")
		for _, anomaly := range securityResult.Anomalies {
			log.Printf("   - %s", anomaly)
		}
	}

	// Perform anti-tamper checks
	if !core.PerformAntiTamperChecks() {
		log.Println("⚠️  Anti-tamper check warning: unusual runtime environment detected")
	}

	// Print startup banner
	fmt.Printf("🌐 NetTool %s starting...\n", Version)
	fmt.Printf("📅 Built: %s (commit: %s)\n", BuildTime, GitCommit)

	// Check if frontend build exists
	if _, err := os.Stat(frontendDir); os.IsNotExist(err) {
		log.Printf("⚠️  Frontend not built. Run 'cd frontend && npm run build' first.")
	} else {
		log.Printf("✅ React frontend found at %s", frontendDir)
	}

	// Ensure plugin directories exist
	os.MkdirAll("app/plugins/plugins", 0700)

	// Set Gin to release mode for cleaner output
	gin.SetMode(gin.ReleaseMode)

	// Initialize the router
	r := gin.Default()

	// Add security headers middleware
	r.Use(func(c *gin.Context) {
		// Prevent clickjacking attacks
		c.Header("X-Frame-Options", "DENY")
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")
		// Enable XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")
		// Strict Transport Security (only if HTTPS is used)
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		// Prevent referrer leaking
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		// Content Security Policy - Updated for React SPA
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; img-src 'self' data: https:; font-src 'self' data: https://fonts.gstatic.com; connect-src 'self' ws: wss:")
		c.Next()
	})

	// Start network info broadcaster in the background
	go startNetworkInfoBroadcaster()

	// Initialize plugin manager
	pluginManager := plugins.NewPluginManager()

	// Register plugins - our new implementation handles both modular and hardcoded plugins
	pluginManager.RegisterPlugins()

	// Initialize plugin installer
	pluginInstaller := plugins.NewPluginInstaller("app/plugins/plugins", pluginManager)

	// GitHub API configuration tip
	log.Println("💡 TIP: To avoid GitHub API rate limits, add a personal access token to app/plugins/config.json")
	log.Println("   Instructions: https://github.com/settings/tokens (generate token with 'public_repo' scope)")

	// API endpoints - registered BEFORE the SPA catch-all
	api := r.Group("/api")
	{
		// Get all plugins
		api.GET("/plugins", func(c *gin.Context) {
			c.JSON(http.StatusOK, pluginManager.GetPlugins())
		})

		// Get specific plugin info
		api.GET("/plugins/:id", func(c *gin.Context) {
			pluginID := c.Param("id")
			plugin, err := pluginManager.GetPlugin(pluginID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, plugin)
		})

		// Run a plugin
		api.POST("/plugins/:id/run", func(c *gin.Context) {
			pluginID := c.Param("id")
			var params map[string]interface{}
			if err := c.BindJSON(&params); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			result, err := pluginManager.RunPlugin(pluginID, params)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, result)
		})

		// Get network information for the dashboard
		api.GET("/network-info", func(c *gin.Context) {
			networkInfo, err := core.GetNetworkInfo()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// Update the timestamp to current time
			networkInfo.Timestamp = time.Now()

			c.JSON(http.StatusOK, networkInfo)
		})

		// Get binary integrity status
		api.GET("/integrity", func(c *gin.Context) {
			status := core.GetCachedIntegrityStatus()
			if status == nil {
				status = core.VerifyBinaryIntegrity()
			}
			c.JSON(http.StatusOK, status)
		})

		// Get binary information
		api.GET("/binary-info", func(c *gin.Context) {
			info := core.GetBinaryInfo()
			info["version"] = Version
			info["build_time"] = BuildTime
			info["git_commit"] = GitCommit
			c.JSON(http.StatusOK, info)
		})

		// Get security status
		api.GET("/security", func(c *gin.Context) {
			result := core.PerformSecurityChecks()
			c.JSON(http.StatusOK, result)
		})

		// Get network interfaces for auto-detection
		api.GET("/interfaces", func(c *gin.Context) {
			interfaces, err := core.GetInterfaces()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, interfaces)
		})

		// Get primary WiFi interface
		api.GET("/interfaces/wifi", func(c *gin.Context) {
			iface, err := core.GetPrimaryWiFiInterface()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if iface == nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "No WiFi interface found"})
				return
			}
			c.JSON(http.StatusOK, iface)
		})

		// Get primary Ethernet interface
		api.GET("/interfaces/ethernet", func(c *gin.Context) {
			iface, err := core.GetPrimaryEthernetInterface()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if iface == nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "No Ethernet interface found"})
				return
			}
			c.JSON(http.StatusOK, iface)
		})

		// Get interface by type
		api.GET("/interfaces/type/:type", func(c *gin.Context) {
			ifaceType := c.Param("type")
			var t core.InterfaceType
			switch ifaceType {
			case "ethernet":
				t = core.InterfaceTypeEthernet
			case "wifi":
				t = core.InterfaceTypeWiFi
			case "loopback":
				t = core.InterfaceTypeLoopback
			case "virtual":
				t = core.InterfaceTypeVirtual
			case "bridge":
				t = core.InterfaceTypeBridge
			case "vpn":
				t = core.InterfaceTypeVPN
			default:
				t = core.InterfaceTypeUnknown
			}
			interfaces, err := core.GetInterfacesByType(t)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, interfaces)
		})

		// General plugin runner endpoint for dashboard features
		api.POST("/run-plugin", func(c *gin.Context) {
			var request struct {
				ID     string                 `json:"id"`
				Params map[string]interface{} `json:"params"`
			}

			if err := c.BindJSON(&request); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			result, err := pluginManager.RunPlugin(request.ID, request.Params)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, result)
		})

		// Plugin Manager API endpoints
		pluginManage := api.Group("/plugins/manage")
		{
			// List all installed plugins
			pluginManage.GET("/list", func(c *gin.Context) {
				plugins, err := pluginInstaller.ListInstalledPlugins()
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, plugins)
			})

			// Get plugin details
			pluginManage.GET("/details/:id", func(c *gin.Context) {
				pluginID := c.Param("id")
				details, err := pluginInstaller.GetPluginDetails(pluginID)
				if err != nil {
					c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, details)
			})

			// List available plugins from GitHub and local catalog
			pluginManage.GET("/available", func(c *gin.Context) {
				response, err := pluginInstaller.ListAvailablePluginsWithMeta(false)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, response)
			})

			// Refresh plugin catalog from GitHub
			pluginManage.POST("/refresh-catalog", func(c *gin.Context) {
				response, err := pluginInstaller.RefreshPluginCatalog()
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, response)
			})

			// Install plugin from repository
			pluginManage.POST("/install", func(c *gin.Context) {
				var request struct {
					Repository string `json:"repository"`
				}

				if err := c.BindJSON(&request); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				err := pluginInstaller.InstallPluginFromRepository(request.Repository)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				c.JSON(http.StatusOK, gin.H{"message": "Plugin installed successfully"})
			})

			// Bulk install plugins from repositories
			pluginManage.POST("/bulk-install", func(c *gin.Context) {
				var request struct {
					Repositories []string `json:"repositories"`
				}

				if err := c.BindJSON(&request); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				if len(request.Repositories) == 0 {
					c.JSON(http.StatusBadRequest, gin.H{"error": "No repositories provided"})
					return
				}

				result := pluginInstaller.BulkInstallPlugins(request.Repositories)
				c.JSON(http.StatusOK, result)
			})

			// Upload plugin (ZIP file)
			pluginManage.POST("/upload", func(c *gin.Context) {
				file, _, err := c.Request.FormFile("plugin")
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "No plugin file uploaded"})
					return
				}
				defer file.Close()

				metadata, err := pluginInstaller.UploadPlugin(file)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				c.JSON(http.StatusOK, metadata)
			})

			// Update plugin
			pluginManage.POST("/update/:id", func(c *gin.Context) {
				pluginID := c.Param("id")
				metadata, err := pluginInstaller.UpdatePlugin(pluginID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				c.JSON(http.StatusOK, metadata)
			})

			// Uninstall plugin
			pluginManage.POST("/uninstall/:id", func(c *gin.Context) {
				pluginID := c.Param("id")
				metadata, err := pluginInstaller.UninstallPlugin(pluginID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				c.JSON(http.StatusOK, metadata)
			})

			// Check if a file exists
			pluginManage.GET("/file-exists", func(c *gin.Context) {
				filePath := c.Query("path")
				if filePath == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Path parameter is required"})
					return
				}

				_, err := os.Stat(filePath)
				exists := !os.IsNotExist(err)

				c.JSON(http.StatusOK, gin.H{"exists": exists})
			})

			// View file contents (for README files, etc.)
			pluginManage.GET("/view-file", func(c *gin.Context) {
				filePath := c.Query("path")
				if filePath == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Path parameter is required"})
					return
				}

				// Ensure the file exists
				_, err := os.Stat(filePath)
				if os.IsNotExist(err) {
					c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
					return
				}

				// Read the file
				content, err := ioutil.ReadFile(filePath)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
					return
				}

				// Check file extension to determine content type
				extension := strings.ToLower(filepath.Ext(filePath))

				switch extension {
				case ".md":
					// For markdown files, render as HTML
					c.Header("Content-Type", "text/html")

					// Very simple markdown to HTML conversion
					htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; line-height: 1.6; padding: 20px; max-width: 900px; margin: 0 auto; }
        pre { background: #f5f5f5; padding: 10px; border-radius: 5px; overflow-x: auto; }
        code { background: #f5f5f5; padding: 2px 4px; border-radius: 3px; }
        h1, h2, h3 { margin-top: 24px; }
        a { color: #0366d6; }
        img { max-width: 100%%; }
    </style>
</head>
<body>
    <h1>%s</h1>
    <div>%s</div>
</body>
</html>`, filepath.Base(filePath), filepath.Base(filePath), string(content))

					c.String(http.StatusOK, htmlContent)
				default:
					// For other files, just return the content as plain text
					c.String(http.StatusOK, string(content))
				}
			})

			// Sync with repository
			pluginManage.POST("/sync", func(c *gin.Context) {
				// This endpoint will check for updates from GitHub for all plugins
				// and update the plugin.json files with version information

				plugins, err := pluginInstaller.ListInstalledPlugins()
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				// Count how many plugins were updated
				updated := 0

				// For each plugin, check for updates and update version info
				for _, plugin := range plugins {
					// Update the plugin.json with the latest version info
					err := pluginInstaller.UpdateVersionInfo(plugin.ID)
					if err == nil {
						updated++
					}
				}

				c.JSON(http.StatusOK, gin.H{
					"message": "Successfully synced with repository",
					"updated": updated,
				})
			})

			// Update all plugins
			pluginManage.POST("/update-all", func(c *gin.Context) {
				// Get all plugins
				plugins, err := pluginInstaller.ListInstalledPlugins()
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				// Count how many plugins were updated
				updated := 0

				// Update each plugin that has an update available
				for _, plugin := range plugins {
					if plugin.UpdateAvailable {
						_, err := pluginInstaller.UpdatePlugin(plugin.ID)
						if err == nil {
							updated++
						}
					}
				}

				c.JSON(http.StatusOK, gin.H{
					"message": "Successfully updated plugins",
					"updated": updated,
				})
			})
		}
	}

	// WebSocket for real-time updates
	r.GET("/ws", func(c *gin.Context) {
		handleWebSocketConnection(c.Writer, c.Request)
	})

	// Serve React SPA - this must be registered AFTER all API routes
	serveSPA(r)

	// Start the server - bind to all interfaces (0.0.0.0)
	log.Printf("🚀 Starting NetTool server on 0.0.0.0:%d", *port)
	log.Printf("📱 Access locally: http://localhost:%d", *port)
	log.Printf("🌐 Access from network: http://<your-ip>:%d", *port)
	log.Fatal(r.Run(fmt.Sprintf("0.0.0.0:%d", *port)))
}

// Clients map to manage WebSocket connections
var clients = make(map[*websocket.Conn]bool)
var clientsMutex = sync.Mutex{}

func handleWebSocketConnection(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		return
	}
	defer ws.Close()

	// Register new client
	clientsMutex.Lock()
	clients[ws] = true
	clientsMutex.Unlock()

	// Remove client when connection closes
	defer func() {
		clientsMutex.Lock()
		delete(clients, ws)
		clientsMutex.Unlock()
	}()

	// No need to start individual updaters anymore
	// We're using the global broadcaster

	// Handle incoming messages (not required for this application, but included for completeness)
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}
	}
}

func sendPeriodicNetworkUpdates(ws *websocket.Conn) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			networkInfo, err := core.GetNetworkInfo()
			if err != nil {
				log.Printf("Error getting network info: %v", err)
				continue
			}

			// Check if this specific client is still connected
			clientsMutex.Lock()
			if !clients[ws] {
				clientsMutex.Unlock()
				return
			}
			clientsMutex.Unlock()

			// Send update to this client
			err = ws.WriteJSON(map[string]interface{}{
				"type":      "network_update",
				"data":      networkInfo,
				"timestamp": time.Now().Format(time.RFC3339),
			})
			if err != nil {
				log.Printf("Error sending network update: %v", err)
				return
			}
		}
	}
}

// Broadcast network information to all connected clients
func broadcastNetworkInfo() {
	networkInfo, err := core.GetNetworkInfo()
	if err != nil {
		log.Printf("Error getting network info for broadcast: %v", err)
		return
	}

	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	// Send update to all connected clients
	for client := range clients {
		err := client.WriteJSON(map[string]interface{}{
			"type":      "network_update",
			"data":      networkInfo,
			"timestamp": time.Now().Format(time.RFC3339),
		})
		if err != nil {
			log.Printf("Error sending network update to client: %v", err)
			// Consider removing the client from the clients map if needed
		}
	}
}

// startNetworkInfoBroadcaster sends network updates to all connected clients
func startNetworkInfoBroadcaster() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C

		// Only broadcast if there are clients connected
		clientsMutex.Lock()
		clientCount := len(clients)
		clientsMutex.Unlock()

		if clientCount == 0 {
			continue
		}

		// Get network info once for all clients
		networkInfo, err := core.GetNetworkInfo()
		if err != nil {
			log.Printf("Error getting network info for broadcast: %v", err)
			continue
		}

		// Set timestamp to current time
		networkInfo.Timestamp = time.Now()

		// Prepare the message once for all clients
		message := map[string]interface{}{
			"type":      "network_update",
			"data":      networkInfo,
			"timestamp": time.Now().Format(time.RFC3339),
		}

		// Broadcast to all clients
		clientsMutex.Lock()
		for client := range clients {
			// Send in a non-blocking way
			go func(c *websocket.Conn) {
				if err := c.WriteJSON(message); err != nil {
					log.Printf("Error broadcasting to client: %v", err)

					// Close and remove failed client
					c.Close()
					clientsMutex.Lock()
					delete(clients, c)
					clientsMutex.Unlock()
				}
			}(client)
		}
		clientsMutex.Unlock()
	}
}

// serveSPA configures the router to serve the React Single Page Application
// It serves static assets from frontend/dist/assets/ and falls back to index.html
// for all other non-API routes to support client-side routing
func serveSPA(r *gin.Engine) {
	// Check if the frontend build exists
	if _, err := os.Stat(frontendDir); os.IsNotExist(err) {
		log.Printf("⚠️  Frontend directory not found at %s", frontendDir)
		log.Printf("   Run 'cd frontend && npm run build' to build the frontend")

		// Fallback: serve a simple HTML page directing users to build the frontend
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/ws") {
				c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
				return
			}
			c.Header("Content-Type", "text/html")
			c.String(http.StatusOK, `<!DOCTYPE html>
<html>
<head><title>NetTool - Frontend Not Built</title></head>
<body style="font-family: sans-serif; text-align: center; padding: 50px;">
<h1>🔧 NetTool Frontend Not Built</h1>
<p>The React frontend has not been built yet.</p>
<p>Run the following commands to build it:</p>
<pre style="background: #f5f5f5; padding: 20px; display: inline-block;">
cd frontend
npm install
npm run build
</pre>
<p>Then restart the server.</p>
</body>
</html>`)
		})
		return
	}

	// Serve static assets (JS, CSS, images)
	r.Static("/assets", filepath.Join(frontendDir, "assets"))

	// Serve favicon and other root static files if they exist
	rootFiles := []string{"favicon.ico", "favicon.svg", "robots.txt", "manifest.json"}
	for _, file := range rootFiles {
		filePath := filepath.Join(frontendDir, file)
		if _, err := os.Stat(filePath); err == nil {
			localFile := file
			localPath := filePath
			r.GET("/"+localFile, func(c *gin.Context) {
				c.File(localPath)
			})
		}
	}

	// Handle all other routes by serving index.html (for client-side routing)
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// Don't serve SPA for API or WebSocket routes
		if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/ws") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Endpoint not found"})
			return
		}

		// Serve the React app's index.html for all other routes
		indexPath := filepath.Join(frontendDir, "index.html")
		c.File(indexPath)
	})

	log.Printf("✅ React SPA configured to serve from %s", frontendDir)
}

// performIntegrityCheck verifies binary integrity at startup
// If verification fails or cannot be performed, the binary will NOT run
func performIntegrityCheck(showStatus bool) {
	log.Println("🔐 Verifying binary integrity...")

	status := core.VerifyBinaryIntegrity()

	if showStatus {
		fmt.Println("\n═══════════════════════════════════════")
		fmt.Println("       NetTool Integrity Status")
		fmt.Println("═══════════════════════════════════════")
		fmt.Printf("Binary Path:    %s\n", status.BinaryPath)
		fmt.Printf("Binary Size:    %d bytes\n", status.BinarySize)
		fmt.Printf("Platform:       %s/%s\n", status.RuntimeOS, status.RuntimeArch)
		fmt.Printf("Checked At:     %s\n", status.CheckedAt)
		fmt.Printf("Source:         %s\n", status.Source)
		fmt.Println("───────────────────────────────────────")

		if status.ActualHash != "" {
			fmt.Printf("Actual Hash:    %s\n", status.ActualHash)
		}
		if status.ExpectedHash != "" {
			fmt.Printf("Expected Hash:  %s\n", status.ExpectedHash)
		}

		fmt.Println("───────────────────────────────────────")
		if status.Verified {
			fmt.Println("Status:         ✅ VERIFIED")
		} else {
			fmt.Println("Status:         ❌ VERIFICATION FAILED")
			if status.TamperDetected {
				fmt.Println("                ⚠️  POSSIBLE TAMPERING DETECTED")
			}
		}

		if status.Error != "" {
			fmt.Printf("Note:           %s\n", status.Error)
		}

		if status.ShouldBlock {
			fmt.Println("───────────────────────────────────────")
			fmt.Println("⛔ Binary will NOT run - integrity check CANNOT be skipped")
			fmt.Println("   This is a security measure to protect against tampering")
		}
		fmt.Println("═══════════════════════════════════════\n")
		return
	}

	// CRITICAL: Block execution if integrity check fails
	if status.ShouldBlock {
		switch status.Source {
		case "embedded":
			log.Println("❌ INTEGRITY CHECK FAILED - Binary modified!")
			log.Println("⚠️  The binary hash does not match the embedded hash.")
			log.Println("⚠️  This binary may have been tampered with.")
		case "github-release", "github-beta":
			log.Println("❌ INTEGRITY CHECK FAILED - Binary modified!")
			log.Println("⚠️  The binary hash does not match the official release.")
			log.Println("⚠️  This binary may have been compromised.")
		case "blocked":
			log.Println("❌ INTEGRITY CHECK FAILED - Cannot verify binary!")
			log.Println("⚠️  No trusted hash source available (embedded or GitHub).")
			log.Println("⚠️  This may be a development build or network is offline.")
		default:
			log.Println("❌ INTEGRITY CHECK FAILED")
			if status.Error != "" {
				log.Printf("⚠️  Error: %s", status.Error)
			}
		}

		log.Println("")
		log.Println("⛔ CRITICAL: This binary will NOT run.")
		log.Println("   Integrity verification is MANDATORY and cannot be bypassed.")
		log.Println("   Only official releases from GitHub are trusted.")
		os.Exit(1)
	}

	// Verification passed
	switch status.Source {
	case "embedded":
		log.Println("✅ Binary integrity verified (embedded hash)")
	case "github-release":
		log.Println("✅ Binary integrity verified (official release)")
	case "github-beta":
		log.Println("✅ Binary integrity verified (beta release)")
	case "development":
		log.Println("⚠️  Development build - integrity verification not available")
	default:
		log.Printf("✅ Binary integrity verified (source: %s)", status.Source)
	}
}
