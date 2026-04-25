package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	headerSessionID       = "Mcp-Session-Id"
	headerProtocolVersion = "Mcp-Protocol-Version"
	defaultProtocolVer    = "2025-11-25"
)

type httpBridge struct {
	url             string
	httpClient      *http.Client
	sessionID       string
	protocolVersion string
	mu              sync.Mutex // guards stdout writes + session/protocol
	stdoutMu        sync.Mutex
}

func newHTTPBridge(url string) *httpBridge {
	return &httpBridge{
		url:        url,
		httpClient: &http.Client{Timeout: 0}, // long-lived; per-request contexts handle deadlines
	}
}

// run reads JSON-RPC frames from stdin one at a time, forwards each upstream,
// and writes the response (if any) back to stdout. Sequential by design — the
// MCP-Session-Id is captured from the initialize response and must be present
// on the next request, so we can't fire requests in parallel.
func (b *httpBridge) run(ctx context.Context) error {
	scanner := bufio.NewScanner(os.Stdin)
	// MCP messages can be large; cap at 8 MB to comfortably exceed the editor's 4254 KB limit.
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		// Copy because scanner reuses its buffer.
		msg := append([]byte(nil), line...)
		if err := b.forward(ctx, msg); err != nil {
			fmt.Fprintf(os.Stderr, "neostack-connect: forward error: %v\n", err)
			return err
		}
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		return err
	}
	// stdin closed — clean session shutdown.
	b.shutdown(ctx)
	return nil
}

func (b *httpBridge) forward(ctx context.Context, body []byte) error {
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, b.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	b.mu.Lock()
	if b.sessionID != "" {
		req.Header.Set(headerSessionID, b.sessionID)
	}
	if b.protocolVersion != "" {
		req.Header.Set(headerProtocolVersion, b.protocolVersion)
	}
	b.mu.Unlock()

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Capture session/protocol once on initialize response.
	b.mu.Lock()
	if newSession := resp.Header.Get(headerSessionID); newSession != "" && b.sessionID == "" {
		b.sessionID = newSession
	}
	if newProtocol := resp.Header.Get(headerProtocolVersion); newProtocol != "" {
		b.protocolVersion = newProtocol
	}
	b.mu.Unlock()

	// 202 Accepted = notification, no body to relay.
	if resp.StatusCode == http.StatusAccepted {
		return nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to surface JSON-RPC error responses to the client; otherwise log.
		if json.Valid(respBody) {
			return b.writeStdoutLine(respBody)
		}
		return fmt.Errorf("editor returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	if len(bytes.TrimSpace(respBody)) == 0 {
		return nil
	}
	return b.writeStdoutLine(respBody)
}

func (b *httpBridge) writeStdoutLine(body []byte) error {
	b.stdoutMu.Lock()
	defer b.stdoutMu.Unlock()
	if _, err := os.Stdout.Write(body); err != nil {
		return err
	}
	if !bytes.HasSuffix(body, []byte{'\n'}) {
		if _, err := os.Stdout.Write([]byte{'\n'}); err != nil {
			return err
		}
	}
	return nil
}

func (b *httpBridge) shutdown(parent context.Context) {
	b.mu.Lock()
	sessionID := b.sessionID
	protocol := b.protocolVersion
	b.mu.Unlock()
	if sessionID == "" {
		return
	}

	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, b.url, nil)
	if err != nil {
		return
	}
	req.Header.Set(headerSessionID, sessionID)
	if protocol != "" {
		req.Header.Set(headerProtocolVersion, protocol)
	}
	resp, err := b.httpClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}
