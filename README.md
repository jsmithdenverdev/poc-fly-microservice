# POC Fly Microservice

A Go microservice prototype demonstrating deployment on Fly.io with scale-to-zero capabilities.

## Features

- Auto-shutdown after configurable amount of inactivity
- Scale-to-zero configuration for service

## Prerequisites

- Go 1.24 or later
- Docker and Docker Compose
- [Task](https://taskfile.dev) for development automation
- Fly.io CLI for deployment

## Development

Start the development environment:

```bash
task dev
```

### Available Tasks

```bash
task                 # Show available tasks
task dev             # Start development environment
task dev:down        # Stop development environment
task dev:clean       # Clean development environment (removes volumes)
task health          # Check service health
task create:item     # Create a new item (requires name argument)
task list:items      # List all items
```

### Example Usage

```bash
# Start the development environment
task dev

# In another terminal:
# Create a new item
task create:item -- "Test Item"

# List all items
task list:items

# Check service health
task health
```

### Debugging

The application runs with Delve debugger enabled. You can connect to it using your IDE:

#### VS Code

Add the following configuration to your `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug app",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/app",
      "env": {
        "APP_HOST": "localhost",
        "APP_PORT": "8080",
        "ENABLE_INACTIVITY_TIMEOUT": "true",
        "INACTIVITY_TIMEOUT": "5",
        "FLY_REGION": "nyc1"
      }
    }
  ]
}
```

#### GoLand

1. Go to Run -> Edit Configurations
2. Add New Configuration -> Go Remote
3. Set the following:
   - Host: `localhost`
   - Port: `40000`
   - Path mappings:
     - Local: `$PROJECT_DIR$`
     - Remote: `/app`

## Architecture

```mermaid
graph TB
    subgraph Docker Build Stages
        base[Base Stage<br/>golang:1.23.4-alpine<br/>+ git]

        dev[Development Stage<br/>+ Delve debugger<br/>+ bash]
        prod[Production Stage<br/>Build binary]
        final[Final Stage<br/>scratch image]

        base --> dev
        base --> prod
        prod --> final
    end

    subgraph Runtime Architecture
        client[Client]

        subgraph Fly.io Platform
            app[Go Microservice<br/>Auto-shutdown]

            app -- "Tracks activity" --> app
        end

        client -- "HTTP Requests" --> app

        subgraph Local Development
            debug[Debug Mode<br/>:40000]
            vscode[VS Code]

            vscode -- "Remote Debug" --> debug
        end
    end

    style base fill:#f9f,stroke:#333
    style dev fill:#bbf,stroke:#333
    style prod fill:#bfb,stroke:#333
    style final fill:#fbb,stroke:#333
    style app fill:#9cf,stroke:#333
```

## API Endpoints

- `GET /health` - Health check endpoint
- `POST /items` - Create a new item
- `GET /items/` - List all items

### Request Examples

Create Item:

```json
POST /items
{
    "name": "Example Item"
}
```

## Local Development

The development environment uses Docker Compose with hot-reload capabilities. The service will automatically rebuild when changes are detected.

## Deployment

```bash
fly deploy
```
