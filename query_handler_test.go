package claudesdk

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

// controlMockTransport is a specialized mock that simulates bidirectional control protocol.
// It watches for outgoing control_requests and sends back matching control_responses.
type controlMockTransport struct {
	mu           sync.Mutex
	written      []string
	msgCh        chan json.RawMessage
	connected    bool
	closed       bool
	onWriteHooks []func(data string)
}

func newControlMockTransport() *controlMockTransport {
	return &controlMockTransport{
		msgCh: make(chan json.RawMessage, 64),
	}
}

func (t *controlMockTransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.connected = true
	return nil
}

func (t *controlMockTransport) Write(ctx context.Context, data string) error {
	t.mu.Lock()
	t.written = append(t.written, data)
	hooks := make([]func(string), len(t.onWriteHooks))
	copy(hooks, t.onWriteHooks)
	t.mu.Unlock()
	for _, h := range hooks {
		h(data)
	}
	return nil
}

func (t *controlMockTransport) ReadMessages(ctx context.Context) (<-chan json.RawMessage, <-chan error) {
	errCh := make(chan error)
	go func() { <-ctx.Done(); close(errCh) }()
	return t.msgCh, errCh
}

func (t *controlMockTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
	close(t.msgCh)
	return nil
}

func (t *controlMockTransport) IsReady() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.connected && !t.closed
}

func (t *controlMockTransport) EndInput() error { return nil }

// sendResponse injects a control_response into the message channel.
func (t *controlMockTransport) sendResponse(requestID string, response any) {
	msg := map[string]any{
		"type": "control_response",
		"response": map[string]any{
			"subtype":    "success",
			"request_id": requestID,
			"response":   response,
		},
	}
	data, _ := json.Marshal(msg)
	t.msgCh <- json.RawMessage(data)
}

// sendControlRequest injects a control_request (from CLI) into the message channel.
func (t *controlMockTransport) sendControlRequest(requestID, subtype string, fields map[string]any) {
	req := map[string]any{
		"subtype": subtype,
	}
	for k, v := range fields {
		req[k] = v
	}
	msg := map[string]any{
		"type":       "control_request",
		"request_id": requestID,
		"request":    req,
	}
	data, _ := json.Marshal(msg)
	t.msgCh <- json.RawMessage(data)
}

// onWrite registers a callback for each write.
func (t *controlMockTransport) onWrite(fn func(data string)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onWriteHooks = append(t.onWriteHooks, fn)
}

// lastWritten returns the last written data.
func (t *controlMockTransport) lastWritten() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.written) == 0 {
		return ""
	}
	return t.written[len(t.written)-1]
}

// --- Tests for permission handling ---

func TestQueryHandler_HandlePermissionRequest_DefaultAllow(t *testing.T) {
	mt := newControlMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mt.Connect(ctx)
	opts := &ClaudeAgentOptions{} // No CanUseTool -> default allow
	handler := newQueryHandler(mt, opts)
	handler.runMessageRouter(ctx)

	// Simulate a can_use_tool request from CLI
	mt.sendControlRequest("perm-1", "can_use_tool", map[string]any{
		"tool_name":   "Read",
		"input":       json.RawMessage(`{"file_path":"/tmp/test"}`),
		"tool_use_id": "tu-1",
	})

	// Wait for the response to be written
	time.Sleep(100 * time.Millisecond)

	// Check the response was an allow
	found := false
	mt.mu.Lock()
	for _, w := range mt.written {
		var msg map[string]any
		json.Unmarshal([]byte(w), &msg)
		if msg["type"] == "control_response" {
			resp := msg["response"].(map[string]any)
			if resp["request_id"] == "perm-1" {
				innerResp := resp["response"].(map[string]any)
				if innerResp["behavior"] == "allow" {
					found = true
				}
			}
		}
	}
	mt.mu.Unlock()
	if !found {
		t.Fatal("expected allow response for permission request with no CanUseTool callback")
	}
}

func TestQueryHandler_HandlePermissionRequest_CustomDeny(t *testing.T) {
	mt := newControlMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mt.Connect(ctx)
	opts := &ClaudeAgentOptions{
		CanUseTool: func(toolName string, input map[string]any, ctx ToolPermissionContext) (PermissionResult, error) {
			return PermissionResult{
				Deny: &PermissionResultDeny{Message: "tool not allowed"},
			}, nil
		},
	}
	handler := newQueryHandler(mt, opts)
	handler.runMessageRouter(ctx)

	mt.sendControlRequest("perm-2", "can_use_tool", map[string]any{
		"tool_name":   "Bash",
		"input":       json.RawMessage(`{"command":"rm -rf /"}`),
		"tool_use_id": "tu-2",
	})

	time.Sleep(100 * time.Millisecond)

	found := false
	mt.mu.Lock()
	for _, w := range mt.written {
		var msg map[string]any
		json.Unmarshal([]byte(w), &msg)
		if msg["type"] == "control_response" {
			resp := msg["response"].(map[string]any)
			if resp["request_id"] == "perm-2" {
				innerResp := resp["response"].(map[string]any)
				if innerResp["behavior"] == "deny" && innerResp["message"] == "tool not allowed" {
					found = true
				}
			}
		}
	}
	mt.mu.Unlock()
	if !found {
		t.Fatal("expected deny response for permission request with custom CanUseTool callback")
	}
}

func TestQueryHandler_HandlePermissionRequest_CustomAllow(t *testing.T) {
	mt := newControlMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mt.Connect(ctx)
	opts := &ClaudeAgentOptions{
		CanUseTool: func(toolName string, input map[string]any, ctx ToolPermissionContext) (PermissionResult, error) {
			return PermissionResult{
				Allow: &PermissionResultAllow{
					UpdatedInput: map[string]any{"sanitized": true},
				},
			}, nil
		},
	}
	handler := newQueryHandler(mt, opts)
	handler.runMessageRouter(ctx)

	mt.sendControlRequest("perm-3", "can_use_tool", map[string]any{
		"tool_name":   "Write",
		"input":       json.RawMessage(`{"file_path":"/tmp/out"}`),
		"tool_use_id": "tu-3",
	})

	time.Sleep(100 * time.Millisecond)

	found := false
	mt.mu.Lock()
	for _, w := range mt.written {
		var msg map[string]any
		json.Unmarshal([]byte(w), &msg)
		if msg["type"] == "control_response" {
			resp := msg["response"].(map[string]any)
			if resp["request_id"] == "perm-3" {
				innerResp := resp["response"].(map[string]any)
				if innerResp["behavior"] == "allow" {
					updatedInput := innerResp["updatedInput"].(map[string]any)
					if updatedInput["sanitized"] == true {
						found = true
					}
				}
			}
		}
	}
	mt.mu.Unlock()
	if !found {
		t.Fatal("expected allow response with updated input")
	}
}

func TestQueryHandler_HandlePermissionRequest_CallbackError(t *testing.T) {
	mt := newControlMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mt.Connect(ctx)
	opts := &ClaudeAgentOptions{
		CanUseTool: func(toolName string, input map[string]any, ctx ToolPermissionContext) (PermissionResult, error) {
			return PermissionResult{}, fmt.Errorf("callback error")
		},
	}
	handler := newQueryHandler(mt, opts)
	handler.runMessageRouter(ctx)

	mt.sendControlRequest("perm-4", "can_use_tool", map[string]any{
		"tool_name": "Bash",
		"input":     json.RawMessage(`{}`),
	})

	time.Sleep(100 * time.Millisecond)

	found := false
	mt.mu.Lock()
	for _, w := range mt.written {
		var msg map[string]any
		json.Unmarshal([]byte(w), &msg)
		if msg["type"] == "control_response" {
			resp := msg["response"].(map[string]any)
			if resp["request_id"] == "perm-4" {
				innerResp := resp["response"].(map[string]any)
				if innerResp["behavior"] == "deny" && innerResp["message"] == "callback error" {
					found = true
				}
			}
		}
	}
	mt.mu.Unlock()
	if !found {
		t.Fatal("expected deny response when CanUseTool returns error")
	}
}

// --- Tests for hook callbacks ---

func TestQueryHandler_HandleHookCallback_Success(t *testing.T) {
	mt := newControlMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mt.Connect(ctx)

	called := false
	opts := &ClaudeAgentOptions{}
	handler := newQueryHandler(mt, opts)

	// Register a hook callback directly
	handler.hookCallbacks["hook_0"] = func(input json.RawMessage, toolUseID string) (HookJSONOutput, error) {
		called = true
		return HookJSONOutput{}, nil
	}

	handler.runMessageRouter(ctx)

	mt.sendControlRequest("hook-1", "hook_callback", map[string]any{
		"callback_id": "hook_0",
		"input":       json.RawMessage(`{"tool_name":"Read"}`),
		"tool_use_id": "tu-hook-1",
	})

	time.Sleep(100 * time.Millisecond)

	if !called {
		t.Fatal("hook callback was not invoked")
	}

	// Verify success response was sent
	found := false
	mt.mu.Lock()
	for _, w := range mt.written {
		var msg map[string]any
		json.Unmarshal([]byte(w), &msg)
		if msg["type"] == "control_response" {
			resp := msg["response"].(map[string]any)
			if resp["request_id"] == "hook-1" && resp["subtype"] == "success" {
				found = true
			}
		}
	}
	mt.mu.Unlock()
	if !found {
		t.Fatal("expected success response for hook callback")
	}
}

func TestQueryHandler_HandleHookCallback_NotFound(t *testing.T) {
	mt := newControlMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mt.Connect(ctx)
	handler := newQueryHandler(mt, &ClaudeAgentOptions{})
	handler.runMessageRouter(ctx)

	mt.sendControlRequest("hook-2", "hook_callback", map[string]any{
		"callback_id": "nonexistent",
		"input":       json.RawMessage(`{}`),
	})

	time.Sleep(100 * time.Millisecond)

	found := false
	mt.mu.Lock()
	for _, w := range mt.written {
		var msg map[string]any
		json.Unmarshal([]byte(w), &msg)
		if msg["type"] == "control_response" {
			resp := msg["response"].(map[string]any)
			if resp["request_id"] == "hook-2" && resp["subtype"] == "error" {
				found = true
			}
		}
	}
	mt.mu.Unlock()
	if !found {
		t.Fatal("expected error response for unknown callback ID")
	}
}

// --- Tests for cancel request ---

func TestQueryHandler_HandleCancelRequest(t *testing.T) {
	mt := newControlMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mt.Connect(ctx)
	handler := newQueryHandler(mt, &ClaudeAgentOptions{})
	handler.runMessageRouter(ctx)

	mt.sendControlRequest("cancel-1", "control_cancel_request", nil)

	time.Sleep(100 * time.Millisecond)

	found := false
	mt.mu.Lock()
	for _, w := range mt.written {
		var msg map[string]any
		json.Unmarshal([]byte(w), &msg)
		if msg["type"] == "control_response" {
			resp := msg["response"].(map[string]any)
			if resp["request_id"] == "cancel-1" {
				innerResp := resp["response"].(map[string]any)
				if innerResp["acknowledged"] == true {
					found = true
				}
			}
		}
	}
	mt.mu.Unlock()
	if !found {
		t.Fatal("expected acknowledged response for cancel request")
	}
}

// --- Tests for control request methods (interrupt, setModel, etc.) ---

func TestQueryHandler_Interrupt(t *testing.T) {
	mt := newControlMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mt.Connect(ctx)
	handler := newQueryHandler(mt, &ClaudeAgentOptions{})
	handler.runMessageRouter(ctx)

	// Auto-respond to the interrupt control_request
	mt.onWrite(func(data string) {
		var msg map[string]any
		json.Unmarshal([]byte(data), &msg)
		if msg["type"] == "control_request" {
			reqID := msg["request_id"].(string)
			req := msg["request"].(map[string]any)
			if req["subtype"] == "interrupt" {
				mt.sendResponse(reqID, map[string]any{"interrupted": true})
			}
		}
	})

	err := handler.interrupt(ctx)
	if err != nil {
		t.Fatalf("interrupt: %v", err)
	}
}

func TestQueryHandler_SetPermissionMode(t *testing.T) {
	mt := newControlMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mt.Connect(ctx)
	handler := newQueryHandler(mt, &ClaudeAgentOptions{})
	handler.runMessageRouter(ctx)

	mt.onWrite(func(data string) {
		var msg map[string]any
		json.Unmarshal([]byte(data), &msg)
		if msg["type"] == "control_request" {
			reqID := msg["request_id"].(string)
			req := msg["request"].(map[string]any)
			if req["subtype"] == "set_permission_mode" {
				mt.sendResponse(reqID, map[string]any{"ok": true})
			}
		}
	})

	err := handler.setPermissionMode(ctx, PermissionModeDontAsk)
	if err != nil {
		t.Fatalf("setPermissionMode: %v", err)
	}

	// Verify the mode was sent correctly
	mt.mu.Lock()
	var foundMode string
	for _, w := range mt.written {
		var msg map[string]any
		json.Unmarshal([]byte(w), &msg)
		if msg["type"] == "control_request" {
			req := msg["request"].(map[string]any)
			if req["subtype"] == "set_permission_mode" {
				foundMode, _ = req["mode"].(string)
			}
		}
	}
	mt.mu.Unlock()
	if foundMode != string(PermissionModeDontAsk) {
		t.Fatalf("expected mode %q, got %q", PermissionModeDontAsk, foundMode)
	}
}

func TestQueryHandler_SetModel(t *testing.T) {
	mt := newControlMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mt.Connect(ctx)
	handler := newQueryHandler(mt, &ClaudeAgentOptions{})
	handler.runMessageRouter(ctx)

	mt.onWrite(func(data string) {
		var msg map[string]any
		json.Unmarshal([]byte(data), &msg)
		if msg["type"] == "control_request" {
			reqID := msg["request_id"].(string)
			req := msg["request"].(map[string]any)
			if req["subtype"] == "set_model" {
				mt.sendResponse(reqID, map[string]any{"ok": true})
			}
		}
	})

	err := handler.setModel(ctx, "claude-opus-4-6")
	if err != nil {
		t.Fatalf("setModel: %v", err)
	}
}

func TestQueryHandler_McpServerStatus(t *testing.T) {
	mt := newControlMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mt.Connect(ctx)
	handler := newQueryHandler(mt, &ClaudeAgentOptions{})
	handler.runMessageRouter(ctx)

	mt.onWrite(func(data string) {
		var msg map[string]any
		json.Unmarshal([]byte(data), &msg)
		if msg["type"] == "control_request" {
			reqID := msg["request_id"].(string)
			req := msg["request"].(map[string]any)
			if req["subtype"] == "mcp_status" {
				mt.sendResponse(reqID, []map[string]any{
					{"name": "test-server", "status": "connected"},
				})
			}
		}
	})

	statuses, err := handler.mcpServerStatus(ctx)
	if err != nil {
		t.Fatalf("mcpServerStatus: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
}

func TestQueryHandler_ContextUsage(t *testing.T) {
	mt := newControlMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mt.Connect(ctx)
	handler := newQueryHandler(mt, &ClaudeAgentOptions{})
	handler.runMessageRouter(ctx)

	mt.onWrite(func(data string) {
		var msg map[string]any
		json.Unmarshal([]byte(data), &msg)
		if msg["type"] == "control_request" {
			reqID := msg["request_id"].(string)
			req := msg["request"].(map[string]any)
			if req["subtype"] == "context_usage" {
				mt.sendResponse(reqID, map[string]any{
					"total_tokens": 1000,
					"used_tokens":  500,
				})
			}
		}
	})

	usage, err := handler.contextUsage(ctx)
	if err != nil {
		t.Fatalf("contextUsage: %v", err)
	}
	if usage == nil {
		t.Fatal("expected non-nil context usage")
	}
}

func TestQueryHandler_RewindFiles(t *testing.T) {
	mt := newControlMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mt.Connect(ctx)
	handler := newQueryHandler(mt, &ClaudeAgentOptions{})
	handler.runMessageRouter(ctx)

	mt.onWrite(func(data string) {
		var msg map[string]any
		json.Unmarshal([]byte(data), &msg)
		if msg["type"] == "control_request" {
			reqID := msg["request_id"].(string)
			req := msg["request"].(map[string]any)
			if req["subtype"] == "rewind_files" {
				mt.sendResponse(reqID, map[string]any{"ok": true})
			}
		}
	})

	err := handler.rewindFiles(ctx, "msg-123")
	if err != nil {
		t.Fatalf("rewindFiles: %v", err)
	}
}

func TestQueryHandler_ReconnectMcpServer(t *testing.T) {
	mt := newControlMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mt.Connect(ctx)
	handler := newQueryHandler(mt, &ClaudeAgentOptions{})
	handler.runMessageRouter(ctx)

	mt.onWrite(func(data string) {
		var msg map[string]any
		json.Unmarshal([]byte(data), &msg)
		if msg["type"] == "control_request" {
			reqID := msg["request_id"].(string)
			req := msg["request"].(map[string]any)
			if req["subtype"] == "mcp_reconnect" {
				mt.sendResponse(reqID, map[string]any{"ok": true})
			}
		}
	})

	err := handler.reconnectMcpServer(ctx, "my-server")
	if err != nil {
		t.Fatalf("reconnectMcpServer: %v", err)
	}
}

func TestQueryHandler_ToggleMcpServer(t *testing.T) {
	mt := newControlMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mt.Connect(ctx)
	handler := newQueryHandler(mt, &ClaudeAgentOptions{})
	handler.runMessageRouter(ctx)

	mt.onWrite(func(data string) {
		var msg map[string]any
		json.Unmarshal([]byte(data), &msg)
		if msg["type"] == "control_request" {
			reqID := msg["request_id"].(string)
			req := msg["request"].(map[string]any)
			if req["subtype"] == "mcp_toggle" {
				mt.sendResponse(reqID, map[string]any{"ok": true})
			}
		}
	})

	err := handler.toggleMcpServer(ctx, "my-server", false)
	if err != nil {
		t.Fatalf("toggleMcpServer: %v", err)
	}
}

func TestQueryHandler_StopTask(t *testing.T) {
	mt := newControlMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mt.Connect(ctx)
	handler := newQueryHandler(mt, &ClaudeAgentOptions{})
	handler.runMessageRouter(ctx)

	mt.onWrite(func(data string) {
		var msg map[string]any
		json.Unmarshal([]byte(data), &msg)
		if msg["type"] == "control_request" {
			reqID := msg["request_id"].(string)
			req := msg["request"].(map[string]any)
			if req["subtype"] == "stop_task" {
				mt.sendResponse(reqID, map[string]any{"ok": true})
			}
		}
	})

	err := handler.stopTask(ctx, "task-42")
	if err != nil {
		t.Fatalf("stopTask: %v", err)
	}
}

// --- Tests for initialize with various options ---

func TestQueryHandler_Initialize_WithSystemPromptString(t *testing.T) {
	mt := newControlMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mt.Connect(ctx)
	opts := &ClaudeAgentOptions{
		SystemPrompt: "You are a helpful assistant",
	}
	handler := newQueryHandler(mt, opts)
	handler.runMessageRouter(ctx)

	// Auto-respond to initialize
	mt.onWrite(func(data string) {
		var msg map[string]any
		json.Unmarshal([]byte(data), &msg)
		if msg["type"] == "control_request" {
			reqID := msg["request_id"].(string)
			req := msg["request"].(map[string]any)
			if req["subtype"] == "initialize" {
				// Verify systemPrompt was sent
				if req["systemPrompt"] != "You are a helpful assistant" {
					t.Errorf("expected systemPrompt, got %v", req["systemPrompt"])
				}
				mt.sendResponse(reqID, map[string]any{
					"response": map[string]any{
						"commands": []any{},
						"models":   []any{},
					},
				})
			}
		}
	})

	_, err := handler.initialize(ctx)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
}

func TestQueryHandler_Initialize_WithHookCallbacks(t *testing.T) {
	mt := newControlMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mt.Connect(ctx)

	callbackCalled := false
	opts := &ClaudeAgentOptions{
		HookCallbacks: map[HookEvent][]HookCallbackMatcher{
			HookEventPreToolUse: {
				{
					Matcher: "Bash",
					Hooks: []HookCallback{
						func(input json.RawMessage, toolUseID string) (HookJSONOutput, error) {
							callbackCalled = true
							return HookJSONOutput{}, nil
						},
					},
				},
			},
		},
	}
	handler := newQueryHandler(mt, opts)
	handler.runMessageRouter(ctx)

	// Auto-respond to initialize
	mt.onWrite(func(data string) {
		var msg map[string]any
		json.Unmarshal([]byte(data), &msg)
		if msg["type"] == "control_request" {
			reqID := msg["request_id"].(string)
			req := msg["request"].(map[string]any)
			if req["subtype"] == "initialize" {
				// Verify hooks were sent
				if req["hooks"] == nil {
					t.Error("expected hooks in initialize request")
				}
				mt.sendResponse(reqID, map[string]any{
					"response": map[string]any{},
				})
			}
		}
	})

	_, err := handler.initialize(ctx)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}

	// Verify hook was registered
	if len(handler.hookCallbacks) != 1 {
		t.Fatalf("expected 1 hook callback, got %d", len(handler.hookCallbacks))
	}

	// Test the callback can be invoked
	for id, cb := range handler.hookCallbacks {
		_, err := cb(json.RawMessage(`{}`), "")
		if err != nil {
			t.Fatalf("callback %s: %v", id, err)
		}
	}
	if !callbackCalled {
		t.Fatal("callback was not invoked")
	}
}

// --- Tests for unknown control request ---

func TestQueryHandler_HandleUnknownControlRequest(t *testing.T) {
	mt := newControlMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mt.Connect(ctx)
	handler := newQueryHandler(mt, &ClaudeAgentOptions{})
	handler.runMessageRouter(ctx)

	mt.sendControlRequest("unknown-1", "some_future_subtype", nil)

	time.Sleep(100 * time.Millisecond)

	// Verify a default response was sent
	found := false
	mt.mu.Lock()
	for _, w := range mt.written {
		var msg map[string]any
		json.Unmarshal([]byte(w), &msg)
		if msg["type"] == "control_response" {
			resp := msg["response"].(map[string]any)
			if resp["request_id"] == "unknown-1" {
				found = true
			}
		}
	}
	mt.mu.Unlock()
	if !found {
		t.Fatal("expected default response for unknown control request subtype")
	}
}
