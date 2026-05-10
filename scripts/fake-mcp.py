"""Tiny HTTP server that mimics an MCP `tools/list` endpoint for ACP import tests."""
import http.server, json, sys

PORT = int(sys.argv[1]) if len(sys.argv) > 1 else 9090

TOOLS = {
    "tools": [
        {
            "name": "list_directory",
            "description": "List files in a directory on the local filesystem.",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "path": {"type": "string", "description": "Absolute directory path."},
                    "recursive": {"type": "boolean", "default": False},
                },
                "required": ["path"],
            },
        },
        {
            "name": "read_file",
            "description": "Read the contents of a file as UTF-8 text.",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "path": {"type": "string"},
                    "max_bytes": {"type": "integer", "default": 65536},
                },
                "required": ["path"],
            },
        },
        {
            "name": "search_repos",
            "description": "Search GitHub for repositories matching a query.",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "query": {"type": "string"},
                    "language": {"type": "string"},
                    "limit": {"type": "integer", "default": 10},
                },
                "required": ["query"],
            },
        },
        {
            "name": "send_email",
            "description": "Send an email via the configured SMTP provider.",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "to": {"type": "string"},
                    "subject": {"type": "string"},
                    "body": {"type": "string"},
                },
                "required": ["to", "subject", "body"],
            },
        },
    ]
}

class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/tools/list":
            body = json.dumps(TOOLS).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        if self.path == "/healthz":
            self.send_response(200); self.end_headers(); return
        self.send_response(404); self.end_headers()
    def log_message(self, *a, **k): pass

print(f"fake-mcp listening on :{PORT}", flush=True)
http.server.HTTPServer(("127.0.0.1", PORT), H).serve_forever()
