package models

import (
	"time"

	"gorm.io/gorm"
)

// PluginStatus 插件状态枚举
type PluginStatus string

const (
	PluginStatusEnabled  PluginStatus = "enabled"
	PluginStatusDisabled PluginStatus = "disabled"
	PluginStatusError    PluginStatus = "error"
)

// Plugin 插件模型
type Plugin struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"unique;not null;size:100" json:"name"`
	Version     string         `gorm:"not null;size:20" json:"version"`
	DisplayName string         `gorm:"size:100" json:"display_name"`
	Description string         `gorm:"size:500" json:"description"`
	Author      string         `gorm:"size:100" json:"author"`
	Homepage    string         `gorm:"size:255" json:"homepage"`
	Status      PluginStatus   `gorm:"type:varchar(20);default:'disabled'" json:"status"`
	Config      string         `gorm:"type:text" json:"config"` // JSON格式的插件配置
	FilePath    string         `gorm:"size:500" json:"file_path"` // 插件文件路径
	Checksum    string         `gorm:"size:64" json:"checksum"`   // 文件校验和
	LoadOrder   int            `gorm:"default:0" json:"load_order"` // 加载顺序
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (Plugin) TableName() string {
	return "plugins"
}

// PluginRoute 插件路由模型
type PluginRoute struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	PluginID   uint           `gorm:"not null;index" json:"plugin_id"`
	Method     string         `gorm:"not null;size:10" json:"method"` // GET, POST, PUT, DELETE
	Path       string         `gorm:"not null;size:255" json:"path"`
	Handler    string         `gorm:"not null;size:255" json:"handler"` // 处理函数名
	Permission string         `gorm:"size:100" json:"permission"` // 所需权限
	Plugin     Plugin         `gorm:"foreignKey:PluginID" json:"-"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (PluginRoute) TableName() string {
	return "plugin_routes"
}

// PluginMenu 插件菜单模型
type PluginMenu struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	PluginID   uint           `gorm:"not null;index" json:"plugin_id"`
	Key        string         `gorm:"not null;size:100" json:"key"`
	Label      string         `gorm:"not null;size:100" json:"label"`
	Icon       string         `gorm:"size:50" json:"icon"`
	Path       string         `gorm:"size:255" json:"path"`
	ParentID   *uint          `gorm:"index" json:"parent_id"`
	Order      int            `gorm:"default:0" json:"order"`
	Permission string         `gorm:"size:100" json:"permission"` // 所需权限
	Plugin     Plugin         `gorm:"foreignKey:PluginID" json:"-"`
	Parent     *PluginMenu    `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children   []PluginMenu   `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (PluginMenu) TableName() string {
	return "plugin_menus"
}

// SystemConfig 系统配置模型
type SystemConfig struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Key         string         `gorm:"unique;not null;size:100" json:"key"`
	Value       string         `gorm:"type:text" json:"value"`
	Type        string         `gorm:"size:20;default:'string'" json:"type"` // string, number, boolean, json
	Category    string         `gorm:"size:50" json:"category"` // system, plugin, user
	Description string         `gorm:"size:255" json:"description"`
	IsPublic    bool           `gorm:"default:false" json:"is_public"` // 是否对前端公开
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (SystemConfig) TableName() string {
	return "system_configs"
}