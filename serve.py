from http.server import BaseHTTPRequestHandler, HTTPServer

PORT = 54821
KEY_URL = (
    "https://raw.githubusercontent.com/kissanjamgit/private_stream/main/key/enc.key"
)

pathToKey = "/enc.key"


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == pathToKey:
            self.send_response(302)
            self.send_header("Location", KEY_URL)
            self.end_headers()
        else:
            self.send_response(404)
            self.end_headers()


server = HTTPServer(("127.0.0.1", PORT), Handler)

print("Server running on port", PORT)
print("http://127.0.0.1:" + str(PORT) + pathToKey)

try:
    server.serve_forever()
except KeyboardInterrupt:
    print("\nStopping server...")
    server.server_close()
