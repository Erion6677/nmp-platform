package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestAppError(t *testing.T) {
	t.Run("创建基本应用错误", func(t *testing.T) {
		err := NewAppError(ErrCodeValidationFailed, "测试错误", http.StatusBadRequest)
		
		assert.Equal(t, ErrCodeValidationFailed, err.Code)
		assert.Equal(t, "测试错误", err.Message)
		assert.Equal(t, http.StatusBadRequest, err.StatusCode)
		assert.Nil(t, err.Cause)
		assert.Empty(t, err.Details)
	})

	t.Run("创建带原因的应用错误", func(t *testing.T) {
		cause := errors.New("原始错误")
		err := NewAppErrorWithCause(ErrCodeInternalError, "包装错误", http.StatusInternalServerError, cause)
		
		assert.Equal(t, ErrCodeInternalError, err.Code)
		assert.Equal(t, "包装错误", err.Message)
		assert.Equal(t, http.StatusInternalServerError, err.StatusCode)
		assert.Equal(t, cause, err.Cause)
	})

	t.Run("创建带详细信息的应用错误", func(t *testing.T) {
		err := NewAppErrorWithDetails(ErrCodeValidationFailed, "验证失败", "字段不能为空", http.StatusBadRequest)
		
		assert.Equal(t, ErrCodeValidationFailed, err.Code)
		assert.Equal(t, "验证失败", err.Message)
		assert.Equal(t, "字段不能为空", err.Details)
		assert.Equal(t, http.StatusBadRequest, err.StatusCode)
	})

	t.Run("错误字符串表示", func(t *testing.T) {
		// 无原因错误
		err1 := NewAppError(ErrCodeNotFound, "资源未找到", http.StatusNotFound)
		assert.Equal(t, "NOT_FOUND: 资源未找到", err1.Error())

		// 有原因错误
		cause := errors.New("数据库连接失败")
		err2 := NewAppErrorWithCause(ErrCodeInternalError, "内部错误", http.StatusInternalServerError, cause)
		assert.Equal(t, "INTERNAL_ERROR: 内部错误 (caused by: 数据库连接失败)", err2.Error())
	})

	t.Run("错误解包", func(t *testing.T) {
		cause := errors.New("原始错误")
		err := NewAppErrorWithCause(ErrCodeInternalError, "包装错误", http.StatusInternalServerError, cause)
		
		assert.Equal(t, cause, errors.Unwrap(err))
	})
}

func TestErrorHandlerMiddleware(t *testing.T) {
	// 设置测试环境
	gin.SetMode(gin.TestMode)
	logger := zaptest.NewLogger(t)

	t.Run("处理应用错误", func(t *testing.T) {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("request_id", "test-request-id")
		})
		router.Use(ErrorHandlerMiddleware(logger))
		
		router.GET("/test", func(c *gin.Context) {
			err := NewAppErrorWithDetails(ErrCodeValidationFailed, "验证失败", "用户名不能为空", http.StatusBadRequest)
			c.Error(err)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, ErrCodeValidationFailed, response.Error)
		assert.Equal(t, "验证失败", response.Message)
		assert.Equal(t, "用户名不能为空", response.Details)
		assert.Equal(t, "test-request-id", response.RequestID)
		assert.NotZero(t, response.Timestamp)
	})

	t.Run("处理绑定错误", func(t *testing.T) {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("request_id", "test-request-id")
		})
		router.Use(ErrorHandlerMiddleware(logger))
		
		router.POST("/test", func(c *gin.Context) {
			var data struct {
				Name string `json:"name" binding:"required"`
			}
			if err := c.ShouldBindJSON(&data); err != nil {
				c.Error(err)
			}
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(`{"invalid": "json"}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, ErrCodeValidationFailed, response.Error)
		assert.Equal(t, "Request validation failed", response.Message)
		assert.NotEmpty(t, response.Details)
	})

	t.Run("处理未知错误", func(t *testing.T) {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("request_id", "test-request-id")
		})
		router.Use(ErrorHandlerMiddleware(logger))
		
		router.GET("/test", func(c *gin.Context) {
			c.Error(errors.New("未知错误"))
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, ErrCodeInternalError, response.Error)
		assert.Equal(t, "An unexpected error occurred", response.Message)
	})

	t.Run("无错误时不处理", func(t *testing.T) {
		router := gin.New()
		router.Use(ErrorHandlerMiddleware(logger))
		
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "success", response["message"])
	})
}

func TestAbortWithError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zaptest.NewLogger(t)

	t.Run("AbortWithError", func(t *testing.T) {
		router := gin.New()
		router.Use(ErrorHandlerMiddleware(logger))
		
		router.GET("/test", func(c *gin.Context) {
			err := NewAppError(ErrCodeUnauthorized, "未授权访问", http.StatusUnauthorized)
			AbortWithError(c, err)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("AbortWithAppError", func(t *testing.T) {
		router := gin.New()
		router.Use(ErrorHandlerMiddleware(logger))
		
		router.GET("/test", func(c *gin.Context) {
			AbortWithAppError(c, ErrCodeForbidden, "访问被拒绝", http.StatusForbidden)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("AbortWithValidationError", func(t *testing.T) {
		router := gin.New()
		router.Use(ErrorHandlerMiddleware(logger))
		
		router.GET("/test", func(c *gin.Context) {
			AbortWithValidationError(c, "参数验证失败")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "参数验证失败", response.Details)
	})

	t.Run("AbortWithNotFoundError", func(t *testing.T) {
		router := gin.New()
		router.Use(ErrorHandlerMiddleware(logger))
		
		router.GET("/test", func(c *gin.Context) {
			AbortWithNotFoundError(c, "用户")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "用户 not found", response.Message)
	})

	t.Run("AbortWithInternalError", func(t *testing.T) {
		router := gin.New()
		router.Use(ErrorHandlerMiddleware(logger))
		
		router.GET("/test", func(c *gin.Context) {
			cause := errors.New("数据库连接失败")
			AbortWithInternalError(c, cause)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestIsBindingError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "绑定错误",
			err:      errors.New("binding failed"),
			expected: true,
		},
		{
			name:     "验证错误",
			err:      errors.New("validation error"),
			expected: true,
		},
		{
			name:     "必填字段错误",
			err:      errors.New("field is required"),
			expected: true,
		},
		{
			name:     "无效输入错误",
			err:      errors.New("invalid input"),
			expected: true,
		},
		{
			name:     "其他错误",
			err:      errors.New("database connection failed"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBindingError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "包含子字符串",
			s:        "this is a test",
			substr:   "test",
			expected: true,
		},
		{
			name:     "不包含子字符串",
			s:        "this is a test",
			substr:   "hello",
			expected: false,
		},
		{
			name:     "完全匹配",
			s:        "test",
			substr:   "test",
			expected: true,
		},
		{
			name:     "空子字符串",
			s:        "test",
			substr:   "",
			expected: true,
		},
		{
			name:     "空字符串",
			s:        "",
			substr:   "test",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetCurrentTimestamp(t *testing.T) {
	// 保存原始函数
	originalGetCurrentTime := getCurrentTime
	defer func() {
		getCurrentTime = originalGetCurrentTime
	}()

	// Mock时间
	mockTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	getCurrentTime = func() time.Time {
		return mockTime
	}

	timestamp := getCurrentTimestamp()
	expected := mockTime.Unix()
	
	assert.Equal(t, expected, timestamp)
}

// 基准测试
func BenchmarkErrorHandling(b *testing.B) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	
	router := gin.New()
	router.Use(ErrorHandlerMiddleware(logger))
	router.GET("/test", func(c *gin.Context) {
		err := NewAppError(ErrCodeValidationFailed, "测试错误", http.StatusBadRequest)
		c.Error(err)
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
	}
}