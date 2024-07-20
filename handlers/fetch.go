package handlers

import (
	"antalpha-service/services"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func FetchHandler(c *gin.Context) {
	//allResults := make(map[string]interface{})

	username := c.MustGet("username").(string)
	fmt.Printf("FetchHandler username: %v\n", username)

	// 创建 UserCacheService 实例
	userCacheService := services.NewUserCacheService(services.DB) // 假设 db 是你的数据库连接实例

	// 从数据库读取该用户名的缓存数据
	logs, err := userCacheService.FetchUserCacheByUsername(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)

	//services.ResultCache.Range(func(key, value interface{}) bool {
	//	allResults[key.(string)] = value
	//	return true
	//})
	//
	//c.JSON(200, allResults)
}
