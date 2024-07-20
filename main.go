package main

import (
	"antalpha-service/handlers"
	"antalpha-service/services"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	services.InitDB()

	router := gin.Default()

	// 配置 CORS 中间件
	config := cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"}, // 前端地址
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Content-Length", "Authorization", "Credentials"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	router.Use(cors.New(config))

	router.POST("/login", handlers.LoginHandler)

	protected := router.Group("/").Use(handlers.JWTMiddleware())
	{
		protected.POST("/upload", handlers.HandleUploadHandler)
		protected.POST("/update", handlers.UpdateHandler)
		protected.GET("/fetch", handlers.FetchHandler)
	}

	// 启动服务器
	router.Run(":8080")
}
