package api

import (
	"net/http"
	"strconv"

	"nmp-platform/internal/models"
	"nmp-platform/internal/service"

	"github.com/gin-gonic/gin"
)

// TagHandler 标签处理器
type TagHandler struct {
	tagService service.TagService
}

// NewTagHandler 创建新的标签处理器
func NewTagHandler(tagService service.TagService) *TagHandler {
	return &TagHandler{
		tagService: tagService,
	}
}

// CreateTagRequest 创建标签请求
type CreateTagRequest struct {
	Name        string `json:"name" binding:"required"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// UpdateTagRequest 更新标签请求
type UpdateTagRequest struct {
	Name        string `json:"name" binding:"required"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// ListTagsRequest 标签列表请求
type ListTagsRequest struct {
	Page     int `form:"page,default=1"`
	PageSize int `form:"page_size,default=20"`
}

// CreateTag 创建标签
func (h *TagHandler) CreateTag(c *gin.Context) {
	var req CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// 创建标签对象
	tag := &models.Tag{
		Name:        req.Name,
		Color:       req.Color,
		Description: req.Description,
	}

	// 设置默认颜色
	if tag.Color == "" {
		tag.Color = "#007bff"
	}

	err := h.tagService.CreateTag(tag)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    tag,
	})
}

// GetTag 获取标签详情
func (h *TagHandler) GetTag(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid tag ID",
		})
		return
	}

	tag, err := h.tagService.GetTag(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tag,
	})
}

// UpdateTag 更新标签
func (h *TagHandler) UpdateTag(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid tag ID",
		})
		return
	}

	var req UpdateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// 获取现有标签
	tag, err := h.tagService.GetTag(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 更新标签信息
	tag.Name = req.Name
	tag.Color = req.Color
	tag.Description = req.Description

	// 设置默认颜色
	if tag.Color == "" {
		tag.Color = "#007bff"
	}

	err = h.tagService.UpdateTag(tag)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tag,
	})
}

// DeleteTag 删除标签
func (h *TagHandler) DeleteTag(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid tag ID",
		})
		return
	}

	err = h.tagService.DeleteTag(uint(id))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Tag deleted successfully",
	})
}

// ListTags 获取标签列表
func (h *TagHandler) ListTags(c *gin.Context) {
	var req ListTagsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	// 计算偏移量
	offset := (req.Page - 1) * req.PageSize
	if offset < 0 {
		offset = 0
	}

	tags, total, err := h.tagService.ListTags(offset, req.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 计算分页信息
	totalPages := (int(total) + req.PageSize - 1) / req.PageSize

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"tags":        tags,
			"total":       total,
			"page":        req.Page,
			"page_size":   req.PageSize,
			"total_pages": totalPages,
		},
	})
}

// GetAllTags 获取所有标签（不分页）
func (h *TagHandler) GetAllTags(c *gin.Context) {
	tags, err := h.tagService.GetAllTags()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tags,
	})
}

// RegisterRoutes 注册标签相关路由
func (h *TagHandler) RegisterRoutes(router *gin.RouterGroup) {
	tags := router.Group("/tags")
	{
		tags.POST("", h.CreateTag)
		tags.GET("", h.ListTags)
		tags.GET("/all", h.GetAllTags)
		tags.GET("/:id", h.GetTag)
		tags.PUT("/:id", h.UpdateTag)
		tags.DELETE("/:id", h.DeleteTag)
	}
}