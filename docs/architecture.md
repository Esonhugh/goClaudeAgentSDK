# Architecture

## Overview

The Go Claude Agent SDK communicates with the Claude Code CLI via a bidirectional JSON streaming protocol over stdin/stdout. The SDK spawns the CLI as a subprocess, sends control requests and user prompts, and receives structured messages back.

```
┌──────────────┐   stdin (JSON lines)   ┌──────────────────┐
│              │ ─────────────────────▸ │                  │
│   Go SDK     │                        │  Claude Code CLI │
│              │ ◂───────────────────── │                  │
└──────────────┘   stdout (JSON lines)  └──────────────────┘
                   stderr (debug logs)
```

## Layer Diagram

```
┌───────────────────────────────────────────────────────────┐
│  Agent / Pipeline / Parallel   (High-level orchestration) │
├───────────────────────────────────────────────────────────┤
│  Client (ClaudeSDKClient)      (Multi-turn conversation)  │
├───────────────────────────────────────────────────────────┤
│  Query Handler                 (Control protocol routing) │
├───────────────────────────────────────────────────────────┤
│  Transport (interface)         (Subprocess I/O)           │
├───────────────────────────────────────────────────────────┤
│  SubprocessTransport           (Process lifecycle)        │
└───────────────────────────────────────────────────────────┘
```

## Components

### Transport Layer (`transport.go`, `subprocess_transport.go`)

The `Transport` interface abstracts CLI communication:

```go
type Transport interface {
    Connect(ctx context.Context) error
    Write(ctx context.Context, data string) error
    ReadMessages(ctx context.Context) (<-chan json.RawMessage, <-chan error)
    Close() error
    IsReady() bool
    EndInput() error
}
```

`SubprocessTransport` implements this by:
1. Finding the `claude` binary (PATH, known locations, custom path)
2. Building CLI arguments from `ClaudeAgentOptions`
3. Spawning the process with filtered environment variables
4. Reading/writing JSON lines over stdin/stdout

### Query Handler (`query.go`)

Manages the control protocol:
- **Initialize handshake**: Sends `initialize` control request with options (agents, hooks, system prompt)
- **Control request/response**: Sends typed requests (interrupt, set_model, mcp_status, etc.) and correlates responses by request ID
- **Incoming request routing**: Handles `can_use_tool` (permission callbacks), `hook_callback`, and `control_cancel_request` from the CLI
- **Message routing**: Separates control messages from data messages (assistant, system, result)

### Message Parser (`message_parser.go`)

Converts raw JSON into typed Go messages:
- `AssistantMessage` — model responses with content blocks (text, thinking, tool_use, tool_result)
- `SystemMessage` — system events (init, task_started, task_progress, task_notification)
- `ResultMessage` — final conversation results with cost/usage info
- `StreamEvent` — streaming events (rate_limit, partial messages)

Unknown message types return `nil` (forward-compatible).

### Client (`client.go`)

`ClaudeSDKClient` provides the mid-level API:
- `Connect()` → starts transport, runs initialize handshake
- `SendQuery(prompt)` → sends user message
- `ReceiveMessages()` → returns channel of typed messages
- Control methods: `Interrupt()`, `SetPermissionMode()`, `SetModel()`, `GetMCPStatus()`, `GetContextUsage()`, `RewindFiles()`, `StopTask()`

### Agent (`agent.go`)

High-level abstraction for single-prompt interactions:
- Wraps Client with `AgentConfig` (name, model, prompt, system prompt, max turns)
- `Run(ctx, prompt)` → returns `AgentResult` (messages, cost, session ID, duration)
- `StartSession()` → returns a Client for multi-turn use

### Pipeline (`pipeline.go`)

Sequential agent composition:
- Each `PipelineStep` has an Agent and optional `TransformOutput` function
- Output of step N becomes input to step N+1
- Collects per-step results and total cost

### Parallel (`parallel.go`)

Concurrent agent execution:
- `RunParallel(tasks)` → runs agents concurrently, returns results
- `RunParallelCollect(tasks)` → same but collects all results
- `FanOut(prompt, agents)` → same prompt to multiple agents
- `Race(tasks)` → returns first completed result

### Error Hierarchy (`errors.go`)

```
ClaudeSDKError (base)
├── CLINotFoundError    — binary not found
├── CLIConnectionError  — pipe/process errors
├── ProcessError        — non-zero exit code
├── CLIJSONDecodeError  — malformed JSON from CLI
└── MessageParseError   — unexpected message structure
```

### Sessions (`sessions.go`)

Offline session management (no running CLI needed):
- `ListSessions()` — enumerate JSONL session files
- `GetSessionInfo()` / `GetSessionMessages()` — read session data
- `RenameSession()` / `TagSession()` / `DeleteSession()` / `ForkSession()`

## Protocol Flow

```
SDK                                  CLI
 │                                    │
 │──── control_request(initialize) ──▸│
 │◂─── control_response(init_ack) ────│
 │                                    │
 │──── user message (prompt) ────────▸│
 │◂─── assistant message ────────────│
 │◂─── control_request(can_use_tool) ─│
 │──── control_response(allow) ──────▸│
 │◂─── assistant message ────────────│
 │◂─── result message ───────────────│
 │                                    │
 │──── [close stdin] ────────────────▸│
```

## Environment Variables

The SDK manages environment variables at two levels:

1. **SDK-internal vars** (set automatically): `CLAUDE_CODE_ENTRYPOINT`, `CLAUDE_AGENT_SDK_VERSION`, `CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING`
2. **User-configurable vars** (via `ClaudeAgentOptions.Env`): Any env var can be passed through
3. **Nesting prevention**: `CLAUDECODE` is always filtered from the inherited environment

See `env.go` for the full list of constants.
