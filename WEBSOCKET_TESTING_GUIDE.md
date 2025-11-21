# WebSocket Real-Time Streaming - Testing Guide

## Overview

This guide demonstrates how to test the WebSocket real-time streaming functionality for workflow execution monitoring.

## WebSocket Endpoint

```
GET /api/v2/workflows/runs/:runId/stream
```

This endpoint establishes a WebSocket connection to receive real-time updates for a specific workflow run.

## Message Types

The WebSocket broadcasts four types of messages:

### 1. step_start
Sent when a workflow step begins execution.

```json
{
  "runId": "550e8400-e29b-41d4-a716-446655440000",
  "type": "step_start",
  "payload": {
    "stepId": "step1",
    "stepName": "Validate Input"
  }
}
```

### 2. step_complete
Sent when a workflow step completes execution.

```json
{
  "runId": "550e8400-e29b-41d4-a716-446655440000",
  "type": "step_complete",
  "payload": {
    "stepId": "step1",
    "stepName": "Validate Input",
    "status": "success",
    "duration": 1250
  }
}
```

### 3. step_log
Sent for each log message during step execution (debug, info, warn, error).

```json
{
  "runId": "550e8400-e29b-41d4-a716-446655440000",
  "type": "step_log",
  "payload": {
    "stepId": "step1",
    "level": "info",
    "message": "Starting step: Validate Input",
    "timestamp": "2025-11-21T10:30:45.123Z"
  }
}
```

### 4. variable_change
Sent when a workflow variable is modified.

```json
{
  "runId": "550e8400-e29b-41d4-a716-446655440000",
  "type": "variable_change",
  "payload": {
    "stepId": "step1",
    "varName": "userId",
    "oldValue": null,
    "newValue": "12345",
    "changeType": "create"
  }
}
```

## Testing Methods

### Method 1: Using wscat (Command Line)

Install wscat:
```bash
npm install -g wscat
```

Connect to WebSocket:
```bash
# Start a workflow execution first to get a runId
curl -X POST http://localhost:8080/api/v2/workflows/my-workflow-id/execute

# Connect to the WebSocket (replace RUN_ID with actual runId)
wscat -c "ws://localhost:8080/api/v2/workflows/runs/RUN_ID/stream"
```

You will receive real-time messages as the workflow executes.

### Method 2: Using JavaScript (Browser Console)

```javascript
// Open browser console and run:
const runId = "YOUR_RUN_ID_HERE";
const ws = new WebSocket(`ws://localhost:8080/api/v2/workflows/runs/${runId}/stream`);

ws.onopen = () => {
  console.log('WebSocket connected');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Received:', message);

  switch(message.type) {
    case 'step_start':
      console.log(`Step ${message.payload.stepId} started`);
      break;
    case 'step_complete':
      console.log(`Step ${message.payload.stepId} completed in ${message.payload.duration}ms`);
      break;
    case 'step_log':
      console.log(`[${message.payload.level.toUpperCase()}] ${message.payload.message}`);
      break;
    case 'variable_change':
      console.log(`Variable ${message.payload.varName} changed`);
      break;
  }
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('WebSocket disconnected');
};
```

### Method 3: Using Python

```python
import asyncio
import websockets
import json

async def monitor_workflow(run_id):
    uri = f"ws://localhost:8080/api/v2/workflows/runs/{run_id}/stream"

    async with websockets.connect(uri) as websocket:
        print(f"Connected to workflow run: {run_id}")

        while True:
            try:
                message = await websocket.recv()
                data = json.loads(message)

                msg_type = data['type']
                payload = data['payload']

                if msg_type == 'step_start':
                    print(f"✓ Step started: {payload['stepName']}")
                elif msg_type == 'step_complete':
                    print(f"✓ Step completed: {payload['stepName']} ({payload['duration']}ms)")
                elif msg_type == 'step_log':
                    print(f"  [{payload['level'].upper()}] {payload['message']}")
                elif msg_type == 'variable_change':
                    print(f"  Variable {payload['varName']} = {payload['newValue']}")

            except websockets.exceptions.ConnectionClosed:
                print("Connection closed")
                break

# Usage
run_id = "YOUR_RUN_ID_HERE"
asyncio.run(monitor_workflow(run_id))
```

### Method 4: Using Go Client

```go
package main

import (
    "fmt"
    "log"
    "encoding/json"

    "github.com/gorilla/websocket"
)

type Message struct {
    RunID   string                 `json:"runId"`
    Type    string                 `json:"type"`
    Payload map[string]interface{} `json:"payload"`
}

func main() {
    runID := "YOUR_RUN_ID_HERE"
    url := fmt.Sprintf("ws://localhost:8080/api/v2/workflows/runs/%s/stream", runID)

    conn, _, err := websocket.DefaultDialer.Dial(url, nil)
    if err != nil {
        log.Fatal("Dial error:", err)
    }
    defer conn.Close()

    fmt.Printf("Connected to workflow run: %s\n", runID)

    for {
        _, message, err := conn.ReadMessage()
        if err != nil {
            log.Println("Read error:", err)
            break
        }

        var msg Message
        if err := json.Unmarshal(message, &msg); err != nil {
            log.Println("Parse error:", err)
            continue
        }

        switch msg.Type {
        case "step_start":
            fmt.Printf("✓ Step started: %s\n", msg.Payload["stepName"])
        case "step_complete":
            fmt.Printf("✓ Step completed: %s (%.0fms)\n",
                msg.Payload["stepName"], msg.Payload["duration"])
        case "step_log":
            fmt.Printf("  [%s] %s\n",
                msg.Payload["level"], msg.Payload["message"])
        case "variable_change":
            fmt.Printf("  Variable %s = %v\n",
                msg.Payload["varName"], msg.Payload["newValue"])
        }
    }
}
```

## Complete Test Scenario

### 1. Create a Test Workflow

```bash
curl -X POST http://localhost:8080/api/v2/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "workflowId": "test-websocket-workflow",
    "name": "WebSocket Test Workflow",
    "description": "Test workflow for WebSocket streaming",
    "definition": {
      "name": "Test Workflow",
      "variables": {
        "apiUrl": "https://jsonplaceholder.typicode.com/posts/1"
      },
      "steps": {
        "step1": {
          "id": "step1",
          "name": "Fetch User Data",
          "type": "http",
          "config": {
            "url": "{{apiUrl}}",
            "method": "GET"
          },
          "output": {
            "userId": "userId",
            "title": "title"
          }
        },
        "step2": {
          "id": "step2",
          "name": "Process Data",
          "type": "command",
          "dependsOn": ["step1"],
          "config": {
            "command": "echo",
            "args": ["Processing user {{userId}}"]
          }
        }
      }
    }
  }'
```

### 2. Execute the Workflow

```bash
curl -X POST http://localhost:8080/api/v2/workflows/test-websocket-workflow/execute \
  -H "Content-Type: application/json" \
  -d '{
    "variables": {
      "apiUrl": "https://jsonplaceholder.typicode.com/posts/1"
    }
  }'
```

Save the `runId` from the response.

### 3. Connect to WebSocket

```bash
wscat -c "ws://localhost:8080/api/v2/workflows/runs/<RUN_ID>/stream"
```

You should see real-time messages like:

```
< {"runId":"...","type":"step_start","payload":{"stepId":"step1","stepName":"Fetch User Data"}}
< {"runId":"...","type":"step_log","payload":{"stepId":"step1","level":"info","message":"Starting step: Fetch User Data","timestamp":"2025-11-21T10:30:45.123Z"}}
< {"runId":"...","type":"step_complete","payload":{"stepId":"step1","stepName":"Fetch User Data","status":"success","duration":245}}
< {"runId":"...","type":"step_start","payload":{"stepId":"step2","stepName":"Process Data"}}
< {"runId":"...","type":"step_complete","payload":{"stepId":"step2","stepName":"Process Data","status":"success","duration":150}}
```

## Connection Features

### Heartbeat / Ping-Pong

The WebSocket implementation includes automatic ping-pong for connection health:
- Server sends PING every 54 seconds
- Client must respond with PONG within 60 seconds
- Connection auto-closes if client doesn't respond

### Multiple Clients

Multiple clients can connect to the same runId simultaneously. Each will receive all broadcast messages.

### Automatic Cleanup

When a workflow run completes, clients can disconnect. The hub automatically cleans up closed connections.

## Troubleshooting

### Connection Refused
- Ensure the server is running on the correct port
- Check CORS settings if connecting from browser
- Verify the WebSocket handler is registered in main.go

### No Messages Received
- Verify the runId exists and is active
- Check that the hub is running: `go hub.Run()` should be called
- Ensure the WorkflowExecutor was initialized with the hub

### Connection Drops
- Check network connectivity
- Verify ping-pong responses are being sent
- Review server logs for errors

## Production Considerations

### Security

Update the CheckOrigin function in `websocket_handler.go`:

```go
var upgrader = gorillaws.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        // In production, restrict to allowed origins
        origin := r.Header.Get("Origin")
        return origin == "https://yourdomain.com"
    },
}
```

### Authentication

Add authentication middleware before the WebSocket upgrade:

```go
api.GET("/workflows/runs/:runId/stream", authMiddleware(), h.StreamWorkflowRun)
```

### Rate Limiting

Implement connection limits per user/IP to prevent abuse.

### Monitoring

Add metrics for:
- Active WebSocket connections
- Message broadcast rate
- Connection duration
- Error rates
