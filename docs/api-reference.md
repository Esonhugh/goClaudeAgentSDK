# API Reference

## Package `claudesdk`

Import: `github.com/Esonhugh/goClaudeAgentSDK`

---

## Top-Level Functions

### `Query`

```go
func Query(ctx context.Context, prompt string, opts ClaudeAgentOptions) (<-chan Message, <-chan error)
```

One-shot convenience function. Spawns the CLI, sends a prompt, and streams messages back. The channels close when the conversation ends.

### `RunParallel`

```go
func RunParallel(ctx context.Context, tasks []ParallelTask) []ParallelResult
```

Runs multiple agent tasks concurrently, returning results in the same order as input.

### `RunParallelCollect`

```go
func RunParallelCollect(ctx context.Context, tasks []ParallelTask) ([]ParallelResult, error)
```

Like `RunParallel` but returns an error if any task fails.

### `FanOut`

```go
func FanOut(ctx context.Context, prompt string, agents []*Agent) []ParallelResult
```

Sends the same prompt to multiple agents concurrently.

### `Race`

```go
func Race(ctx context.Context, tasks []ParallelTask) (*ParallelResult, error)
```

Returns the first completed result, cancelling remaining tasks.

---

## Session Management Functions

### `ListSessions`

```go
func ListSessions(opts ListSessionsOptions) []SDKSessionInfo
```

Lists sessions from disk. Filter by `ProjectDir` and limit with `Limit`.

### `GetSessionInfo`

```go
func GetSessionInfo(sessionID string) (*SDKSessionInfo, error)
```

Reads metadata for a specific session.

### `GetSessionMessages`

```go
func GetSessionMessages(sessionID string) ([]SessionMessage, error)
```

Reads all messages from a session file.

### `RenameSession`

```go
func RenameSession(sessionID, newName string) error
```

### `TagSession`

```go
func TagSession(sessionID, tag string) error
```

### `DeleteSession`

```go
func DeleteSession(sessionID string) error
```

### `ForkSession`

```go
func ForkSession(sessionID string, atMessageID string) (*ForkSessionResult, error)
```

Creates a copy of a session, optionally truncated at a specific message.

---

## Types

### `ClaudeAgentOptions`

Primary configuration struct for all SDK operations.

| Field | Type | Description |
|-------|------|-------------|
| `Model` | `string` | Model identifier (e.g., `"claude-sonnet-4-6"`) |
| `FallbackModel` | `string` | Fallback model if primary unavailable |
| `PermissionMode` | `PermissionMode` | Tool permission handling mode |
| `AllowDangerouslySkipPermissions` | `bool` | Skip all permission checks |
| `SystemPrompt` | `any` | `string`, `SystemPromptPreset`, or `SystemPromptFile` |
| `MaxTurns` | `*int` | Maximum conversation turns |
| `MaxThinkingTokens` | `*int` | Max tokens for thinking (deprecated, use `Thinking`) |
| `MaxBudgetUsd` | `*float64` | Maximum spend in USD |
| `Thinking` | `*ThinkingConfig` | Thinking/reasoning configuration |
| `Effort` | `*EffortLevel` | Effort level (`low`, `medium`, `high`, `max`) |
| `OutputFormat` | `*OutputFormat` | Output format with optional JSON schema |
| `Betas` | `[]string` | Beta feature flags |
| `CWD` | `string` | Working directory for the CLI |
| `Continue` | `bool` | Continue last conversation |
| `Resume` | `string` | Resume a specific session |
| `SessionID` | `string` | Use a specific session ID |
| `Tools` | `any` | `[]string` or tools preset |
| `AllowedTools` | `[]string` | Allowlisted tool names |
| `DisallowedTools` | `[]string` | Blocklisted tool names |
| `Agent` | `string` | Agent name to use |
| `Agents` | `map[string]AgentDefinition` | Custom agent definitions |
| `McpServers` | `map[string]json.RawMessage` | MCP server configurations |
| `Plugins` | `[]SdkPluginConfig` | Plugin configurations |
| `Env` | `map[string]string` | Additional environment variables |
| `PathToClaudeCode` | `string` | Custom path to the CLI binary |
| `User` | `string` | OS user to run the subprocess as |
| `Sandbox` | `*SandboxSettings` | Sandbox configuration |
| `Settings` | `any` | Settings JSON or path |
| `TaskBudget` | `*TaskBudget` | Token budget for tasks |
| `CanUseTool` | `CanUseToolFunc` | Permission callback |
| `OnElicitation` | `OnElicitationFunc` | Elicitation callback |
| `Stderr` | `func(string)` | Stderr line callback |
| `HookCallbacks` | `map[HookEvent][]HookCallbackMatcher` | Hook event callbacks |

### `Agent`

```go
type Agent struct { /* ... */ }

func NewAgent(config AgentConfig) *Agent
func (a *Agent) Run(ctx context.Context, prompt string) (*AgentResult, error)
func (a *Agent) StartSession() (*ClaudeSDKClient, error)
```

### `AgentConfig`

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Agent name |
| `Model` | `string` | Model override |
| `Prompt` | `string` | System prompt |
| `Options` | `ClaudeAgentOptions` | Full options |

### `AgentResult`

| Field | Type | Description |
|-------|------|-------------|
| `Messages` | `[]Message` | All conversation messages |
| `FinalResult` | `*ResultMessage` | The final result message |
| `CostUSD` | `float64` | Total cost in USD |
| `SessionID` | `string` | Session identifier |
| `Duration` | `time.Duration` | Wall-clock time |

### `Pipeline`

```go
func NewPipeline(name string, steps []PipelineStep) *Pipeline
func (p *Pipeline) Run(ctx context.Context, input string) (*PipelineResult, error)
```

### `ClaudeSDKClient`

```go
func NewClient(opts ClaudeAgentOptions) *ClaudeSDKClient
func (c *ClaudeSDKClient) Connect(ctx context.Context) error
func (c *ClaudeSDKClient) SendQuery(ctx context.Context, prompt string) error
func (c *ClaudeSDKClient) ReceiveMessages(ctx context.Context) <-chan Message
func (c *ClaudeSDKClient) ReceiveResponse(ctx context.Context) (*ResultMessage, error)
func (c *ClaudeSDKClient) Interrupt(ctx context.Context) error
func (c *ClaudeSDKClient) SetPermissionMode(ctx context.Context, mode PermissionMode) error
func (c *ClaudeSDKClient) SetModel(ctx context.Context, model string) error
func (c *ClaudeSDKClient) GetMCPStatus(ctx context.Context) ([]McpServerStatus, error)
func (c *ClaudeSDKClient) GetContextUsage(ctx context.Context) (*ContextUsageResponse, error)
func (c *ClaudeSDKClient) RewindFiles(ctx context.Context, messageID string) error
func (c *ClaudeSDKClient) ReconnectMcpServer(ctx context.Context, name string) error
func (c *ClaudeSDKClient) ToggleMcpServer(ctx context.Context, name string, enabled bool) error
func (c *ClaudeSDKClient) StopTask(ctx context.Context, taskID string) error
func (c *ClaudeSDKClient) GetServerInfo(ctx context.Context) (*InitializeResponse, error)
func (c *ClaudeSDKClient) Close() error
```

---

## Message Types

All messages implement the `Message` interface:

```go
type Message interface {
    MessageType() string
}
```

| Type | `MessageType()` | Description |
|------|-----------------|-------------|
| `AssistantMessage` | `"assistant"` | Model response with content blocks |
| `SystemMessage` | `"system"` | System events (init, task started/progress) |
| `ResultMessage` | `"result"` | Final result with cost and usage |
| `UserMessage` | `"user"` | User input message |
| `StreamEvent` | `"stream_event"` | Streaming events (rate limits, partials) |
| `TaskStartedMessage` | `"system"` | Task started notification |
| `TaskProgressMessage` | `"system"` | Task progress update |
| `TaskNotificationMessage` | `"system"` | Task notification |

### Content Blocks

Content blocks implement the `ContentBlock` interface:

| Type | Description |
|------|-------------|
| `TextBlock` | Plain text content |
| `ThinkingBlock` | Model reasoning/thinking |
| `ToolUseBlock` | Tool invocation with input |
| `ToolResultBlock` | Tool execution result |
| `ServerToolUseBlock` | Server-side tool use |

---

## Enums

### `PermissionMode`

`default`, `acceptEdits`, `bypassPermissions`, `plan`, `dontAsk`, `auto`

### `EffortLevel`

`low`, `medium`, `high`, `max`

### `HookEvent`

`PreToolUse`, `PostToolUse`, `Notification`, `Stop`, `SubagentStop`, `SubagentToolUse`, `ModelResponse`, `SessionStart`, `SessionEnd`, `StopFailure`, `PostCompact`, `PermissionDenied`, `Setup`, `Elicitation`, `ElicitationResult`, `ConfigChange`, `InstructionsLoaded`, `CwdChanged`, `FileChanged`, `UserPromptSubmit`

---

## Error Types

| Type | When |
|------|------|
| `CLINotFoundError` | Claude binary not found |
| `CLIConnectionError` | Pipe or process communication failure |
| `ProcessError` | CLI exited with non-zero code |
| `CLIJSONDecodeError` | Invalid JSON from CLI stdout |
| `MessageParseError` | Unexpected message structure |

---

## Environment Variable Constants

See `env.go` for the complete list. Key constants:

| Go Constant | Env Var | Purpose |
|-------------|---------|---------|
| `CLAUDE_CODE_API_KEY` | `CLAUDE_API_KEY` | API key |
| `CLAUDE_CODE_MODEL` | `ANTHROPIC_MODEL` | Default model |
| `CLAUDE_CODE_USE_BEDROCK` | `CLAUDE_CODE_USE_BEDROCK` | Enable Bedrock |
| `CLAUDE_CODE_USE_VERTEX` | `CLAUDE_CODE_USE_VERTEX` | Enable Vertex |
| `CLAUDE_CODE_DEBUG` | `CLAUDE_DEBUG` | Enable debug mode |
| `CLAUDE_CODE_MCP_TIMEOUT` | `MCP_TIMEOUT` | MCP connection timeout |

See `env.go` for 70+ additional constants covering authentication, feature flags, sandbox, proxy, telemetry, and more.

---

## Mock Transport

For testing without a real CLI:

```go
transport := claudesdk.NewMockTransport([]claudesdk.Message{
    claudesdk.QuickMockMessages("Hello from mock"),
})
```

`MockTransportFactory(messages)` returns a function compatible with internal transport creation.
