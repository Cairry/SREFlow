package repos

import (
	"RCAgent/internal/models"

	"gorm.io/gorm"
)

type (
	UserRepo struct {
		mysql *gorm.DB
	}

	UserRepoInter interface {
		Register(req models.UserModel) error
		Detail(identity string) (models.UserModel, error)
	}
)

func NewRepoUser(mysql *gorm.DB) UserRepoInter {
	return &UserRepo{
		mysql: mysql,
	}
}

func (user *UserRepo) Register(req models.UserModel) error {
	db := user.mysql.Model(&models.UserModel{})
	if err := db.Create(&req).Error; err != nil {
		return err
	}

	return nil
}

func (user *UserRepo) Detail(identity string) (models.UserModel, error) {
	db := user.mysql.Model(&models.UserModel{})
	if identity != "" {
		db.Where("username = ? OR email = ? OR mobile = ?", identity, identity, identity)
	}

	var u models.UserModel
	if err := db.First(&u).Error; err != nil {
		return u, err
	}

	return u, nil
}
