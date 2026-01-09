package repository

import (
	"errors"
	"time"

	"nmp-platform/internal/models"

	"gorm.io/gorm"
)

// UserRepository 用户仓库接口
type UserRepository interface {
	Create(user *models.User) error
	GetByID(id uint) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	Update(user *models.User) error
	Delete(id uint) error
	List(page, size int, search string) ([]*models.User, int64, error)
	UpdateLastLogin(userID uint) error
	AssignRoles(userID uint, roleIDs []uint) error
}

// userRepository 用户仓库实现
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建新的用户仓库
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create 创建用户
func (r *userRepository) Create(user *models.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	// 检查用户名是否已存在
	var existingUser models.User
	if err := r.db.Where("username = ?", user.Username).First(&existingUser).Error; err == nil {
		return errors.New("username already exists")
	}

	// 检查邮箱是否已存在
	if user.Email != "" {
		if err := r.db.Where("email = ?", user.Email).First(&existingUser).Error; err == nil {
			return errors.New("email already exists")
		}
	}

	return r.db.Create(user).Error
}

// GetByID 根据ID获取用户
func (r *userRepository) GetByID(id uint) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Roles").Preload("Roles.Permissions").First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// GetByUsername 根据用户名获取用户
func (r *userRepository) GetByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Roles").Preload("Roles.Permissions").Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail 根据邮箱获取用户
func (r *userRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Roles").Preload("Roles.Permissions").Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// Update 更新用户
func (r *userRepository) Update(user *models.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	return r.db.Save(user).Error
}

// Delete 删除用户
func (r *userRepository) Delete(id uint) error {
	return r.db.Delete(&models.User{}, id).Error
}

// List 获取用户列表
func (r *userRepository) List(page, size int, search string) ([]*models.User, int64, error) {
	var users []*models.User
	var total int64

	query := r.db.Model(&models.User{})
	
	// 添加搜索条件
	if search != "" {
		query = query.Where("username ILIKE ? OR email ILIKE ? OR full_name ILIKE ?", 
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 计算偏移量
	offset := (page - 1) * size

	// 获取用户列表
	err := query.Preload("Roles").Offset(offset).Limit(size).Find(&users).Error
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// UpdateLastLogin 更新最后登录时间
func (r *userRepository) UpdateLastLogin(userID uint) error {
	now := time.Now()
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("last_login", &now).Error
}

// AssignRoles 分配角色给用户
func (r *userRepository) AssignRoles(userID uint, roleIDs []uint) error {
	// 开始事务
	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除现有角色关联
	if err := tx.Where("user_id = ?", userID).Delete(&models.UserRole{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 添加新的角色关联
	if len(roleIDs) > 0 {
		userRoles := make([]models.UserRole, len(roleIDs))
		for i, roleID := range roleIDs {
			userRoles[i] = models.UserRole{
				UserID: userID,
				RoleID: roleID,
			}
		}

		if err := tx.Create(&userRoles).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}