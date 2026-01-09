package system_admin

import (
	"gorm.io/gorm"

	"nmp-platform/internal/auth"
	"nmp-platform/internal/plugin"
)

// RegisterPlugin 注册系统管理插件
func RegisterPlugin(manager *plugin.Manager, db *gorm.DB, authService *auth.AuthService) error {
	systemAdminPlugin := NewSystemAdminPlugin(db, authService)
	return manager.RegisterPlugin(systemAdminPlugin)
}