package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"
)

// MCP Server Types
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Simple MCP Server
type MCPServer struct {
	tools map[string]Tool
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

func NewMCPServer() *MCPServer {
	return &MCPServer{
		tools: make(map[string]Tool),
	}
}

func (s *MCPServer) setupTools() {
	// Add basic tools
	s.tools["system_info"] = Tool{
		Name:        "system_info",
		Description: "Get system information",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	}
	
	s.tools["echo"] = Tool{
		Name:        "echo",
		Description: "Echo back a message",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"message": map[string]interface{}{
					"type":        "string",
					"description": "Message to echo",
				},
			},
			"required": []string{"message"},
		},
	}
}

func main() {
	server := NewMCPServer()
	server.setupTools()

	// Root handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Go MCP Server Running!",
			"version": "1.0.0",
			"endpoints": map[string]string{
				"health": "/health",
				"mcp":    "/mcp",
			},
			"timestamp": time.Now().UTC(),
		})
	})

	// Health endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "healthy",
			"server":  "Go MCP Server",
			"version": "1.0.0",
			"uptime":  time.Now().Unix(),
		})
	})

	// MCP endpoint
	http.HandleFunc("/mcp", server.handleMCP)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("🚀 Go MCP Server starting on port %s\n", port)
	fmt.Printf("📡 MCP endpoint: http://localhost:%s/mcp\n", port)
	fmt.Printf("💓 Health check: http://localhost:%s/health\n", port)
	fmt.Printf("🏠 Root endpoint: http://localhost:%s/\n", port)
	
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func (s *MCPServer) handleMCP(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle preflight
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Handle GET - return server info
	if r.Method == "GET" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"name":     "Go MCP Server",
			"version":  "1.0.0",
			"protocol": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]bool{
					"listChanged": true,
				},
			},
		})
		return
	}

	// Handle POST - JSON-RPC
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(&JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      nil,
			Error: &JSONRPCError{
				Code:    -32700,
				Message: "Parse error",
			},
		})
		return
	}

	// Handle different methods
	switch req.Method {
	case "initialize":
		json.NewEncoder(w).Encode(&JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]bool{
						"listChanged": true,
					},
				},
				"serverInfo": map[string]interface{}{
					"name":    "Go MCP Server",
					"version": "1.0.0",
				},
			},
		})

	case "tools/list":
		tools := []Tool{}
		for _, tool := range s.tools {
			tools = append(tools, tool)
		}
		json.NewEncoder(w).Encode(&JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"tools": tools,
			},
		})

	case "tools/call":
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		json.Unmarshal(req.Params, &params)

		result := s.executeTool(params.Name, params.Arguments)
		json.NewEncoder(w).Encode(&JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		})

	default:
		json.NewEncoder(w).Encode(&JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32601,
				Message: "Method not found",
			},
		})
	}
}

func (s *MCPServer) executeTool(name string, args json.RawMessage) interface{} {
	switch name {
	case "system_info":
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("OS: %s\nArch: %s\nGo Version: %s\nCPUs: %d",
						runtime.GOOS, runtime.GOARCH, runtime.Version(), runtime.NumCPU()),
				},
			},
		}
	case "echo":
		var params struct {
			Message string `json:"message"`
		}
		json.Unmarshal(args, &params)
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("Echo: %s", params.Message),
				},
			},
		}
	default:
		return map[string]interface{}{
			"error": "Unknown tool",
		}
	}
}