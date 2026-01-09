package models

import (
	"time"

	"gorm.io/gorm"
)

// UserStatus 用户状态枚举
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusBlocked  UserStatus = "blocked"
)

// User 用户模型
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"unique;not null;size:50" json:"username"`
	Password  string         `gorm:"not null;size:255" json:"-"` // 不在JSON中返回密码
	Email     string         `gorm:"unique;size:100" json:"email"`
	FullName  string         `gorm:"size:100" json:"full_name"`
	Status    UserStatus     `gorm:"type:varchar(20);default:'active'" json:"status"`
	LastLogin *time.Time     `json:"last_login"`
	Roles     []Role         `gorm:"many2many:user_roles;" json:"roles"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// Role 角色模型
type Role struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"unique;not null;size:50" json:"name"`
	DisplayName string         `gorm:"size:100" json:"display_name"`
	Description string         `gorm:"size:255" json:"description"`
	IsSystem    bool           `gorm:"default:false" json:"is_system"` // 系统角色不可删除
	Users       []User         `gorm:"many2many:user_roles;" json:"-"`
	Permissions []Permission   `gorm:"many2many:role_permissions;" json:"permissions"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (Role) TableName() string {
	return "roles"
}

// Permission 权限模型
type Permission struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Resource    string         `gorm:"not null;size:50" json:"resource"`    // 资源类型，如 device, user, plugin
	Action      string         `gorm:"not null;size:50" json:"action"`      // 操作类型，如 create, read, update, delete
	Scope       string         `gorm:"size:50" json:"scope"`                // 权限范围，如 own, group, all
	Description string         `gorm:"size:255" json:"description"`
	Roles       []Role         `gorm:"many2many:role_permissions;" json:"-"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (Permission) TableName() string {
	return "permissions"
}

// UserRole 用户角色关联表
type UserRole struct {
	UserID    uint      `gorm:"primaryKey" json:"user_id"`
	RoleID    uint      `gorm:"primaryKey" json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (UserRole) TableName() string {
	return "user_roles"
}

// RolePermission 角色权限关联表
type RolePermission struct {
	RoleID       uint      `gorm:"primaryKey" json:"role_id"`
	PermissionID uint      `gorm:"primaryKey" json:"permission_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// TableName 指定表名
func (RolePermission) TableName() string {
	return "role_permissions"
}