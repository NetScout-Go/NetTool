package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/anoam/netscout-pi/app/core"
	"github.com/anoam/netscout-pi/app/plugins"
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

	return r
}

func main() {
	// Parse command line flags
	port := flag.Int("port", 8080, "Port to run the server on")
	flag.Parse()

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

	// Serve static files
	r.Static("/static", "./app/static")

	// Main dashboard route
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "dashboard.html", gin.H{
			"title":   "NetScout-Pi Dashboard",
			"plugins": pluginManager.GetPlugins(),
		})
	})

	// Additional explicit dashboard route
	r.GET("/dashboard", func(c *gin.Context) {
		c.HTML(http.StatusOK, "dashboard.html", gin.H{
			"title":   "NetScout-Pi Dashboard",
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
	}

	// WebSocket for real-time updates
	r.GET("/ws", func(c *gin.Context) {
		handleWebSocketConnection(c.Writer, c.Request)
	})

	// Start the server
	log.Printf("Starting NetScout-Pi server on :%d", *port)
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
