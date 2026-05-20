package mcppool

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"

	"SREFlow/internal/models"
)

// MCPPool manages cached mcp-go/client.Client instances by server ID.
type MCPPool struct {
	clients sync.Map // serverID -> *poolEntry
}

type poolEntry struct {
	client    *mcpclient.Client
	serverURL string
	mu        sync.Mutex
}

// DefaultMCPPool is the global MCP client pool.
var DefaultMCPPool = &MCPPool{}

// GetOrCreateClient returns a cached MCP client or creates a new one.
func (p *MCPPool) GetOrCreateClient(ctx context.Context, serverID, serverURL string, serverType models.MCPServerType, headers map[string]string, apiKey string, timeout int) (*mcpclient.Client, error) {
	// Fast path: return cached client if available and initialized
	if val, ok := p.clients.Load(serverID); ok {
		entry := val.(*poolEntry)
		entry.mu.Lock()
		client := entry.client
		entry.mu.Unlock()
		if client != nil {
			return client, nil
		}
		// Stale entry (previous creation failed), remove and recreate
		p.clients.Delete(serverID)
	}

	// Slow path: create a new entry and client
	entry := &poolEntry{serverURL: serverURL}
	entry.mu.Lock()
	defer entry.mu.Unlock()

	actual, loaded := p.clients.LoadOrStore(serverID, entry)
	if loaded {
		// Another goroutine is creating the client — wait for it
		existing := actual.(*poolEntry)
		existing.mu.Lock()
		defer existing.mu.Unlock()
		if existing.client != nil {
			return existing.client, nil
		}
		// Previous creation failed — clean up and return error
		p.clients.Delete(serverID)
		return nil, fmt.Errorf("MCP client for server %s is not available (previous initialization failed)", serverID)
	}

	client, err := p.createClient(ctx, serverURL, serverType, headers, apiKey, timeout)
	if err != nil {
		p.clients.Delete(serverID)
		return nil, err
	}

	entry.client = client
	return client, nil
}

func (p *MCPPool) createClient(ctx context.Context, serverURL string, serverType models.MCPServerType, headers map[string]string, apiKey string, timeout int) (*mcpclient.Client, error) {
	log.Printf("[MCPPool] Creating MCP client: serverURL=%s, type=%s, timeout=%ds, hasAPIKey=%v, headerCount=%d",
		serverURL, serverType, timeout, apiKey != "", len(headers))

	if apiKey != "" {
		if headers == nil {
			headers = make(map[string]string)
		}
		headers["Authorization"] = "Bearer " + apiKey
	}

	var client *mcpclient.Client
	var err error

	timeoutDuration := time.Duration(timeout) * time.Second
	if timeoutDuration <= 0 {
		log.Printf("[MCPPool] Timeout was %ds (<=0), defaulting to 30s", timeout)
		timeoutDuration = 30 * time.Second
	}

	switch serverType {
	case models.MCPServerTypeHTTP, models.MCPServerTypeStreamableHTTP:
		log.Printf("[MCPPool] Using StreamableHTTP transport for %s", serverURL)
		opts := []transport.StreamableHTTPCOption{
			transport.WithHTTPTimeout(timeoutDuration),
		}
		if len(headers) > 0 {
			opts = append(opts, transport.WithHTTPHeaders(headers))
		}
		client, err = mcpclient.NewStreamableHttpClient(serverURL, opts...)
	case models.MCPServerTypeSSE:
		log.Printf("[MCPPool] Using SSE transport for %s", serverURL)
		httpClient := &http.Client{Timeout: timeoutDuration}
		opts := []transport.ClientOption{
			transport.WithHTTPClient(httpClient),
		}
		if len(headers) > 0 {
			opts = append(opts, transport.WithHeaders(headers))
		}
		client, err = mcpclient.NewSSEMCPClient(serverURL, opts...)
	default:
		log.Printf("[MCPPool] Unsupported server type: %s", serverType)
		return nil, fmt.Errorf("unsupported MCP server type: %s", serverType)
	}

	if err != nil {
		log.Printf("[MCPPool] Failed to create MCP client: %v", err)
		return nil, fmt.Errorf("failed to create MCP client: %w", err)
	}

	log.Printf("[MCPPool] Client created, calling Start()...")
	if err := client.Start(ctx); err != nil {
		log.Printf("[MCPPool] Client.Start() failed: %v", err)
		return nil, fmt.Errorf("failed to start MCP client: %w", err)
	}
	log.Printf("[MCPPool] Client.Start() succeeded, calling Initialize()...")

	initResult, err := client.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "SREFlow",
				Version: "1.0.0",
			},
		},
	})
	if err != nil {
		log.Printf("[MCPPool] Initialize() failed: %v", err)
		client.Close()
		return nil, fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	log.Printf("[MCPPool] Connected to MCP server %s (protocol: %s, server: %s)",
		serverURL, initResult.ProtocolVersion, initResult.ServerInfo.Name)

	return client, nil
}

// CloseAll closes all cached clients and clears the pool.
func (p *MCPPool) CloseAll() {
	p.clients.Range(func(key, value any) bool {
		entry := value.(*poolEntry)
		if entry.client != nil {
			entry.client.Close()
		}
		p.clients.Delete(key)
		return true
	})
}

// RemoveClient removes and closes a specific client from the pool.
func (p *MCPPool) RemoveClient(serverID string) {
	val, ok := p.clients.LoadAndDelete(serverID)
	if ok {
		entry := val.(*poolEntry)
		if entry.client != nil {
			entry.client.Close()
		}
	}
}
