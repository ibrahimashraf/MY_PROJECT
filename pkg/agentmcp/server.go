package agentmcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"integin/pkg/domain"
)

var (
	ErrPathEscape       = errors.New("access denied: path escapes sandboxed root")
	ErrInvalidJSONRPC   = errors.New("invalid JSON-RPC 2.0 request")
	ErrExecutionFailed  = errors.New("command execution failed")
)

// ServerConfig defines the sandbox boundaries and execution constraints.
type ServerConfig struct {
	ReadRoot  string
	WriteRoot string
}

// JSONRPCRequest represents a standard incoming MCP / JSON-RPC 2.0 call.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a standard outgoing MCP / JSON-RPC 2.0 reply.
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Server exposes the domain CommandDispatcher via a sandboxed loopback JSON-RPC/MCP protocol.
type Server struct {
	cfg        ServerConfig
	dispatcher *domain.Dispatcher
	mu         sync.Mutex
}

// NewServer creates a new sandboxed MCP server instance.
func NewServer(cfg ServerConfig, dispatcher *domain.Dispatcher) (*Server, error) {
	absRead, err := filepath.Abs(cfg.ReadRoot)
	if err != nil {
		return nil, fmt.Errorf("invalid read root: %w", err)
	}
	absWrite, err := filepath.Abs(cfg.WriteRoot)
	if err != nil {
		return nil, fmt.Errorf("invalid write root: %w", err)
	}

	return &Server{
		cfg: ServerConfig{
			ReadRoot:  absRead,
			WriteRoot: absWrite,
		},
		dispatcher: dispatcher,
	}, nil
}

// ValidatePath ensures that any filesystem path parameter stays strictly within the sandboxed root.
func (s *Server) ValidatePath(targetPath string, isWrite bool) (string, error) {
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path: %w", err)
	}

	root := s.cfg.ReadRoot
	if isWrite {
		root = s.cfg.WriteRoot
	}

	rel, err := filepath.Rel(root, absTarget)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("%w: target %s is outside root %s", ErrPathEscape, targetPath, root)
	}

	return absTarget, nil
}

// HandleRequest processes an incoming JSON-RPC payload and invokes the domain dispatcher.
func (s *Server) HandleRequest(ctx context.Context, rawReq []byte) JSONRPCResponse {
	var req JSONRPCRequest
	if err := json.Unmarshal(rawReq, &req); err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      nil,
			Error: &RPCError{
				Code:    -32700,
				Message: fmt.Sprintf("%v: %v", ErrInvalidJSONRPC, err),
			},
		}
	}

	switch req.Method {
	case "mcp.list_tools":
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"tools": []string{"domain.execute", "fs.validate_path"},
			},
		}

	case "domain.execute":
		var cmd domain.Command
		if err := json.Unmarshal(req.Params, &cmd); err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &RPCError{
					Code:    -32602,
					Message: fmt.Sprintf("invalid command params: %v", err),
				},
			}
		}

		if cmd.Timestamp.IsZero() {
			cmd.Timestamp = time.Now()
		}

		res, err := s.dispatcher.Execute(ctx, cmd)
		if err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &RPCError{
					Code:    -32000,
					Message: err.Error(),
				},
			}
		}

		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  res,
		}

	default:
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32601,
				Message: fmt.Sprintf("method not found: %s", req.Method),
			},
		}
	}
}

// ServeStream processes newline-delimited JSON-RPC messages over an io.Reader/Writer pair (e.g. pipe or socket).
func (s *Server) ServeStream(ctx context.Context, r io.Reader, w io.Writer) error {
	decoder := json.NewDecoder(r)
	encoder := json.NewEncoder(w)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var raw json.RawMessage
			if err := decoder.Decode(&raw); err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			}

			resp := s.HandleRequest(ctx, raw)
			if err := encoder.Encode(resp); err != nil {
				return err
			}
		}
	}
}

// ServeHTTP provides standard HTTP POST JSON-RPC transport for the loopback MCP daemon.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed: MCP requires POST", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	resp := s.HandleRequest(r.Context(), body)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
