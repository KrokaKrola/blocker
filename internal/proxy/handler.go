package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/user/blocker/internal/blocker"
)

// Handler handles proxy requests
type Handler struct {
	blocker   *blocker.Blocker
	transport *http.Transport
}

// NewHandler creates a new proxy handler
func NewHandler(b *blocker.Blocker) *Handler {
	return &Handler{
		blocker: b,
		transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

// ServeHTTP handles incoming proxy requests
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		h.handleConnect(w, r)
		return
	}
	h.handleHTTP(w, r)
}

// handleHTTP handles regular HTTP requests
func (h *Handler) handleHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract host
	host := r.Host
	if host == "" {
		host = r.URL.Host
	}

	// Check if blocked
	if h.blocker.IsBlocked(host) {
		h.serveBlocked(w, r)
		return
	}

	// Create outgoing request
	outReq := new(http.Request)
	*outReq = *r
	outReq.RequestURI = ""

	// Ensure URL is absolute
	if outReq.URL.Scheme == "" {
		outReq.URL.Scheme = "http"
	}
	if outReq.URL.Host == "" {
		outReq.URL.Host = host
	}

	// Remove hop-by-hop headers
	removeHopHeaders(outReq.Header)

	// Forward the request
	resp, err := h.transport.RoundTrip(outReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Proxy error: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	removeHopHeaders(resp.Header)
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Write status code and body
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// handleConnect handles HTTPS CONNECT requests
func (h *Handler) handleConnect(w http.ResponseWriter, r *http.Request) {
	host := r.Host

	// Check if blocked
	if h.blocker.IsBlocked(host) {
		http.Error(w, "Blocked", http.StatusForbidden)
		return
	}

	// Connect to destination
	destConn, err := net.DialTimeout("tcp", host, 30*time.Second)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to connect: %v", err), http.StatusBadGateway)
		return
	}

	// Hijack the connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		destConn.Close()
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, fmt.Sprintf("Hijack failed: %v", err), http.StatusInternalServerError)
		destConn.Close()
		return
	}

	// Send 200 Connection Established
	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// Tunnel data between client and destination
	go transfer(destConn, clientConn)
	go transfer(clientConn, destConn)
}

// serveBlocked returns a blocked response
func (h *Handler) serveBlocked(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
    <title>Blocked</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: #f5f5f5;
        }
        .container {
            text-align: center;
            padding: 40px;
            background: white;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 { color: #e74c3c; margin-bottom: 10px; }
        p { color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Access Blocked</h1>
        <p>This website has been blocked by Network Blocker.</p>
    </div>
</body>
</html>`))
}

// transfer copies data from src to dst and closes both when done
func transfer(dst io.WriteCloser, src io.ReadCloser) {
	defer dst.Close()
	defer src.Close()
	io.Copy(dst, src)
}

// Hop-by-hop headers that should be removed
var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

// removeHopHeaders removes hop-by-hop headers
func removeHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
}
