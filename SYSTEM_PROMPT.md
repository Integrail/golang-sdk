# Integrail Golang SDK

## Project Overview

The **Integrail Golang SDK** is a comprehensive toolkit for building backend-as-a-service (BaaS) applications with a focus on real-time communication, AI/ML integration, observability, and system execution capabilities. This SDK provides essential building blocks for modern cloud-native applications.

## Project Structure

```
github.com/integrail/golang-sdk/
├── pkg/
│   ├── exec/           # Host command execution utilities
│   ├── llm/            # Large Language Model integrations
│   ├── mongodb/        # MongoDB-based services
│   │   └── messagebus/ # Real-time message broker
│   └── observatory/    # Logging and observability
├── .github/workflows/  # CI/CD automation
├── go.mod             # Go module definition
├── welder.yaml        # Build configuration
└── tools.go           # Development tools
```

## Core Components

### 1. MongoDB MessageBus (`pkg/mongodb/messagebus/`)

A generic, type-safe message broker implementation using MongoDB change streams for real-time communication.

**Key Features:**
- **Generic Implementation**: Uses Go generics (`Broker[T any]`) for type-safe message handling
- **Real-time Communication**: Leverages MongoDB change streams for instant message delivery
- **Session-based Messaging**: Messages are organized by session IDs for isolated communication channels
- **Automatic Migrations**: Embedded database migrations using `golang-migrate`
- **Persistent Storage**: Messages are stored in MongoDB collections for reliability

**Core Interface:**
```go
type Broker[T any] interface {
    Send(ctx context.Context, sessionID string, evt T) error
    OnMessage(ctx context.Context, sessionID string, onSubStarted func(), onMsgCallback func(evt T)) error
}
```

**Implementation Details:**
- Uses MongoDB's `Watch()` API to monitor collection changes
- Filters for insert operations only
- JSON serialization/deserialization for message payload
- Configurable database and collection names
- Automatic connection management with cleanup

### 2. LLM Integration (`pkg/llm/`)

Unified interface for multiple Large Language Model providers with consistent API.

**Supported Providers:**
- **Azure OpenAI**: Enterprise-grade OpenAI integration with Azure AD authentication
- **OpenAI**: Direct OpenAI API integration
- **Ollama**: Local LLM deployment support

**Core Interface:**
```go
type Client interface {
    Generate(ctx context.Context, request GenerateRequest) (*GenerateResponse, error)
}
```

**Features:**
- Unified request/response structures across providers
- Configurable retry logic and cooldown periods
- Comprehensive generation metadata
- Support for both API key and Azure AD authentication
- Temperature and token limit controls

### 3. Observatory (`pkg/observatory/`)

Advanced logging and observability system with buffered log shipping to external services.

**Key Features:**
- **Buffered Logging**: Collects logs in memory buffers before batch shipping
- **Multi-tenant Support**: Separate buffers per execution context
- **Automatic Flushing**: Time-based and size-based buffer flushing
- **External Integration**: Ships logs to Everworker observatory endpoints
- **Context-aware**: Extracts execution context from HTTP headers

**Components:**
- `ObservatorySink`: Main logging sink with buffering capabilities
- `WithExecutionCtx()`: Context enrichment from HTTP requests
- Structured log levels (Error, Warn, Info)
- Bearer token authentication for log shipping

### 4. Exec (`pkg/exec/`)

Host command execution utilities with advanced process management.

**Features:**
- **Context-aware Execution**: Respects Go context cancellation
- **Signal Proxying**: Forwards signals between parent and child processes
- **Output Capture**: Configurable output handling (capture vs. proxy)
- **Environment Management**: Custom environment variable injection
- **Working Directory Control**: Execute commands in specific directories

**Execution Modes:**
- `ExecCommand()`: Capture output and return as string
- `ProxyExec()`: Stream I/O directly to parent process
- Signal handling for graceful process termination

## Build System & Tooling

### Welder Configuration (`welder.yaml`)

The project uses **Welder** as its primary build system with the following configuration:

- **Project Name**: `baas` (Backend-as-a-Service)
- **Multi-module Support**: Supports both `golang-sdk` and `baas-cli` modules
- **Automated Tooling**: Installs and manages development tools
- **Linting Pipeline**: Integrated `golangci-lint` and `gofumpt` formatting
- **Release Automation**: Automatic Git tagging and deployment

### Development Tools (`tools.go`)

Managed development dependencies:
- **mockery**: Mock generation for testing
- **gofumpt**: Advanced Go code formatting
- **golangci-lint**: Comprehensive Go linting

### CI/CD Pipeline (`.github/workflows/push.yaml`)

Automated build and release pipeline:
- **Calendar Versioning**: Uses CalVer for version management
- **Automated Testing**: Runs full test suite on push
- **Release Automation**: Creates Git tags and releases
- **Welder Integration**: Uses Welder for build orchestration

## Dependencies & Technology Stack

### Core Dependencies
- **MongoDB Driver**: `go.mongodb.org/mongo-driver` for database operations
- **Migration Support**: `github.com/golang-migrate/migrate/v4` for schema management
- **Error Handling**: `github.com/pkg/errors` for enhanced error context
- **Azure Integration**: Azure SDK for authentication and services
- **OpenAI SDKs**: Official OpenAI and Azure OpenAI Go clients
- **Ollama**: Local LLM deployment support

### Development Dependencies
- **Testing**: `github.com/onsi/gomega` for BDD-style testing
- **Testcontainers**: `github.com/testcontainers/testcontainers-go` for integration testing
- **Utilities**: `github.com/samber/lo` for functional programming utilities
- **AWS Lambda**: `github.com/simple-container-com/go-aws-lambda-sdk` for serverless support

## Key Design Patterns

### 1. Generic Programming
Extensive use of Go generics for type-safe APIs, particularly in the MessageBus implementation.

### 2. Interface-driven Design
Clear separation of concerns through well-defined interfaces (Client, Broker, etc.).

### 3. Context Propagation
Consistent use of Go contexts for cancellation, timeouts, and value propagation.

### 4. Error Wrapping
Comprehensive error context using `github.com/pkg/errors` for better debugging.

### 5. Embedded Resources
Use of `//go:embed` for packaging migrations and static resources.

## Testing Strategy

- **Unit Tests**: Individual component testing with mocks
- **Integration Tests**: Real database and service integration using Testcontainers
- **BDD Testing**: Behavior-driven development with Gomega assertions
- **Mock Generation**: Automated mock generation with Mockery

## Usage Patterns

### MessageBus Example
```go
// Create a message bus for custom event types
broker, err := messagebus.NewMessageBus[MyEvent](ctx, logger, mongoURI, "events")

// Send a message
err = broker.Send(ctx, "session-123", MyEvent{Data: "example"})

// Listen for messages
err = broker.OnMessage(ctx, "session-123", 
    func() { /* subscription started */ },
    func(evt MyEvent) { /* handle message */ })
```

### LLM Integration Example
```go
// Create Azure OpenAI client
client, err := llm.NewAzureOpenAI(logger, endpoint, apiKey, model, apiVersion)

// Generate content
response, err := client.Generate(ctx, llm.GenerateRequest{
    Prompt: "Your prompt here",
    Model:  "gpt-4",
})
```

## Project Goals

1. **Modularity**: Provide reusable components for BaaS applications
2. **Type Safety**: Leverage Go's type system for compile-time guarantees
3. **Observability**: Built-in logging and monitoring capabilities
4. **Cloud Native**: Designed for containerized and serverless environments
5. **Developer Experience**: Comprehensive tooling and clear APIs

## Contributing

The project follows standard Go conventions with additional tooling:
- Run `welder run fmt` for code formatting
- Run `welder run linters` for code quality checks
- Use embedded migrations for database schema changes
- Follow the established interface patterns for new components

This SDK serves as the foundation for building scalable, observable, and maintainable backend services with modern Go practices and comprehensive tooling support.
