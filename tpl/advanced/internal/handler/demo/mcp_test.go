package demo

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMCPHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hdl := &DemoHandler{}
	e := gin.New()
	e.GET("/mcp", hdl.MCP)
	e.POST("/mcp", hdl.MCP)

	t.Run("GET /mcp returns fixed overview", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if resp["status"] != "ok" {
			t.Fatalf("expected status ok, got %v", resp["status"])
		}
		if resp["server"] != "kun-advanced-mcp-demo" {
			t.Fatalf("expected server name kun-advanced-mcp-demo, got %v", resp["server"])
		}
	})

	t.Run("POST /mcp initialize", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
		req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var resp MCPResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if resp.Error != nil {
			t.Fatalf("expected no error, got %v", resp.Error)
		}
		resultMap, ok := resp.Result.(map[string]any)
		if !ok || resultMap["protocolVersion"] != "2024-11-05" {
			t.Fatalf("unexpected result: %v", resp.Result)
		}
	})

	t.Run("POST /mcp tools/list", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`
		req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var resp MCPResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		resultMap, ok := resp.Result.(map[string]any)
		if !ok {
			t.Fatalf("expected result map, got %v", resp.Result)
		}
		tools, ok := resultMap["tools"].([]any)
		if !ok || len(tools) == 0 {
			t.Fatalf("expected tools list, got %v", resultMap)
		}
	})

	t.Run("POST /mcp tools/call returns fixed data", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_fixed_data"}}`
		req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var resp MCPResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if resp.Error != nil {
			t.Fatalf("expected no error, got %v", resp.Error)
		}
		resultMap, ok := resp.Result.(map[string]any)
		if !ok {
			t.Fatalf("expected result map, got %v", resp.Result)
		}
		contents, ok := resultMap["content"].([]any)
		if !ok || len(contents) == 0 {
			t.Fatalf("expected content, got %v", resultMap)
		}
		firstContent := contents[0].(map[string]any)
		text := firstContent["text"].(string)
		if !strings.Contains(text, "fixed_value") {
			t.Fatalf("expected text to contain fixed_value, got %s", text)
		}
	})
}
