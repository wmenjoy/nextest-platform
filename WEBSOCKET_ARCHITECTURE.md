# WebSocket Architecture Diagram

## System Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Client Applications                             │
│  (Browser, CLI, Mobile App, etc.)                                       │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │ WebSocket Connection
                                │ ws://host/api/v2/workflows/runs/:runId/stream
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         WebSocket Handler                                │
│  internal/handler/websocket_handler.go                                  │
│                                                                          │
│  - Upgrades HTTP to WebSocket                                           │
│  - Creates Client instance                                              │
│  - Registers with Hub                                                   │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           WebSocket Hub                                  │
│  internal/websocket/hub.go                                              │
│                                                                          │
│  ┌──────────────────────────────────────────────────────────┐          │
│  │  Clients Map:                                             │          │
│  │  {                                                        │          │
│  │    "runId-1": [client1, client2, client3],              │          │
│  │    "runId-2": [client4],                                │          │
│  │    ...                                                   │          │
│  │  }                                                        │          │
│  └──────────────────────────────────────────────────────────┘          │
│                                                                          │
│  Channels:                                                               │
│  • broadcast chan *Message   (256 buffer)                               │
│  • register chan *Client                                                │
│  • unregister chan *Client                                              │
└───────────────┬────────────────────────────────┬────────────────────────┘
                │                                │
                │ Broadcasts                     │ Manages
                ▼                                ▼
┌──────────────────────────┐      ┌──────────────────────────────────┐
│   WebSocket Client 1     │      │      WebSocket Client 2          │
│   internal/websocket/    │      │      internal/websocket/         │
│   client.go              │      │      client.go                   │
│                          │      │                                  │
│  • ReadPump (goroutine)  │      │  • ReadPump (goroutine)         │
│  • WritePump (goroutine) │      │  • WritePump (goroutine)        │
│  • send chan (256 buffer)│      │  • send chan (256 buffer)       │
└──────────────────────────┘      └──────────────────────────────────┘
                ▲                                 ▲
                │                                 │
                └─────────────┬───────────────────┘
                              │
                     Messages broadcast here
                              │
┌─────────────────────────────┴─────────────────────────────────────────┐
│                       Workflow Executor                                │
│  internal/workflow/executor.go                                         │
│                                                                        │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  Execute(workflowID, workflowDef) {                          │   │
│  │    1. Create WorkflowRun                                      │   │
│  │    2. Initialize ExecutionContext with BroadcastStepLogger   │   │
│  │    3. Build DAG                                               │   │
│  │    4. For each layer:                                         │   │
│  │         For each step (parallel):                             │   │
│  │           hub.Broadcast(runID, "step_start", {...})          │   │
│  │           Execute step                                        │   │
│  │           logger.Info/Warn/Error (broadcasts via hub)        │   │
│  │           hub.Broadcast(runID, "step_complete", {...})       │   │
│  │    5. Finalize run                                            │   │
│  │  }                                                             │   │
│  └──────────────────────────────────────────────────────────────┘   │
└─────────────────────────────┬──────────────────────────────────────────┘
                              │
                              │ Uses
                              ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      Broadcast Step Logger                               │
│  internal/workflow/broadcast_logger.go                                  │
│                                                                          │
│  log(level, stepID, message) {                                          │
│    1. Save to database (WorkflowStepLog)                                │
│    2. Broadcast to WebSocket:                                           │
│         hub.Broadcast(runID, "step_log", {                              │
│           stepId, level, message, timestamp                             │
│         })                                                               │
│    3. Print to console                                                  │
│  }                                                                       │
└─────────────────────────────────────────────────────────────────────────┘
```

## Message Flow

### 1. Client Connection Flow
```
Client                WebSocketHandler              Hub
  |                          |                       |
  |---(1) HTTP GET -------->|                       |
  |     /runs/:runId/stream |                       |
  |                          |                       |
  |<--(2) Upgrade to WS-----|                       |
  |                          |                       |
  |                          |---(3) Register ------>|
  |                          |       Client          |
  |                          |                       |
  |                          |---(4) Start---------->|
  |                          |   ReadPump/WritePump  |
  |                          |                       |
  |<--(5) Ready to receive messages------------------|
```

### 2. Workflow Execution Flow
```
WorkflowExecutor    BroadcastLogger      Hub         Client
     |                    |               |             |
     |---(1) executeStep->|               |             |
     |                    |               |             |
     |---(2) Broadcast--->|-------------->|             |
     |    "step_start"    |               |             |
     |                    |               |----(3)----->|
     |                    |               |  Message    |
     |                    |               |             |
     |---(4) logger.Info->|               |             |
     |                    |               |             |
     |                    |---(5) Broadcast------------>|
     |                    |    "step_log" |             |
     |                    |               |             |
     |---(6) Step complete|               |             |
     |                    |               |             |
     |---(7) Broadcast------------------->|             |
     |    "step_complete" |               |             |
     |                    |               |----(8)----->|
     |                    |               |  Message    |
```

### 3. Broadcast Distribution Flow
```
Hub.Broadcast(runID, type, payload)
     |
     |---[broadcast chan]---> Hub.Run() goroutine
                                   |
                                   |--Get clients for runID
                                   |
                                   +--> Client 1 send chan
                                   |
                                   +--> Client 2 send chan
                                   |
                                   +--> Client 3 send chan
                                        |
                                        |--[WritePump]-->
                                        |
                                        +--JSON encode-->
                                        |
                                        +--WebSocket frame-->
                                        |
                                        +--Network--> Browser/CLI
```

## Component Responsibilities

### WebSocket Hub
- **Single Instance**: One hub per application
- **Thread-Safe**: Uses mutexes for client map access
- **Connection Manager**: Tracks all active connections by runID
- **Message Broker**: Distributes messages to relevant clients
- **Lifecycle**: Runs continuously in background goroutine

### WebSocket Client
- **Per Connection**: One client instance per WebSocket connection
- **Dual Goroutines**: ReadPump (incoming) + WritePump (outgoing)
- **Buffered**: 256-message send buffer for performance
- **Auto-Cleanup**: Unregisters on disconnect
- **Heartbeat**: Ping-pong every 54s to detect dead connections

### Broadcast Step Logger
- **Triple Output**: Database + WebSocket + Console
- **Non-Blocking**: Uses buffered channels
- **Real-Time**: Immediate broadcast on log call
- **Persistent**: Database storage for history
- **Structured**: JSON payload format

### Workflow Executor
- **Integration Point**: Owns hub reference
- **Event Source**: Broadcasts step lifecycle events
- **Context Provider**: Creates logger with hub
- **Parallel Execution**: Multiple steps can broadcast simultaneously

## Concurrency Model

```
Main Goroutine
  ├─> hub.Run() goroutine (manages broadcast distribution)
  │
  └─> HTTP Server goroutines
        ├─> WebSocket Handler (per connection)
        │     ├─> Client.ReadPump() goroutine
        │     └─> Client.WritePump() goroutine
        │
        └─> Workflow Execution (per workflow run)
              ├─> Layer 1 steps (parallel goroutines)
              ├─> Layer 2 steps (parallel goroutines)
              └─> Layer N steps (parallel goroutines)
                    └─> Each broadcasts to hub
```

## Threading Safety

### Protected by Mutex
- Hub.clients map (read/write)
- Hub client registration/unregistration

### Channel-Based Communication
- Hub.broadcast (thread-safe by design)
- Hub.register (thread-safe by design)
- Hub.unregister (thread-safe by design)
- Client.send (thread-safe by design)

### No Synchronization Needed
- Message struct (immutable after creation)
- Client struct fields (accessed only by owner goroutines)

## Performance Characteristics

### Message Throughput
- **Hub broadcast channel**: 256-message buffer
- **Client send channel**: 256-message buffer
- **Total capacity per runID**: 256 + (256 × num_clients) messages

### Latency
- **In-memory**: Sub-millisecond broadcast to hub
- **Network**: Depends on WebSocket connection
- **Batching**: WritePump batches queued messages for efficiency

### Resource Usage
- **Memory per client**: ~512 bytes (buffers) + connection overhead
- **Goroutines per client**: 2 (ReadPump + WritePump)
- **Hub goroutines**: 1 (Run loop)

## Error Handling

### Connection Failures
```
Client disconnect
  ├─> ReadPump detects error
  ├─> Calls hub.Unregister(client)
  ├─> Hub removes from clients map
  ├─> Closes client.send channel
  └─> WritePump detects closed channel and exits
```

### Slow Clients
```
Hub tries to send message
  ├─> select with default case
  ├─> If send buffer full, skip message (non-blocking)
  ├─> Close client connection
  └─> Unregister from hub
```

### Network Errors
```
WritePump write error
  ├─> Log error
  ├─> Close connection
  └─> ReadPump detects close and unregisters
```

## Testing Strategy

### Unit Tests
- Mock hub for executor tests
- Test client registration/unregistration
- Test message broadcasting
- Test concurrent access

### Integration Tests
- Real WebSocket connections
- Multiple clients per runID
- Parallel workflow execution
- Message ordering verification

### Load Tests
- 1000+ concurrent connections
- High-frequency message broadcasting
- Memory leak detection
- Connection stability over time

## Deployment Checklist

- [ ] Configure CORS for production origins
- [ ] Add authentication middleware
- [ ] Set up monitoring for active connections
- [ ] Configure rate limiting
- [ ] Enable TLS/WSS
- [ ] Set up alerts for error rates
- [ ] Test with production-like load
- [ ] Document WebSocket endpoint in API docs
- [ ] Configure reverse proxy (nginx/traefik) for WebSocket
- [ ] Set up logging for WebSocket events
