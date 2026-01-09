package system_admin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// 用户管理处理器

// listUsers 获取用户列表
func (p *SystemAdminPlugin) listUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	search := c.Query("search")

	result, err := p.userService.ListUsers(page, size, search)
	if err != nil {
		p.error(c, http.StatusInternalServerError, err.Error())
		return
	}

	p.success(c, result)
}

// getUser 获取用户详情
func (p *SystemAdminPlugin) getUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		p.error(c, http.StatusBadRequest, "invalid user id")
		return
	}

	result, err := p.userService.GetUser(uint(id))
	if err != nil {
		p.error(c, http.StatusNotFound, err.Error())
		return
	}

	p.success(c, result)
}

// createUser 创建用户
func (p *SystemAdminPlugin) createUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		p.error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := p.userService.CreateUser(&req)
	if err != nil {
		p.error(c, http.StatusInternalServerError, err.Error())
		return
	}

	p.success(c, result)
}

// updateUser 更新用户
func (p *SystemAdminPlugin) updateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		p.error(c, http.StatusBadRequest, "invalid user id")
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		p.error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := p.userService.UpdateUser(uint(id), &req)
	if err != nil {
		p.error(c, http.StatusInternalServerError, err.Error())
		return
	}

	p.success(c, result)
}

// deleteUser 删除用户
func (p *SystemAdminPlugin) deleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		p.error(c, http.StatusBadRequest, "invalid user id")
		return
	}

	if err := p.userService.DeleteUser(uint(id)); err != nil {
		p.error(c, http.StatusInternalServerError, err.Error())
		return
	}

	p.success(c, nil)
}

// updateUserStatus 更新用户状态
func (p *SystemAdminPlugin) updateUserStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		p.error(c, http.StatusBadRequest, "invalid user id")
		return
	}

	var req UpdateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		p.error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := p.userService.UpdateUserStatus(uint(id), &req); err != nil {
		p.error(c, http.StatusInternalServerError, err.Error())
		return
	}

	p.success(c, nil)
}

// updateUserRoles 更新用户角色
func (p *SystemAdminPlugin) updateUserRoles(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		p.error(c, http.StatusBadRequest, "invalid user id")
		return
	}

	var req UpdateUserRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		p.error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := p.userService.UpdateUserRoles(uint(id), &req); err != nil {
		p.error(c, http.StatusInternalServerError, err.Error())
		return
	}

	p.success(c, nil)
}

// 角色管理处理器

// listRoles 获取角色列表
func (p *SystemAdminPlugin) listRoles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	search := c.Query("search")

	result, err := p.roleService.ListRoles(page, size, search)
	if err != nil {
		p.error(c, http.StatusInternalServerError, err.Error())
		return
	}

	p.success(c, result)
}

// getRole 获取角色详情
func (p *SystemAdminPlugin) getRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		p.error(c, http.StatusBadRequest, "invalid role id")
		return
	}

	result, err := p.roleService.GetRole(uint(id))
	if err != nil {
		p.error(c, http.StatusNotFound, err.Error())
		return
	}

	p.success(c, result)
}

// createRole 创建角色
func (p *SystemAdminPlugin) createRole(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		p.error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := p.roleService.CreateRole(&req)
	if err != nil {
		p.error(c, http.StatusInternalServerError, err.Error())
		return
	}

	p.success(c, result)
}

// updateRole 更新角色
func (p *SystemAdminPlugin) updateRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		p.error(c, http.StatusBadRequest, "invalid role id")
		return
	}

	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		p.error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := p.roleService.UpdateRole(uint(id), &req)
	if err != nil {
		p.error(c, http.StatusInternalServerError, err.Error())
		return
	}

	p.success(c, result)
}

// deleteRole 删除角色
func (p *SystemAdminPlugin) deleteRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		p.error(c, http.StatusBadRequest, "invalid role id")
		return
	}

	if err := p.roleService.DeleteRole(uint(id)); err != nil {
		p.error(c, http.StatusInternalServerError, err.Error())
		return
	}

	p.success(c, nil)
}

// updateRolePermissions 更新角色权限
func (p *SystemAdminPlugin) updateRolePermissions(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		p.error(c, http.StatusBadRequest, "invalid role id")
		return
	}

	var req UpdateRolePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		p.error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := p.roleService.UpdateRolePermissions(uint(id), &req); err != nil {
		p.error(c, http.StatusInternalServerError, err.Error())
		return
	}

	p.success(c, nil)
}

// 权限管理处理器

// listPermissions 获取权限列表
func (p *SystemAdminPlugin) listPermissions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	search := c.Query("search")

	result, err := p.permService.ListPermissions(page, size, search)
	if err != nil {
		p.error(c, http.StatusInternalServerError, err.Error())
		return
	}

	p.success(c, result)
}

// getPermission 获取权限详情
func (p *SystemAdminPlugin) getPermission(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		p.error(c, http.StatusBadRequest, "invalid permission id")
		return
	}

	result, err := p.permService.GetPermission(uint(id))
	if err != nil {
		p.error(c, http.StatusNotFound, err.Error())
		return
	}

	p.success(c, result)
}

// createPermission 创建权限
func (p *SystemAdminPlugin) createPermission(c *gin.Context) {
	var req CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		p.error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := p.permService.CreatePermission(&req)
	if err != nil {
		p.error(c, http.StatusInternalServerError, err.Error())
		return
	}

	p.success(c, result)
}

// updatePermission 更新权限
func (p *SystemAdminPlugin) updatePermission(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		p.error(c, http.StatusBadRequest, "invalid permission id")
		return
	}

	var req UpdatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		p.error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := p.permService.UpdatePermission(uint(id), &req)
	if err != nil {
		p.error(c, http.StatusInternalServerError, err.Error())
		return
	}

	p.success(c, result)
}

// deletePermission 删除权限
func (p *SystemAdminPlugin) deletePermission(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		p.error(c, http.StatusBadRequest, "invalid permission id")
		return
	}

	if err := p.permService.DeletePermission(uint(id)); err != nil {
		p.error(c, http.StatusInternalServerError, err.Error())
		return
	}

	p.success(c, nil)
}