package repository

import (
	"context"

	"qim-server/model"

	"gorm.io/gorm"
)

type userRepository struct {
	*baseRepository[model.User]
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		baseRepository: &baseRepository[model.User]{db: db},
		db:             db,
	}
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByPhone(ctx context.Context, phone string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) Search(ctx context.Context, query string, limit int) ([]*model.User, error) {
	var users []*model.User
	searchPattern := "%" + query + "%"
	err := r.db.WithContext(ctx).
		Where("username LIKE ? OR nickname LIKE ? OR phone LIKE ? OR email LIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern).
		Limit(limit).
		Find(&users).Error
	return users, err
}

func (r *userRepository) UpdateStatus(ctx context.Context, id uint, status string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("status", status).Error
}

func (r *userRepository) UpdateLastOnline(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("last_online", gorm.Expr("datetime('now')")).Error
}

func (r *userRepository) WithTx(tx *gorm.DB) BaseRepository[model.User] {
	return &userRepository{
		baseRepository: &baseRepository[model.User]{db: tx},
		db:             tx,
	}
}
