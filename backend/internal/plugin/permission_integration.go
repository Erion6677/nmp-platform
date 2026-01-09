package plugin

import (
	"fmt"
	"log"
	"strings"

	"nmp-platform/internal/auth"
	"nmp-platform/internal/models"
)

// PermissionIntegrator 插件权限集成器
type PermissionIntegrator struct {
	rbacService *auth.RBACService
	logger      *log.Logger
}

// NewPermissionIntegrator 创建权限集成器
func NewPermissionIntegrator(rbacService *auth.RBACService, logger *log.Logger) *PermissionIntegrator {
	return &PermissionIntegrator{
		rbacService: rbacService,
		logger:      logger,
	}
}

// IntegratePluginPermissions 集成插件权限
func (pi *PermissionIntegrator) IntegratePluginPermissions(plugin Plugin) error {
	pluginName := plugin.Name()
	permissions := plugin.GetPermissions()
	
	if len(permissions) == 0 {
		pi.logger.Printf("Plugin %s has no permissions to integrate", pluginName)
		return nil
	}
	
	pi.logger.Printf("Integrating %d permissions for plugin %s", len(permissions), pluginName)
	
	for _, perm := range permissions {
		if err := pi.integratePermission(pluginName, perm); err != nil {
			pi.logger.Printf("Failed to integrate permission %s:%s for plugin %s: %v", 
				perm.Resource, perm.Action, pluginName, err)
			continue
		}
	}
	
	return nil
}

// integratePermission 集成单个权限
func (pi *PermissionIntegrator) integratePermission(pluginName string, perm Permission) error {
	// 为插件权限添加前缀，避免冲突
	resource := fmt.Sprintf("plugin.%s.%s", pluginName, perm.Resource)
	action := perm.Action
	scope := perm.Scope
	if scope == "" {
		scope = "all"
	}
	
	description := perm.Description
	if description == "" {
		description = fmt.Sprintf("Plugin %s permission: %s %s", pluginName, action, perm.Resource)
	}
	
	// 创建权限
	_, err := pi.rbacService.CreatePermission(resource, action, scope, description)
	if err != nil {
		// 如果权限已存在，不视为错误
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "already exists") {
			pi.logger.Printf("Permission %s:%s already exists, skipping", resource, action)
			return nil
		}
		return fmt.Errorf("failed to create permission: %w", err)
	}
	
	pi.logger.Printf("Created permission: %s:%s for plugin %s", resource, action, pluginName)
	return nil
}

// RemovePluginPermissions 移除插件权限
func (pi *PermissionIntegrator) RemovePluginPermissions(pluginName string) error {
	pi.logger.Printf("Removing permissions for plugin %s", pluginName)
	
	resourcePrefix := fmt.Sprintf("plugin.%s.", pluginName)
	
	// 使用RBAC服务删除插件权限
	if err := pi.rbacService.DeletePermissionsByResourcePrefix(resourcePrefix); err != nil {
		return fmt.Errorf("failed to delete permissions for plugin %s: %w", pluginName, err)
	}
	
	pi.logger.Printf("Permissions removed for plugin %s", pluginName)
	return nil
}

// CreatePluginRole 为插件创建专用角色
func (pi *PermissionIntegrator) CreatePluginRole(plugin Plugin) (*models.Role, error) {
	pluginName := plugin.Name()
	roleName := fmt.Sprintf("plugin_%s_user", strings.ToLower(pluginName))
	displayName := fmt.Sprintf("%s 用户", plugin.Description())
	description := fmt.Sprintf("使用 %s 插件的用户角色", plugin.Description())
	
	// 创建角色
	role, err := pi.rbacService.CreateRole(roleName, displayName, description, false)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "already exists") {
			pi.logger.Printf("Role %s already exists for plugin %s", roleName, pluginName)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to create role for plugin %s: %w", pluginName, err)
	}
	
	// 为角色分配插件权限
	permissions := plugin.GetPermissions()
	for _, perm := range permissions {
		resource := fmt.Sprintf("plugin.%s.%s", pluginName, perm.Resource)
		action := perm.Action
		
		// 查找权限
		permission, err := pi.rbacService.GetPermissionByResourceAction(resource, action)
		if err != nil {
			pi.logger.Printf("Permission %s:%s not found, skipping", resource, action)
			continue
		}
		
		// 分配权限到角色
		if err := pi.rbacService.AssignPermissionToRole(role.ID, permission.ID); err != nil {
			pi.logger.Printf("Failed to assign permission %s:%s to role %s: %v", 
				resource, action, roleName, err)
		}
	}
	
	pi.logger.Printf("Created role %s for plugin %s with %d permissions", 
		roleName, pluginName, len(permissions))
	
	return role, nil
}

// CheckPluginPermission 检查插件权限
func (pi *PermissionIntegrator) CheckPluginPermission(userID uint, pluginName, resource, action string) (bool, error) {
	// 构造插件权限资源名
	pluginResource := fmt.Sprintf("plugin.%s.%s", pluginName, resource)
	
	// 使用RBAC服务检查权限
	return pi.rbacService.CheckPermission(userID, pluginResource, action)
}

// GetPluginPermissions 获取插件的所有权限
func (pi *PermissionIntegrator) GetPluginPermissions(pluginName string) ([]*models.Permission, error) {
	resourcePrefix := fmt.Sprintf("plugin.%s.", pluginName)
	return pi.rbacService.GetPermissionsByResourcePrefix(resourcePrefix)
}

// IntegrateAllPluginPermissions 集成所有插件的权限
func (pi *PermissionIntegrator) IntegrateAllPluginPermissions(plugins []Plugin) error {
	pi.logger.Printf("Integrating permissions for %d plugins", len(plugins))
	
	for _, plugin := range plugins {
		if err := pi.IntegratePluginPermissions(plugin); err != nil {
			pi.logger.Printf("Failed to integrate permissions for plugin %s: %v", 
				plugin.Name(), err)
			continue
		}
		
		// 为插件创建默认角色
		if _, err := pi.CreatePluginRole(plugin); err != nil {
			pi.logger.Printf("Failed to create role for plugin %s: %v", 
				plugin.Name(), err)
		}
	}
	
	pi.logger.Println("Plugin permissions integration completed")
	return nil
}