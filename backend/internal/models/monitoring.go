package models

import (
	"time"

	"gorm.io/gorm"
)

// ProxyType 代理类型枚举
type ProxyType string

const (
	ProxyTypeSSH    ProxyType = "ssh"
	ProxyTypeSOCKS5 ProxyType = "socks5"
)

// ProxyStatus 代理状态枚举
type ProxyStatus string

const (
	ProxyStatusConnected    ProxyStatus = "connected"
	ProxyStatusDisconnected ProxyStatus = "disconnected"
	ProxyStatusError        ProxyStatus = "error"
)

// Proxy 代理模型
type Proxy struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	Name           string         `gorm:"not null;size:100" json:"name"`
	Type           ProxyType      `gorm:"type:varchar(20);not null" json:"type"`
	
	// SSH 代理字段
	SSHHost        string         `gorm:"size:255" json:"ssh_host,omitempty"`
	SSHPort        int            `gorm:"default:22" json:"ssh_port,omitempty"`
	SSHUsername    string         `gorm:"size:100" json:"ssh_username,omitempty"`
	SSHPassword    string         `gorm:"size:255" json:"-"`
	
	// SOCKS5 代理字段
	SOCKS5Host     string         `gorm:"size:255" json:"socks5_host,omitempty"`
	SOCKS5Port     int            `gorm:"default:1080" json:"socks5_port,omitempty"`
	SOCKS5Username string         `gorm:"size:100" json:"socks5_username,omitempty"`
	SOCKS5Password string         `gorm:"size:255" json:"-"`
	
	// 链式代理
	ParentProxyID  *uint          `gorm:"index" json:"parent_proxy_id,omitempty"`
	ParentProxy    *Proxy         `gorm:"foreignKey:ParentProxyID" json:"parent_proxy,omitempty"`
	
	Enabled        bool           `gorm:"default:true" json:"enabled"`
	Status         ProxyStatus    `gorm:"type:varchar(20);default:'disconnected'" json:"status"`
	LastError      string         `gorm:"type:text" json:"last_error,omitempty"`
	
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (Proxy) TableName() string {
	return "proxies"
}

// CollectorStatus 采集器状态枚举
type CollectorStatus string

const (
	CollectorStatusNotDeployed CollectorStatus = "not_deployed"
	CollectorStatusDeployed    CollectorStatus = "deployed"
	CollectorStatusRunning     CollectorStatus = "running"
	CollectorStatusStopped     CollectorStatus = "stopped"
	CollectorStatusError       CollectorStatus = "error"
)

// CollectorScript 采集器配置模型
type CollectorScript struct {
	ID             uint            `gorm:"primaryKey" json:"id"`
	DeviceID       uint            `gorm:"not null;uniqueIndex" json:"device_id"`
	Device         Device          `gorm:"foreignKey:DeviceID" json:"-"`
	
	Enabled        bool            `gorm:"default:false" json:"enabled"`
	IntervalMs     int             `gorm:"default:1000" json:"interval_ms"`      // 采集间隔（毫秒）
	PushBatchSize  int             `gorm:"default:10" json:"push_batch_size"`    // 批量推送数量
	ScriptName     string          `gorm:"size:64;default:'nmp-collector'" json:"script_name"`
	SchedulerName  string          `gorm:"size:64;default:'nmp-scheduler'" json:"scheduler_name"`
	
	DeployedAt     *time.Time      `json:"deployed_at,omitempty"`
	LastPushAt     *time.Time      `json:"last_push_at,omitempty"`
	PushCount      int64           `gorm:"default:0" json:"push_count"`
	
	Status         CollectorStatus `gorm:"type:varchar(32);default:'not_deployed'" json:"status"`
	ErrorMessage   string          `gorm:"type:text" json:"error_message,omitempty"`
	
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// TableName 指定表名
func (CollectorScript) TableName() string {
	return "collector_scripts"
}

// PingTarget Ping 目标模型
type PingTarget struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	DeviceID        uint           `gorm:"not null;index" json:"device_id"`
	Device          Device         `gorm:"foreignKey:DeviceID" json:"-"`
	
	TargetAddress   string         `gorm:"not null;size:255" json:"target_address"`
	TargetName      string         `gorm:"not null;size:100" json:"target_name"`
	SourceInterface string         `gorm:"size:100;default:''" json:"source_interface"`
	Enabled         bool           `gorm:"default:true" json:"enabled"`
	
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (PingTarget) TableName() string {
	return "ping_targets"
}

// SystemSetting 系统设置模型
type SystemSetting struct {
	Key         string    `gorm:"primaryKey;size:100" json:"key"`
	Value       string    `gorm:"type:text;not null" json:"value"`
	Description string    `gorm:"type:text" json:"description,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名
func (SystemSetting) TableName() string {
	return "system_settings"
}

// UserDevicePermission 用户设备权限关联表
type UserDevicePermission struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"-"`
	DeviceID  uint      `gorm:"not null;index" json:"device_id"`
	Device    Device    `gorm:"foreignKey:DeviceID" json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (UserDevicePermission) TableName() string {
	return "user_device_permissions"
}

// BeforeCreate 创建前设置唯一约束
func (u *UserDevicePermission) BeforeCreate(tx *gorm.DB) error {
	// 检查是否已存在相同的用户-设备权限
	var count int64
	tx.Model(&UserDevicePermission{}).Where("user_id = ? AND device_id = ?", u.UserID, u.DeviceID).Count(&count)
	if count > 0 {
		return gorm.ErrDuplicatedKey
	}
	return nil
}
