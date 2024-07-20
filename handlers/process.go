package handlers

import (
	"antalpha-service/services"
	"fmt"
	"github.com/gin-gonic/gin"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func UpdateHandler(c *gin.Context) {
	var json struct {
		FileName string `json:"fileName"`
	}

	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(400, gin.H{"error": "无效的请求"})
		return
	}

	if filepath.Ext(json.FileName) != ".tar" {
		c.JSON(400, gin.H{"error": "文件不是 .tar 格式"})
		return
	}

	srcPath := filepath.Join("uploads", json.FileName)
	destPath := services.DestPath

	switchResults, err := services.ProcessFile(srcPath, destPath)
	if err != nil {
		fmt.Printf("err: %v", err)
		c.JSON(500, gin.H{"error": fmt.Sprintf("处理文件失败: %s", err.Error())})
		return
	}

	// 遍历目录，查找包含 miner.log 的文件，并搜索 power on 和 power off 的行
	minerResults, err := services.SearchMinerLogs(filepath.Join(services.DestPath, strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath))))
	if err != nil {
		fmt.Printf("err: %v", err)
		c.JSON(500, gin.H{"error": fmt.Sprintf("搜索 miner.log 失败: %s", err.Error())})
		return
	}

	pattern1 := regexp.MustCompile(`(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})_antminer_log_(\d{4}-\d{2}-\d{2})_(\d{4}-\d{2}-\d{2})\.tar`)
	pattern2 := regexp.MustCompile(`antminer_log_(\d{4}-\d{2}-\d{2})_(\d{4}-\d{2}-\d{2})\.tar`)

	matches1 := pattern1.FindStringSubmatch(json.FileName)
	matches2 := pattern2.FindStringSubmatch(json.FileName)

	newFilename := ""

	if len(matches1) == 4 {
		newFilename = fmt.Sprintf("%s_%s_%s.tar", matches1[1], matches1[2], matches1[3])
		fmt.Println(newFilename)
	} else if len(matches2) == 3 {
		newFilename = fmt.Sprintf("%s_%s.tar", matches2[1], matches2[2])
		fmt.Println(newFilename)
	} else {
		c.JSON(500, gin.H{"error": fmt.Sprintf("更新缓存失败。")})
	}

	username := c.MustGet("username").(string)

	// 使用 UserCacheService 保存或更新记录
	userCacheService := services.NewUserCacheService(services.DB)
	if err := userCacheService.SaveOrUpdate(username, newFilename, switchResults, minerResults); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// 删除压缩文件和解压文件
	if err := os.Remove(srcPath); err != nil {
		fmt.Printf("无法删除压缩文件: %v", err)
		//c.JSON(500, gin.H{"error": fmt.Sprintf("无法删除压缩文件: %s", err.Error())})
		//return
	}

	if err := os.RemoveAll(destPath); err != nil {
		fmt.Printf("无法删除解压文件: %v", err)
		//c.JSON(500, gin.H{"error": fmt.Sprintf("无法删除解压文件: %s", err.Error())})
		//return
	}

	c.JSON(200, gin.H{"message": "缓存已更新"})
}
