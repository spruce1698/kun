package demo

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

// MCP (Model Context Protocol) JSON-RPC 2.0 数据结构定义

type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type MCPResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *MCPError `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type MCPTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type MCPToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type MCPToolCallResult struct {
	Content []MCPToolContent `json:"content"`
	IsError bool             `json:"isError,omitempty"`
}

// MCP Demo 处理器: 支持标准 JSON-RPC 2.0 协议调用, 也支持直接 GET 查看固定演示值
// @Summary MCP Demo
// @Description 模型上下文协议(MCP)端点，直接返回固定数据
// @Tags api
// @Accept json
// @Produce json
// @Router /api/demo/mcp [post]
func (d *DemoHandler) MCP(ctx *gin.Context) {
	// 1. GET 请求直接返回固定信息与说明文档
	if ctx.Request.Method == http.MethodGet {
		ctx.JSON(http.StatusOK, gin.H{
			"status":   "ok",
			"server":   "kun-advanced-mcp-demo",
			"version":  "1.0.0",
			"protocol": "Model Context Protocol (JSON-RPC 2.0)",
			"tools": []string{
				"get_fixed_data",
			},
			"fixed_response": gin.H{
				"id":          1001,
				"name":        "MCP Fixed Demo",
				"description": "This is a direct fixed demo value from kun-advanced.",
				"tags":        []string{"mcp", "agent", "kun"},
			},
		})
		return
	}

	// 2. POST 处理 JSON-RPC 2.0 请求
	var req MCPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, MCPResponse{
			JSONRPC: "2.0",
			ID:      nil,
			Error: &MCPError{
				Code:    -32700,
				Message: "Parse error: " + err.Error(),
			},
		})
		return
	}

	resp := MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	case "initialize":
		// 握手初始化，返回服务元信息与能力
		resp.Result = gin.H{
			"protocolVersion": "2024-11-05",
			"capabilities": gin.H{
				"tools": gin.H{},
			},
			"serverInfo": gin.H{
				"name":    "kun-advanced-mcp-demo",
				"version": "1.0.0",
			},
		}

	case "notifications/initialized":
		// 握手完成通知
		resp.Result = gin.H{}

	case "ping":
		// 心跳探活
		resp.Result = gin.H{}

	case "tools/list":
		// 列出可用工具
		resp.Result = gin.H{
			"tools": []MCPTool{
				{
					Name:        "get_fixed_data",
					Description: "获取 MCP 固定演示数据",
					InputSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"query": map[string]any{
								"type":        "string",
								"description": "可选查询参数",
							},
						},
					},
				},
			},
		}

	case "tools/call":
		// 工具调用：直接返回固定值
		fixedJSON := `{"code":0,"message":"success","data":{"id":1001,"title":"kun-mcp-demo","fixed_value":"Hello! This is a fixed value returned by Kun MCP Demo.","author":"spruce"}}`
		resp.Result = MCPToolCallResult{
			Content: []MCPToolContent{
				{
					Type: "text",
					Text: fixedJSON,
				},
			},
			IsError: false,
		}

	default:
		// 未知方法统一返回友好固定值或标准错误
		resp.Error = &MCPError{
			Code:    -32601,
			Message: "Method not found: " + req.Method,
		}
	}

	ctx.JSON(http.StatusOK, resp)
}
