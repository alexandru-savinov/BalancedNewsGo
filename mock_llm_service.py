import sys
from http.server import BaseHTTPRequestHandler, HTTPServer
import json

class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        response = {
            "score": self.server.score,
            "label": self.server.label
        }
        self.send_response(200)
        self.send_header('Content-type', 'application/json')
        self.end_headers()
        self.wfile.write(json.dumps(response).encode())

def run(label, port):
    if label == "left":
        score = -1.0
    elif label == "center":
        score = 0.0
    elif label == "right":
        score = 1.0
    else:
        score = 0.0

    server = HTTPServer(('', port), Handler)
    server.label = label
    server.score = score
    print(f"Starting mock LLM service for {label} on port {port}...")
    server.serve_forever()

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python mock_llm_service.py <label> <port>")
        sys.exit(1)
    label = sys.argv[1]
    port = int(sys.argv[2])
    run(label, port)