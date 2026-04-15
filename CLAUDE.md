# Claude Code Agent ———— GO SDK

## Overview

This is the project for the Claude Code agent, which is designed to interact with the Claude API using Go.

This is an SDK project which is similar to the official https://github.com/anthropics/claude-agent-sdk-python.git .

## Your Targets

1. take full understanding of the official Python SDK, and then implement the same functionality in Go.
2. the python sdk code is under: ./python-sdk ,which is ignored by .gitignore.
3. the go sdk package name is 'github.com/Esonhugh/goClaudeAgentSDK', the go.mod is next to the CLAUDE.md file.
4. understand the current code structure of the project 
5. keep the same code structure as the python sdk
6. keep your code clean and well-documented, following Go best practices.
7. write unit tests for your code to ensure its reliability and correctness.
8. write down the README.md file for the project, which should include:
   - An introduction to the project and its purpose.
   - Instructions on how to install and use the SDK.
   - Examples of how to interact with the Claude API using the SDK.
   - Any other relevant information that users might find helpful.
9. write the docs/ directory for the project, which should include:
   - Detailed documentation of the SDK's functionality and usage.
   - API reference for the SDK's functions and methods.
   - Any other relevant information that users might find helpful.
10. by the way, claude code source code is under ../
11. make every environment variable configurable, and provide a way to set them easily, and provide/exported them with `const CLAUDE_CODE_XXXXX = "CLAUDE_CODE_XXXXX"` in the code. 
12. claude code leaked source code is under ./claude-code, which is ignored by .gitignore, you can refer it for the implementation details, but do not copy the code directly, you should write your own code based on your understanding of the python sdk and the leaked source code.
13. for 11., you can add more claude code mentioned environment variables, and provide a way to set them easily, such as a bundle of constants which same as the environment variable names.


## Tasks

1. [x] Core types (types.go) - Message types, options, MCP configs, content blocks
2. [x] Transport layer (transport.go, subprocess_transport.go) - Subprocess communication
3. [x] Message parser (message_parser.go) - JSON to typed messages
4. [x] Error hierarchy (errors.go) - SDK error types
5. [x] Query handler (query.go) - Control protocol, message routing
6. [x] Client (client.go) - Bidirectional streaming interface
7. [x] Agent (agent.go) - High-level agent abstraction
8. [x] Pipeline (pipeline.go) - Sequential agent composition
9. [x] Parallel (parallel.go) - Concurrent agent execution
10. [x] Mock transport (mock_transport.go) - Test infrastructure
11. [x] Examples (examples/) - quickstart, pipeline, multiagent, streaming, calculator, codereview, tooluse
12. [x] Environment variable constants (env.go) - 70+ `const CLAUDE_CODE_XXX` for all env vars (API, auth, provider, session, features, model, sandbox, MCP, bash, proxy, telemetry)
13. [x] Missing types (types.go) - TaskStartedMessage, TaskProgressMessage, TaskNotificationMessage, StreamEvent, RateLimitInfo (full), ContextUsage, TaskBudget, SdkBeta, SystemPromptPreset, HookInput types, HookSpecificOutput types
14. [x] Enhanced message parser (message_parser.go) - Handle task_started/task_progress/task_notification system subtypes, stream_event, forward-compatible nil for unknown types
15. [x] Hook callback routing (query.go) - Wire hook_callback control requests, route to HookCallbackMatcher, serialize HookJSONOutput with hookSpecificOutput
16. [x] Session management (sessions.go) - ListSessions, GetSessionInfo, GetSessionMessages, RenameSession, TagSession, DeleteSession, ForkSession
17. [x] Client enhancements (client.go) - RewindFiles, ReconnectMcpServer, ToggleMcpServer, StopTask, GetContextUsage, GetServerInfo
18. [x] Complete buildArgs (subprocess_transport.go) - All options map to CLI flags including --task-budget, --plugin-dir, --system-prompt, --settings, --sandbox, --tools, --json-schema
19. [x] Comprehensive tests - sessions_test.go, transport_test.go, sdk_test.go, agent_test.go, pipeline_test.go, parallel_test.go
20. [x] README.md - Installation, usage, examples, configuration reference, project structure
21. [x] docs/ directory - API reference (api-reference.md), architecture (architecture.md), hook system guide (hooks.md), session management guide (sessions.md)
22. [x] Wire Tools option in buildArgs - Tools field now maps to --tools CLI flag
23. [x] Wire User option in subprocess - SysProcAttr sets UID/GID on Unix when User is specified
24. [x] Read CLAUDE_CODE_STREAM_CLOSE_TIMEOUT - Transport reads timeout from env/opts for init handshake
25. [x] Handle control_cancel_request in query.go - Acknowledge cancellation requests from CLI
26. [x] Fix --json-schema handling - OutputFormat.Schema uses --json-schema, fallback to --output-format-json
27. [x] Add CLI search paths - npm-global, .local/bin, node_modules, .yarn/bin

