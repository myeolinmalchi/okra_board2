// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package module

import (
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gorm.io/gorm"
	"okra_board2/config"
	"okra_board2/controllers"
	"okra_board2/repositories"
	"okra_board2/services"
)

// Injectors from wire.go:

func InitAdminController(db *gorm.DB) controllers.AdminController {
	adminRepository := repositories.NewAdminRepositoryImpl(db)
	adminService := services.NewAdminServiceImpl(adminRepository)
	adminController := controllers.NewAdminControllerImpl(adminService)
	return adminController
}

func InitAuthController(db *gorm.DB) controllers.AuthController {
	authRepository := repositories.NewAuthRepositoryImpl(db)
	adminRepository := repositories.NewAdminRepositoryImpl(db)
	adminService := services.NewAdminServiceImpl(adminRepository)
	authService := services.NewAuthServiceImpl(authRepository, adminService)
	authController := controllers.NewAuthControllerImpl(authService, adminService)
	return authController
}

func InitPostController(db *gorm.DB, conf *config.Config, client *s3.Client) controllers.PostController {
	postRepository := repositories.NewPostRepositoryImpl(db)
	postService := services.NewPostServiceImpl(postRepository, conf, client)
	postController := controllers.NewPostControllerImpl(postService)
	return postController
}
