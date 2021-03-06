package main

import (
	"io"
	"log"
	"okra_board2/config"
	"okra_board2/controllers"
	"okra_board2/module"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {

    conf, err := config.LoadConfig()
    if err != nil {
        log.Println("설정 파일을 불러오지 못했습니다. 서버를 종료합니다.")    
        log.Println(err.Error())
        return
    }

    if file, err := config.InitLogger(conf); err != nil {
        log.Println("로그 파일을 생성하지 못했습니다. 서버를 종료합니다.")
        log.Println(err.Error())
        return
    } else {
        gin.DefaultWriter = io.MultiWriter(file)
        log.SetOutput(file)
    }

    db, err := config.InitDBConnection(conf)
    if err != nil {
        log.Println("DB 연결에 실패했습니다. 서버를 종료합니다.")
        log.Println(err.Error())
        return
    }

    s3, err := config.InitAwsS3Client(conf)
    if err != nil {
        log.Println("AWS S3 연결에 실패했습니다. 서버를 종료합니다.")
        log.Println(err.Error())
        return
    }

    os.Setenv("ACCESS_SECRET", conf.AccessSecret)
    os.Setenv("REFRESH_SECRET", conf.RefreshSecret)
    os.Setenv("DOMAIN", conf.Domain)
    os.Setenv("DEFAULT_THUMBNAIL", conf.DefaultThumbnail)
    
    gin.SetMode(gin.ReleaseMode)
    route := gin.New()
    route.Use(cors.New(cors.Config {
        AllowAllOrigins:    true,
        AllowMethods:       []string{"GET", "POST", "PUT", "DELETE"},
        AllowHeaders:       []string{"Content-Type", "Authorization"},
        ExposeHeaders:      []string{"Authorization"},
        AllowCredentials:   true,
        MaxAge: 12 * time.Hour,
    }))
    route.Use(gin.LoggerWithConfig(gin.LoggerConfig{SkipPaths: []string{"/"}}))
    route.Use(gin.Recovery())

    route.Static("/images", "./public/images")

    authController := module.InitAuthController(db)
    adminController := module.InitAdminController(db)
    postController := module.InitPostController(db, conf, s3)
    //imageController := controllers.NewImageControllerImpl()
    imageController := controllers.NewImageControllerImpl2(conf, s3)

    // Route for health check
    route.GET("/", func(c *gin.Context) {
        c.Status(200)
    })

    v1 := route.Group("/api/v1")
    {
        v1.GET("/posts_enabled", postController.GetPosts(true))
        v1.GET("/posts_enabled/:postId", postController.GetPost(true))
        v1.GET("/thumbnails", postController.GetSelectedThumbnails)

        v1.GET("/posts", authController.Auth, postController.GetPosts(false))
        v1.GET("/posts/:postId", authController.Auth, postController.GetPost(false))

        v1.POST("/posts", authController.Auth, postController.WritePost)
        v1.PUT("/posts/:postId", authController.Auth, postController.UpdatePost)
        v1.DELETE("/posts/:postId", authController.Auth, postController.DeletePost)
        v1.POST("/posts/selected", authController.Auth, postController.ResetSelectedPosts)

        // TODO
        v1.POST("/admin", authController.Auth, adminController.Register)
        v1.PUT("/admin/:id", authController.Auth, adminController.Update)
        v1.DELETE("/admin/:id", authController.Auth, adminController.Delete)
        v1.POST("/admin/login", authController.Login)
        v1.POST("/admin/logout", authController.Logout)
        v1.POST("/admin/auth", authController.ReissueAccessToken)

        v1.POST("/image/upload", authController.Auth, imageController.UploadImage) 
        v1.POST("/image/delete", authController.Auth, imageController.DeleteImage)
    }
    route.Run(":3000")
}
