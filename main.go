package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	// 配置 CORS 中间件
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type"}

	router.Use(cors.New(config))

	// 处理文件上传请求
	router.POST("/upload", handleUpload)
	router.POST("/update", updateHandler)
	router.GET("/fetch", fetchHandler)

	// 启动服务器
	router.Run(":8080")
}

// 处理文件上传的函数
func handleUpload(c *gin.Context) {
	// 从表单中获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("get form file err: %s", err.Error()))
		return
	}

	// 创建保存上传文件的目录
	os.MkdirAll("./uploads", os.ModePerm)

	// 创建一个本地文件来保存上传的文件
	dst := filepath.Join("./uploads", file.Filename)
	out, err := os.Create(dst)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("create file err: %s", err.Error()))
		return
	}
	defer out.Close()

	// 打开上传的文件
	in, err := file.Open()
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("open file err: %s", err.Error()))
		return
	}
	defer in.Close()

	// 将上传的文件内容复制到本地文件
	if _, err := io.Copy(out, in); err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("copy file err: %s", err.Error()))
		return
	}

	// 返回成功信息
	c.String(http.StatusOK, fmt.Sprintf("File %s uploaded successfully.", file.Filename))
}
