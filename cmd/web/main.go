package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/javi11/postie/frontend"
	"github.com/javi11/postie/internal/backend"
	"github.com/javi11/postie/internal/config"
	"github.com/spf13/cobra"
)

// For development, serve static files from disk
// In production, these would be embedded
var frontendBuildPath = "../../frontend/build"

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now, should be configured properly
	},
}

var (
	port string
	host string
)

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	conn *websocket.Conn
	send chan []byte
	hub  *WebSocketHub
	id   string
}

// WebSocketHub manages WebSocket connections and broadcasts
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan []byte
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	mu         sync.RWMutex
}

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
	}
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client connected: %s", client.id)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("WebSocket client disconnected: %s", client.id)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					delete(h.clients, client)
					close(client.send)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// EmitEvent sends an event to all connected clients
func (h *WebSocketHub) EmitEvent(eventType string, data interface{}) {
	message := WebSocketMessage{
		Type: eventType,
		Data: data,
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling WebSocket message: %v", err)
		return
	}

	select {
	case h.broadcast <- jsonData:
	default:
		log.Printf("WebSocket broadcast channel full, dropping message")
	}
}

// readPump handles reading from the WebSocket connection
func (c *WebSocketClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

// writePump handles writing to the WebSocket connection
func (c *WebSocketClient) writePump() {
	defer func() {
		_ = c.conn.Close()
	}()

	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("WebSocket write error: %v", err)
			return
		}
	}
}

// WebEventEmitter is a function that emits events for web mode
type WebEventEmitter func(eventType string, data interface{})

type WebServer struct {
	app          *backend.App
	router       *mux.Router
	wsHub        *WebSocketHub
	eventEmitter WebEventEmitter
}

func NewWebServer() *WebServer {
	app := backend.NewApp()
	router := mux.NewRouter()
	wsHub := NewWebSocketHub()

	ws := &WebServer{
		app:    app,
		router: router,
		wsHub:  wsHub,
	}

	// Create event emitter function
	ws.eventEmitter = func(eventType string, data interface{}) {
		ws.wsHub.EmitEvent(eventType, data)
	}

	// Set web mode and event emitter
	app.SetWebMode(true)
	app.SetWebEventEmitter(ws.eventEmitter)

	ws.setupRoutes()

	// Start WebSocket hub
	go wsHub.Run()

	return ws
}

func (ws *WebServer) setupRoutes() {
	// Request logging middleware
	ws.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// CORS headers
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// API routes - register WebSocket first with explicit path
	ws.router.HandleFunc("/api/ws", ws.handleWebSocket)

	// API routes
	api := ws.router.PathPrefix("/api").Subrouter()

	// REST API endpoints
	api.HandleFunc("/status", ws.handleGetStatus).Methods("GET")
	api.HandleFunc("/config", ws.handleGetConfig).Methods("GET")
	api.HandleFunc("/config", ws.handleSaveConfig).Methods("POST")
	api.HandleFunc("/config/pending/status", ws.handleConfigPendingStatus).Methods("GET")
	api.HandleFunc("/config/pending", ws.handlePendingConfig).Methods("GET")
	api.HandleFunc("/config/pending/apply", ws.handleApplyPendingConfig).Methods("POST")
	api.HandleFunc("/config/pending/discard", ws.handleConfigPendingDiscard).Methods("POST")
	api.HandleFunc("/queue", ws.handleGetQueueItems).Methods("GET")
	api.HandleFunc("/queue", ws.handleClearQueue).Methods("DELETE")
	api.HandleFunc("/queue/add-files", ws.handleAddFilesToQueue).Methods("POST")
	api.HandleFunc("/queue/{id}", ws.handleRemoveFromQueue).Methods("DELETE")
	api.HandleFunc("/queue/{id}/retry", ws.handleRetryJob).Methods("POST")
	api.HandleFunc("/queue/{id}/cancel", ws.handleCancelJob).Methods("DELETE")
	api.HandleFunc("/queue/{id}/priority", ws.handleSetQueueItemPriority).Methods("POST")
	api.HandleFunc("/queue/stats", ws.handleGetQueueStats).Methods("GET")
	api.HandleFunc("/logs", ws.handleGetLogs).Methods("GET")
	api.HandleFunc("/upload", ws.handleUpload).Methods("POST")
	api.HandleFunc("/upload/cancel", ws.handleCancelUpload).Methods("POST")
	api.HandleFunc("/nzb/{id}/download", ws.handleDownloadNZB).Methods("GET")
	api.HandleFunc("/processor/status", ws.handleGetProcessorStatus).Methods("GET")
	api.HandleFunc("/processor/pause", ws.handlePauseProcessing).Methods("POST")
	api.HandleFunc("/processor/resume", ws.handleResumeProcessing).Methods("POST")
	api.HandleFunc("/processor/paused", ws.handleIsProcessingPaused).Methods("GET")
	api.HandleFunc("/running-jobs", ws.handleGetRunningJobs).Methods("GET")
	api.HandleFunc("/running-job-details", ws.handleGetRunningJobDetails).Methods("GET")
	api.HandleFunc("/validate-server", ws.handleValidateServer).Methods("POST")
	api.HandleFunc("/setup/complete", ws.handleSetupComplete).Methods("POST")

	// Serve static files (catch-all)
	ws.router.PathPrefix("/").Handler(ws.getStaticFileHandler())
}

func (ws *WebServer) getStaticFileHandler() http.Handler {
	// Check if we should use embedded filesystem or development path
	if _, err := os.Stat(frontendBuildPath); err == nil {
		// Development mode - serve from disk
		return http.StripPrefix("/", http.FileServer(http.Dir(frontendBuildPath)))
	}

	// Production mode - serve from embedded filesystem
	buildFS, err := frontend.GetBuildFS()
	if err != nil {
		log.Printf("Failed to get embedded filesystem: %v", err)
		// Fallback to disk if embedded fails
		return http.StripPrefix("/", http.FileServer(http.Dir(frontendBuildPath)))
	}

	return http.StripPrefix("/", http.FileServer(http.FS(buildFS)))
}

func (ws *WebServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	log.Printf("WebSocket connection established successfully")

	// Create client
	client := &WebSocketClient{
		conn: conn,
		send: make(chan []byte, 256),
		hub:  ws.wsHub,
		id:   fmt.Sprintf("client-%s", conn.RemoteAddr().String()),
	}

	// Register client
	ws.wsHub.register <- client

	// Start client pumps
	go client.writePump()
	go client.readPump()
}

func (ws *WebServer) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	status := ws.app.GetAppStatus()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

func (ws *WebServer) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	config, err := ws.app.GetConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(config)
}

func (ws *WebServer) handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	var configData config.ConfigData
	if err := json.NewDecoder(r.Body).Decode(&configData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := ws.app.SaveConfig(&configData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) handleGetQueueItems(w http.ResponseWriter, r *http.Request) {
	queueItems, err := ws.app.GetQueueItems()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(queueItems)
}

func (ws *WebServer) handleRetryJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := ws.app.RetryJob(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := ws.app.CancelJob(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) handleConfigPendingStatus(w http.ResponseWriter, r *http.Request) {
	status := ws.app.HasPendingConfigChanges()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

func (ws *WebServer) handleApplyPendingConfig(w http.ResponseWriter, r *http.Request) {
	if err := ws.app.ApplyPendingConfig(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) handleConfigPendingDiscard(w http.ResponseWriter, r *http.Request) {
	if err := ws.app.DiscardPendingConfig(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) handlePendingConfig(w http.ResponseWriter, r *http.Request) {
	pendingConfig := ws.app.GetPendingConfigStatus()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(pendingConfig)
}

func (ws *WebServer) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 100
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	logs, err := ws.app.GetLogsPaginated(limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(logs))
}

func (ws *WebServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		http.Error(w, "No files provided", http.StatusBadRequest)
		return
	}

	log.Printf("Processing %d uploaded files", len(files))

	// Emit initial upload progress
	ws.wsHub.EmitEvent("upload-progress", map[string]interface{}{
		"stage":     "saving",
		"progress":  0,
		"fileCount": len(files),
	})

	// Create temporary directory for uploaded files
	tempDir, err := os.MkdirTemp("", "postie-*")
	if err != nil {
		http.Error(w, "Failed to create temporary directory", http.StatusInternalServerError)
		return
	}

	// Save uploaded files to temporary location
	var filePaths []string
	for i, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, "Failed to open uploaded file", http.StatusInternalServerError)
			return
		}
		defer func() {
			_ = file.Close()
		}()

		// Create temporary file
		tempFilePath := filepath.Join(tempDir, fileHeader.Filename)
		tempFile, err := os.Create(tempFilePath)
		if err != nil {
			http.Error(w, "Failed to create temporary file", http.StatusInternalServerError)
			return
		}
		defer func() {
			_ = tempFile.Close()
		}()

		// Copy uploaded file content to temporary file
		if _, err := io.Copy(tempFile, file); err != nil {
			http.Error(w, "Failed to save uploaded file", http.StatusInternalServerError)
			return
		}

		filePaths = append(filePaths, tempFilePath)

		// Emit progress for file saving
		progress := float64(i+1) / float64(len(files)) * 50 // 50% for saving files
		ws.wsHub.EmitEvent("upload-progress", map[string]interface{}{
			"stage":       "saving",
			"progress":    progress,
			"currentFile": fileHeader.Filename,
			"fileCount":   len(files),
		})
	}

	// Emit progress for processing stage
	ws.wsHub.EmitEvent("upload-progress", map[string]interface{}{
		"stage":     "processing",
		"progress":  75,
		"fileCount": len(files),
	})

	if err := ws.app.HandleDroppedFiles(filePaths); err != nil {
		ws.wsHub.EmitEvent("upload-error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Emit completion
	ws.wsHub.EmitEvent("upload-complete", map[string]interface{}{
		"fileCount": len(files),
		"message":   "Files successfully processed and added to queue",
	})

	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) handleGetProcessorStatus(w http.ResponseWriter, r *http.Request) {
	status := ws.app.GetProcessorStatus()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

func (ws *WebServer) handleGetRunningJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := ws.app.GetRunningJobs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jobs)
}

func (ws *WebServer) handleGetRunningJobDetails(w http.ResponseWriter, r *http.Request) {
	jobDetails, err := ws.app.GetRunningJobsDetails()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jobDetails)
}

func (ws *WebServer) handleClearQueue(w http.ResponseWriter, r *http.Request) {
	if err := ws.app.ClearQueue(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) handleAddFilesToQueue(w http.ResponseWriter, r *http.Request) {
	if err := ws.app.AddFilesToQueue(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) handleRemoveFromQueue(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := ws.app.RemoveFromQueue(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) handleSetQueueItemPriority(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var requestBody struct {
		Priority int `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := ws.app.SetQueueItemPriority(id, requestBody.Priority); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) handleCancelUpload(w http.ResponseWriter, r *http.Request) {
	if err := ws.app.CancelUpload(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) handleDownloadNZB(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	nzbContent, err := ws.app.GetNZBContent(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set headers for file download
	w.Header().Set("Content-Type", "application/x-nzb")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.nzb\"", id))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(nzbContent)))

	// Write the NZB content
	_, _ = w.Write([]byte(nzbContent))
}

func (ws *WebServer) handleGetQueueStats(w http.ResponseWriter, r *http.Request) {
	stats, err := ws.app.GetQueueStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

func (ws *WebServer) handleValidateServer(w http.ResponseWriter, r *http.Request) {
	var serverData backend.ServerData
	if err := json.NewDecoder(r.Body).Decode(&serverData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := ws.app.ValidateNNTPServer(serverData)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (ws *WebServer) handleSetupComplete(w http.ResponseWriter, r *http.Request) {
	var wizardData backend.SetupWizardData
	if err := json.NewDecoder(r.Body).Decode(&wizardData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := ws.app.SetupWizardComplete(wizardData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) handlePauseProcessing(w http.ResponseWriter, r *http.Request) {
	if err := ws.app.PauseProcessing(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) handleResumeProcessing(w http.ResponseWriter, r *http.Request) {
	if err := ws.app.ResumeProcessing(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) handleIsProcessingPaused(w http.ResponseWriter, r *http.Request) {
	paused := ws.app.IsProcessingPaused()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"paused": paused})
}

func main() {
	Execute()
}

var rootCmd = &cobra.Command{
	Use:   "postie-web",
	Short: "Postie Web Server",
	Long: `Postie Web Server provides a web interface for uploading files.
It serves the frontend application and provides REST API endpoints for managing uploads and configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		server := NewWebServer()

		// Initialize the app
		ctx := context.Background()
		server.app.Startup(ctx)

		addr := fmt.Sprintf("%s:%s", host, port)
		log.Printf("Starting web server on %s", addr)

		if err := http.ListenAndServe(addr, server.router); err != nil {
			return fmt.Errorf("failed to start web server: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&port, "port", "p", "8080", "Port to run the web server on")
	rootCmd.PersistentFlags().StringVar(&host, "host", "0.0.0.0", "Host address to bind the web server to")

	// Allow environment variables to override defaults
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}
	if envHost := os.Getenv("HOST"); envHost != "" {
		host = envHost
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
