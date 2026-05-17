#!/usr/bin/env python3
"""Mock MCP HTTP server for manual E2E testing of acp-bridge."""
import http.server
import json
import sys

TOOLS = {
    "github": [
        {"name": "issues_list", "description": "List GitHub issues for a repository with filtering and pagination",
         "inputSchema": {"type": "object", "properties": {
             "owner": {"type": "string", "description": "The account owner of the repository"},
             "repo": {"type": "string", "description": "The name of the repository without the .git extension"},
             "state": {"type": "string", "enum": ["open", "closed", "all"], "description": "State filter"},
             "labels": {"type": "array", "items": {"type": "string"}, "description": "List of label names"}
         }, "required": ["owner", "repo"]}},
        {"name": "issues_create", "description": "Create a new issue in a GitHub repository",
         "inputSchema": {"type": "object", "properties": {
             "owner": {"type": "string", "description": "The account owner of the repository"},
             "repo": {"type": "string", "description": "The name of the repository"},
             "title": {"type": "string", "description": "The title of the issue"},
             "body": {"type": "string", "description": "The contents of the issue"},
             "labels": {"type": "array", "items": {"type": "string"}}
         }, "required": ["owner", "repo", "title"]}},
        {"name": "pull_request_list", "description": "List pull requests",
         "inputSchema": {"type": "object", "properties": {
             "owner": {"type": "string"}, "repo": {"type": "string"},
             "state": {"type": "string", "enum": ["open", "closed", "all"]}
         }, "required": ["owner", "repo"]}},
    ],
    "flyctl": [
        {"name": "apps_list", "description": "List all Fly.io applications in an organization",
         "inputSchema": {"type": "object", "properties": {
             "org": {"type": "string", "description": "Organization slug"}
         }}},
        {"name": "deploy", "description": "Deploy an application to Fly.io from a Docker image",
         "inputSchema": {"type": "object", "properties": {
             "app": {"type": "string", "description": "Application name"},
             "image": {"type": "string", "description": "Docker image reference"},
             "region": {"type": "string", "description": "Target region"}
         }, "required": ["app", "image"]}},
        {"name": "machine_list", "description": "List machines for a Fly.io app",
         "inputSchema": {"type": "object", "properties": {
             "app": {"type": "string"}
         }, "required": ["app"]}},
    ],
    "database": [
        {"name": "query", "description": "Execute a SQL query against the connected database with optional parameters",
         "inputSchema": {"type": "object", "properties": {
             "sql": {"type": "string", "description": "SQL query to execute"},
             "limit": {"type": "integer", "description": "Maximum number of rows to return"},
             "params": {"type": "array", "items": {"type": "string"}, "description": "Query parameters"}
         }, "required": ["sql"]}},
        {"name": "tables_list", "description": "List all tables in a database schema",
         "inputSchema": {"type": "object", "properties": {
             "schema": {"type": "string", "description": "Schema name, defaults to public"}
         }}},
    ],
}

PORT_MAP = {"github": 9001, "flyctl": 9002, "database": 9003}


class MCPHandler(http.server.BaseHTTPRequestHandler):
    source_name = "unknown"

    def do_GET(self):
        if self.path.rstrip("/") == "/tools/list":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps({"tools": TOOLS[self.source_name]}).encode())
        else:
            self.send_response(404)
            self.end_headers()

    def log_message(self, format, *args):
        sys.stderr.write(f"[{self.source_name}] {format % args}\n")


def run_server(source_name):
    port = PORT_MAP[source_name]

    class Handler(MCPHandler):
        pass
    Handler.source_name = source_name

    server = http.server.HTTPServer(("127.0.0.1", port), Handler)
    sys.stderr.write(f"Mock MCP server '{source_name}' on http://127.0.0.1:{port}\n")
    server.serve_forever()


if __name__ == "__main__":
    import threading
    for name in TOOLS:
        t = threading.Thread(target=run_server, args=(name,), daemon=True)
        t.start()
    sys.stderr.write(f"\nAll mock servers running. Press Ctrl+C to stop.\n\n")
    try:
        threading.Event().wait()
    except KeyboardInterrupt:
        sys.stderr.write("\nStopping.\n")
