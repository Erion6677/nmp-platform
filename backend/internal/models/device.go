package models

import (
	"time"

	"gorm.io/gorm"
)

// DeviceType 设备类型枚举
type DeviceType string

const (
	DeviceTypeRouter   DeviceType = "router"
	DeviceTypeSwitch   DeviceType = "switch"
	DeviceTypeFirewall DeviceType = "firewall"
	DeviceTypeServer   DeviceType = "server"
	DeviceTypeOther    DeviceType = "other"
)

// DeviceStatus 设备状态枚举
type DeviceStatus string

const (
	DeviceStatusOnline   DeviceStatus = "online"
	DeviceStatusOffline  DeviceStatus = "offline"
	DeviceStatusUnknown  DeviceStatus = "unknown"
	DeviceStatusError    DeviceStatus = "error"
)

// DeviceOSType 设备操作系统类型枚举
type DeviceOSType string

const (
	DeviceOSTypeMikroTik DeviceOSType = "mikrotik"
	DeviceOSTypeLinux    DeviceOSType = "linux"
)

// Device 设备模型
type Device struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"not null;size:100" json:"name"`
	Type        DeviceType     `gorm:"type:varchar(20);not null" json:"type"`
	OSType      DeviceOSType   `gorm:"type:varchar(20);default:'mikrotik'" json:"os_type"` // mikrotik, linux
	Host        string         `gorm:"not null;size:255" json:"host"`
	Port        int            `gorm:"default:22" json:"port"`           // SSH 端口
	APIPort     int            `gorm:"default:8728" json:"api_port"`     // MikroTik API 端口
	Protocol    string         `gorm:"size:20;default:'ssh'" json:"protocol"` // ssh, snmp, telnet
	Username    string         `gorm:"size:100" json:"username"`
	Password    string         `gorm:"size:255" json:"-"` // 不在JSON中返回密码
	Version     string         `gorm:"size:50" json:"version"` // 设备版本
	Description string         `gorm:"size:500" json:"description"`
	Status      DeviceStatus   `gorm:"type:varchar(20);default:'unknown'" json:"status"`
	LastSeen    *time.Time     `json:"last_seen"`
	ProxyID     *uint          `gorm:"index" json:"proxy_id,omitempty"` // 代理ID
	Tags        []Tag          `gorm:"many2many:device_tags;" json:"tags"`
	Interfaces  []Interface    `json:"interfaces"`
	Groups      []DeviceGroup  `gorm:"many2many:device_group_members;" json:"groups"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (Device) TableName() string {
	return "devices"
}

// Interface 网络接口模型
type Interface struct {
	ID          uint             `gorm:"primaryKey" json:"id"`
	DeviceID    uint             `gorm:"not null;index" json:"device_id"`
	Name        string           `gorm:"not null;size:100" json:"name"`
	Status      InterfaceStatus  `gorm:"type:varchar(20);default:'unknown'" json:"status"`
	Monitored   bool             `gorm:"default:false" json:"monitored"` // 是否监控此接口
	Device      Device           `gorm:"foreignKey:DeviceID" json:"-"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	DeletedAt   gorm.DeletedAt   `gorm:"index" json:"-"`
}

// InterfaceStatus 接口状态枚举
type InterfaceStatus string

const (
	InterfaceStatusUp      InterfaceStatus = "up"
	InterfaceStatusDown    InterfaceStatus = "down"
	InterfaceStatusUnknown InterfaceStatus = "unknown"
)

// TableName 指定表名
func (Interface) TableName() string {
	return "interfaces"
}

// Tag 标签模型
type Tag struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"unique;not null;size:50" json:"name"`
	Color       string         `gorm:"size:7;default:'#007bff'" json:"color"` // 十六进制颜色值
	Description string         `gorm:"size:255" json:"description"`
	Devices     []Device       `gorm:"many2many:device_tags;" json:"-"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (Tag) TableName() string {
	return "tags"
}

// DeviceGroup 设备分组模型
type DeviceGroup struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"unique;not null;size:100" json:"name"`
	Description string         `gorm:"size:500" json:"description"`
	ParentID    *uint          `gorm:"index" json:"parent_id"` // 支持分组嵌套
	Parent      *DeviceGroup   `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children    []DeviceGroup  `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Devices     []Device       `gorm:"many2many:device_group_members;" json:"devices,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (DeviceGroup) TableName() string {
	return "device_groups"
}

// DeviceTag 设备标签关联表
type DeviceTag struct {
	DeviceID  uint      `gorm:"primaryKey" json:"device_id"`
	TagID     uint      `gorm:"primaryKey" json:"tag_id"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (DeviceTag) TableName() string {
	return "device_tags"
}

// DeviceGroupMember 设备分组成员关联表
type DeviceGroupMember struct {
	DeviceID      uint      `gorm:"primaryKey" json:"device_id"`
	DeviceGroupID uint      `gorm:"primaryKey" json:"device_group_id"`
	CreatedAt     time.Time `json:"created_at"`
}

// TableName 指定表名
func (DeviceGroupMember) TableName() string {
	return "device_group_members"
}