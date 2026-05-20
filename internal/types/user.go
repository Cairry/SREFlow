package types

import (
	"SREFlow/internal/ctx"
	"SREFlow/internal/models"
	"fmt"
)

type RequestUserRegister struct {
	Username        string `json:"username"`
	Email           string `json:"email"`
	Mobile          string `json:"mobile"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
}

func (register *RequestUserRegister) Valid(ctx *ctx.Context) error {
	var (
		count int64
		db    = ctx.Database.MySQL()
	)

	db.Model(&models.UserModel{}).Where("username = ?", register.Username).Count(&count)
	if count > 0 {
		return fmt.Errorf("用户名已存在")
	}

	db.Model(&models.UserModel{}).Where("email = ?", register.Email).Count(&count)
	if count > 0 {
		return fmt.Errorf("邮箱已存在")
	}

	db.Model(&models.UserModel{}).Where("mobile = ?", register.Mobile).Count(&count)
	if count > 0 {
		return fmt.Errorf("手机号已存在")
	}

	if register.Password != register.ConfirmPassword {
		return fmt.Errorf("密码不一致")
	}

	return nil
}

type ResponseUserRegister struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Mobile   string `json:"mobile"`
}

type RequestUserLogin struct {
	Identity string `json:"identity"`
	Password string `json:"password"`
}

func (login *RequestUserLogin) Valid(ctx *ctx.Context) error {
	var (
		count int64
		db    = ctx.Database.MySQL()
	)

	db.Model(&models.UserModel{}).
		Where("username = ? OR email = ? OR mobile = ?", login.Identity, login.Identity, login.Identity).
		Count(&count)
	if count <= 0 {
		return fmt.Errorf("用户名/邮箱/手机号 不存在")
	}

	return nil
}

type ResponseUserLogin struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Mobile   string `json:"mobile"`
	Token    string `json:"token"`
}

type RequestUserDetail struct {
	Identity string `json:"identity"`
}

func (detail *RequestUserDetail) Valid() error {
	if detail.Identity == "" {
		return fmt.Errorf("查询条件不可为空")
	}

	return nil
}

type ResponseUserDetail struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Mobile   string `json:"mobile"`
	Password string `json:"password"`
}
