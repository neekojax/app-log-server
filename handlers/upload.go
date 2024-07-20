package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// 处理文件上传的函数
func HandleUploadHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("get form file err: %s", err.Error()))
		return
	}

	os.MkdirAll("./uploads", os.ModePerm)
	dst := filepath.Join("./uploads", file.Filename)
	out, err := os.Create(dst)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("create file err: %s", err.Error()))
		return
	}
	defer out.Close()

	in, err := file.Open()
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("open file err: %s", err.Error()))
		return
	}
	defer in.Close()

	if _, err := io.Copy(out, in); err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("copy file err: %s", err.Error()))
		return
	}

	c.String(http.StatusOK, fmt.Sprintf("File %s uploaded successfully.", file.Filename))
}
