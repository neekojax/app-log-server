package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

var Users = map[string]string{
	"chenqiong": "123456",
	"leike":     "123456",
	"lilei":     "123456",
	"neeko":     "123456",
}

// LoginHandler 处理文件上传的函数
func LoginHandler(c *gin.Context) {
	var loginCredentials Credentials

	if err := c.ShouldBindJSON(&loginCredentials); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证用户名和密码
	password, ok := Users[loginCredentials.Username]
	if !ok || password != loginCredentials.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Username or password is incorrect"})
		return
	}

	token, err := GenerateToken(loginCredentials.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	fmt.Printf("token: %v\n", token)

	expirationTime := time.Now().Add(24 * time.Hour) // 设定cookie的有效期为24小时

	c.SetCookie("token", token, int(expirationTime.Unix()), "/", "localhost", true, true)
	c.JSON(http.StatusOK, gin.H{"message": "Login successful", "token": token})
}
