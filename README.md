# MCP Server for Claude - Ready to Deploy

This is a working MCP (Model Context Protocol) server that integrates with Claude.

## Quick Deploy to Railway

1. Go to: https://railway.app/new
2. Choose "Deploy from GitHub repo" 
3. Or choose "Empty Service" and upload this folder
4. Railway will auto-detect Go and deploy

## Test Your Deployment

Once deployed, get your URL and test:

```bash
# Should return server info
curl https://YOUR-URL/

# Should return health status
curl https://YOUR-URL/health

# Should return MCP protocol info
curl https://YOUR-URL/mcp
```

## Connect to Claude

1. Go to claude.ai → Settings → Feature Preview → Model Context Protocol
2. Add Server:
   - Name: My MCP Server
   - URL: https://YOUR-URL/mcp
   - Authentication: None

## Available Tools

- `system_info` - Get system information
- `echo` - Echo back a message

## Files

- `main.go` - Complete MCP server implementation
- `go.mod` - Go module file (no dependencies needed)