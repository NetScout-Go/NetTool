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
	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections for development
	},
}

// createMyRender creates a multitemplate renderer for proper template inheritance
func createMyRender() multitemplate.Renderer {
	r := multitemplate.NewRenderer()

	// Load templates
	r.AddFromFiles("dashboard.html", "app/templates/layout.html", "app/templates/dashboard.html")
	r.AddFromFiles("error.html", "app/templates/layout.html", "app/templates/error.html")
	r.AddFromFiles("plugin_page.html", "app/templates/layout.html", "app/templates/plugin_page.html")
	r.AddFromFiles("plugin_manager.html", "app/templates/layout.html", "app/templates/plugin_manager.html")

	return r
}

func main() {
	// Parse command line flags
	port := flag.Int("port", 8080, "Port to run the server on")
	flag.Parse()

	// Ensure plugin directories exist
	os.MkdirAll("app/plugins/plugins", 0755)

	// Initialize the router
	r := gin.Default()

	// Start network info broadcaster in the background
	go startNetworkInfoBroadcaster()

	// Set HTML renderer
	r.HTMLRender = createMyRender()

	// Initialize plugin manager
	pluginManager := plugins.NewPluginManager()

	// Register plugins - our new implementation handles both modular and hardcoded plugins
	pluginManager.RegisterPlugins()

	// Initialize plugin installer
	pluginInstaller := plugins.NewPluginInstaller("app/plugins/plugins", pluginManager)

	// Serve static files
	r.Static("/static", "./app/static")

	// Main dashboard route
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "dashboard.html", gin.H{
			"title":   "NetTool Dashboard",
			"plugins": pluginManager.GetPlugins(),
		})
	})

	// Additional explicit dashboard route
	r.GET("/dashboard", func(c *gin.Context) {
		c.HTML(http.StatusOK, "dashboard.html", gin.H{
			"title":   "NetTool Dashboard",
			"plugins": pluginManager.GetPlugins(),
		})
	})

	// Plugin manager route
	r.GET("/plugin-manager", func(c *gin.Context) {
		c.HTML(http.StatusOK, "plugin_manager.html", gin.H{
			"title":   "Plugin Manager",
			"plugins": pluginManager.GetPlugins(),
		})
	})

	// Plugin page route
	r.GET("/plugin/:id", func(c *gin.Context) {
		pluginID := c.Param("id")
		plugin, err := pluginManager.GetPlugin(pluginID)
		if err != nil {
			c.HTML(http.StatusNotFound, "error.html", gin.H{
				"title": "Plugin Not Found",
				"error": err.Error(),
			})
			return
		}

		c.HTML(http.StatusOK, "plugin_page.html", gin.H{
			"title":   plugin.Name,
			"plugin":  plugin,
			"plugins": pluginManager.GetPlugins(),
		})
	})

	// API endpoints
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
				plugins, err := pluginInstaller.ListAvailablePlugins()
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, plugins)
			})

			// Refresh plugin catalog from GitHub
			pluginManage.POST("/refresh-catalog", func(c *gin.Context) {
				err := pluginInstaller.RefreshPluginCatalog()
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"message": "Plugin catalog refreshed successfully"})
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

	// Start the server
	log.Printf("Starting NetTool server on :%d", *port)
	log.Fatal(r.Run(fmt.Sprintf(":%d", *port)))
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
