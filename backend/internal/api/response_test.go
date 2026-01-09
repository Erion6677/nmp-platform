package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Property 6: API 响应格式一致性
// Feature: nmp-bugfix-iteration, Property 6: API 响应格式一致性
// Validates: Requirements 7.1, 7.3, 7.4
// ============================================================================

// APIResponse 统一 API 响应结构
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// TestAPIResponseFormat_Success_Property 属性测试：成功响应格式一致性
// *For any* 后端 API 成功响应，必须包含 `success: true` 且包含 `data` 字段
func TestAPIResponseFormat_Success_Property(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 测试 Success 函数
	t.Run("Success函数返回正确格式", func(t *testing.T) {
		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			Success(c, gin.H{"key": "value"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		// 验证 success 字段为 true
		assert.True(t, response.Success, "成功响应必须包含 success: true")

		// 验证包含 data 字段
		assert.NotNil(t, response.Data, "成功响应必须包含 data 字段")
	})

	// 测试 SuccessWithMessage 函数
	t.Run("SuccessWithMessage函数返回正确格式", func(t *testing.T) {
		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			SuccessWithMessage(c, gin.H{"key": "value"}, "操作成功")
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		// 验证 success 字段为 true
		assert.True(t, response.Success, "成功响应必须包含 success: true")

		// 验证包含 data 字段
		assert.NotNil(t, response.Data, "成功响应必须包含 data 字段")

		// 验证包含 message 字段
		assert.Equal(t, "操作成功", response.Message, "应包含正确的消息")
	})

	// 测试 SuccessPaginated 函数
	t.Run("SuccessPaginated函数返回正确格式", func(t *testing.T) {
		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			items := []string{"item1", "item2"}
			SuccessPaginated(c, items, 100, 1, 10)
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		// 验证 success 字段为 true
		assert.True(t, response["success"].(bool), "成功响应必须包含 success: true")

		// 验证包含 data 字段
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok, "成功响应必须包含 data 字段")

		// 验证分页数据结构
		assert.NotNil(t, data["items"], "分页响应必须包含 items")
		assert.NotNil(t, data["total"], "分页响应必须包含 total")
		assert.NotNil(t, data["page"], "分页响应必须包含 page")
		assert.NotNil(t, data["page_size"], "分页响应必须包含 page_size")
		assert.NotNil(t, data["total_pages"], "分页响应必须包含 total_pages")
	})
}

// TestAPIResponseFormat_Error_Property 属性测试：错误响应格式一致性
// *For any* 后端 API 错误响应，必须包含 `success: false` 且包含 `error` 字符串字段
func TestAPIResponseFormat_Error_Property(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 测试各种错误响应函数
	errorFunctions := []struct {
		name       string
		handler    func(c *gin.Context)
		statusCode int
	}{
		{
			name: "BadRequest",
			handler: func(c *gin.Context) {
				BadRequest(c, "请求参数错误")
			},
			statusCode: http.StatusBadRequest,
		},
		{
			name: "Unauthorized",
			handler: func(c *gin.Context) {
				Unauthorized(c, "未授权访问")
			},
			statusCode: http.StatusUnauthorized,
		},
		{
			name: "Forbidden",
			handler: func(c *gin.Context) {
				Forbidden(c, "权限不足")
			},
			statusCode: http.StatusForbidden,
		},
		{
			name: "NotFound",
			handler: func(c *gin.Context) {
				NotFound(c, "资源不存在")
			},
			statusCode: http.StatusNotFound,
		},
		{
			name: "Conflict",
			handler: func(c *gin.Context) {
				Conflict(c, "资源冲突")
			},
			statusCode: http.StatusConflict,
		},
		{
			name: "InternalError",
			handler: func(c *gin.Context) {
				InternalError(c, "服务器内部错误")
			},
			statusCode: http.StatusInternalServerError,
		},
		{
			name: "Error",
			handler: func(c *gin.Context) {
				Error(c, http.StatusBadGateway, "网关错误")
			},
			statusCode: http.StatusBadGateway,
		},
	}

	for _, ef := range errorFunctions {
		t.Run(ef.name+"函数返回正确格式", func(t *testing.T) {
			router := gin.New()
			router.GET("/test", ef.handler)

			req, _ := http.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// 验证状态码
			assert.Equal(t, ef.statusCode, w.Code, "应返回正确的 HTTP 状态码")

			var response APIResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			// 验证 success 字段为 false
			assert.False(t, response.Success, "错误响应必须包含 success: false")

			// 验证包含 error 字段
			assert.NotEmpty(t, response.Error, "错误响应必须包含 error 字段")
		})
	}
}

// TestAPIResponseFormat_ErrorWithDetails_Property 属性测试：带详情的错误响应格式
func TestAPIResponseFormat_ErrorWithDetails_Property(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		ErrorWithDetails(c, http.StatusBadRequest, "验证失败", "字段 name 不能为空")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// 验证 success 字段为 false
	assert.False(t, response["success"].(bool), "错误响应必须包含 success: false")

	// 验证包含 error 字段
	assert.NotEmpty(t, response["error"], "错误响应必须包含 error 字段")

	// 验证包含 details 字段
	assert.NotEmpty(t, response["details"], "带详情的错误响应应包含 details 字段")
}

// TestAPIResponseFormat_NilData_Property 属性测试：空数据响应格式
func TestAPIResponseFormat_NilData_Property(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		SuccessWithMessage(c, nil, "操作成功")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// 验证 success 字段为 true
	assert.True(t, response["success"].(bool), "成功响应必须包含 success: true")

	// 验证包含 message 字段
	assert.Equal(t, "操作成功", response["message"], "应包含正确的消息")
}

// TestAPIResponseFormat_StructResponse_Property 属性测试：结构体响应格式
func TestAPIResponseFormat_StructResponse_Property(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type TestData struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		Success(c, TestData{ID: 1, Name: "测试"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// 验证 success 字段为 true
	assert.True(t, response["success"].(bool), "成功响应必须包含 success: true")

	// 验证 data 字段包含正确的结构
	data, ok := response["data"].(map[string]interface{})
	assert.True(t, ok, "data 应该是一个对象")
	assert.Equal(t, float64(1), data["id"])
	assert.Equal(t, "测试", data["name"])
}

// TestAPIResponseFormat_ArrayResponse_Property 属性测试：数组响应格式
func TestAPIResponseFormat_ArrayResponse_Property(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		Success(c, []string{"item1", "item2", "item3"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// 验证 success 字段为 true
	assert.True(t, response["success"].(bool), "成功响应必须包含 success: true")

	// 验证 data 字段是数组
	data, ok := response["data"].([]interface{})
	assert.True(t, ok, "data 应该是一个数组")
	assert.Len(t, data, 3)
}

// TestNewErrorResponse_Property 属性测试：ErrorResponse 结构体
func TestNewErrorResponse_Property(t *testing.T) {
	// 测试 NewErrorResponse
	t.Run("NewErrorResponse创建正确格式", func(t *testing.T) {
		resp := NewErrorResponse("测试错误")

		assert.False(t, resp.Success, "错误响应 Success 应为 false")
		assert.Equal(t, "测试错误", resp.Message)
		assert.Equal(t, "测试错误", resp.Error)
	})

	// 测试 NewErrorResponseWithCode
	t.Run("NewErrorResponseWithCode创建正确格式", func(t *testing.T) {
		resp := NewErrorResponseWithCode(400, "参数错误")

		assert.False(t, resp.Success, "错误响应 Success 应为 false")
		assert.Equal(t, 400, resp.Code)
		assert.Equal(t, "参数错误", resp.Message)
		assert.Equal(t, "参数错误", resp.Error)
	})
}

// TestNewSuccessResponse_Property 属性测试：SuccessResponse 结构体
func TestNewSuccessResponse_Property(t *testing.T) {
	// 测试 NewSuccessResponse
	t.Run("NewSuccessResponse创建正确格式", func(t *testing.T) {
		resp := NewSuccessResponse(gin.H{"key": "value"})

		assert.True(t, resp.Success, "成功响应 Success 应为 true")
		assert.NotNil(t, resp.Data)
	})

	// 测试 NewSuccessResponseWithMessage
	t.Run("NewSuccessResponseWithMessage创建正确格式", func(t *testing.T) {
		resp := NewSuccessResponseWithMessage("操作成功", gin.H{"key": "value"})

		assert.True(t, resp.Success, "成功响应 Success 应为 true")
		assert.Equal(t, "操作成功", resp.Message)
		assert.NotNil(t, resp.Data)
	})
}

// TestPaginatedResponse_Property 属性测试：分页响应计算
func TestPaginatedResponse_Property(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		total          int64
		page           int
		pageSize       int
		expectedPages  int
	}{
		{"整除情况", 100, 1, 10, 10},
		{"有余数情况", 101, 1, 10, 11},
		{"单页情况", 5, 1, 10, 1},
		{"空数据情况", 0, 1, 10, 0},
		{"大页码情况", 1000, 5, 20, 50},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/test", func(c *gin.Context) {
				SuccessPaginated(c, []string{}, tc.total, tc.page, tc.pageSize)
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			data := response["data"].(map[string]interface{})
			assert.Equal(t, float64(tc.expectedPages), data["total_pages"], "总页数计算应正确")
			assert.Equal(t, float64(tc.total), data["total"], "总数应正确")
			assert.Equal(t, float64(tc.page), data["page"], "当前页应正确")
			assert.Equal(t, float64(tc.pageSize), data["page_size"], "页大小应正确")
		})
	}
}
