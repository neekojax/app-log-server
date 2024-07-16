package main

import (
	"archive/tar"
	"bufio"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const DestPath = "extracted"
const Result = "result"

var resultCache = sync.Map{}

// processFileHandler 处理文件处理的请求
func processFileHandler(c *gin.Context) {
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

	// 检查缓存中是否已有处理结果
	if cachedResult, found := resultCache.Load(json.FileName); found {
		fmt.Printf("命中缓存(%v)...\n", json.FileName)
		c.JSON(200, cachedResult)
		return
	}

	// 删除之前解压的文件
	//if err := deleteFilesInDir(DestPath); err != nil {
	//	fmt.Printf("err: %v\n", err)
	//	c.JSON(500, gin.H{"error": fmt.Sprintf("删除文件失败: %s", err.Error())})
	//	return
	//}

	srcPath := filepath.Join("uploads", json.FileName)
	destPath := DestPath

	switchResults, err := processFile(srcPath, destPath)
	if err != nil {
		fmt.Printf("err: %v", err)
		c.JSON(500, gin.H{"error": fmt.Sprintf("处理文件失败: %s", err.Error())})
		return
	}

	// 遍历目录，查找包含 miner.log 的文件，并搜索 power on 和 power off 的行
	minerResults, err := searchMinerLogs(DestPath)
	if err != nil {
		fmt.Printf("err: %v", err)
		c.JSON(500, gin.H{"error": fmt.Sprintf("搜索 miner.log 失败: %s", err.Error())})
		return
	}

	result := gin.H{
		"switchLog": switchResults,
		"powerLog":  minerResults,
	}

	// 将处理结果存入缓存
	resultCache.Store(json.FileName, result)

	c.JSON(200, result)
}

// processFile 解压 tar 文件
func processFile(src, dest string) (map[string][]string, error) {
	// 获取压缩文件的名字（不含扩展名）
	fileName := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	// 创建解压后的目标目录
	destPath := filepath.Join(dest, fileName)

	// 检查目标目录是否存在，如果存在则删除
	if _, err := os.Stat(destPath); !os.IsNotExist(err) {
		if err := removeAll(destPath); err != nil {
			return nil, fmt.Errorf("删除已有文件夹失败: %v", err)
		}
	}

	// 创建新的目标目录
	if err := os.MkdirAll(destPath, os.ModePerm); err != nil {
		return nil, fmt.Errorf("创建解压目录失败: %v", err)
	}

	// 在目标目录中解压文件
	if err := untar(src, destPath); err != nil {
		return nil, err
	}

	results, err := searchFiles(destPath)
	if err != nil {
		return nil, err
	}

	// 将结果写入文件
	outputFilePath := filepath.Join(Result, "search_results.txt")
	if err := writeResultsToFile(outputFilePath, results); err != nil {
		return nil, err
	}

	return results, nil
}

// untar 解压 tar 文件
func untar(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	tarReader := tar.NewReader(file)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			targetFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(targetFile, tarReader); err != nil {
				targetFile.Close()
				return err
			}
			targetFile.Close()
		default:
			continue
		}
	}
	return nil
}

// deleteFilesInDir 删除指定目录中的所有文件
func deleteFilesInDir(dir string) error {
	_, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err // 其他错误
	}

	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()

	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}

	return nil
}

// searchFiles 递归遍历目录，查找包含字符串"message"的文件，并读取包含"Stratum+tcp"的行
func searchFiles(root string) (map[string][]string, error) {
	results := make(map[string][]string)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.Contains(info.Name(), "message") {
			lines, err := readStratumLines(path)
			if err != nil {
				return err
			}
			if len(lines) > 0 {
				fmt.Printf("path: %v\n", path)
				pathParts := strings.Split(path, "/")
				results[pathParts[3]] = append(results[pathParts[3]], lines...)
			}
		}
		return nil
	})

	// 对每个 key 里面的 value 按照时间进行排序
	for _, values := range results {
		sort.SliceStable(values, func(i, j int) bool {
			timeI, errI := parseLogTime(values[i])
			timeJ, errJ := parseLogTime(values[j])
			if errI != nil || errJ != nil {
				return values[i] < values[j]
			}
			return timeI.Before(timeJ)
		})
	}

	return results, err
}

// readStratumLines 读取文件中包含"Stratum+tcp"的行
func readStratumLines(filePath string) ([]string, error) {
	var lines []string

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "stratum+tcp") {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// writeResultsToFile 将结果写入指定文件
func writeResultsToFile(filePath string, results map[string][]string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for file, lines := range results {
		fmt.Fprintf(writer, "文件: %s\n", file)
		for _, line := range lines {
			fmt.Fprintf(writer, "%s\n", line)
		}
		fmt.Fprintln(writer, "---------------------")
	}

	return writer.Flush()
}

// searchMinerLogs 遍历目录，查找包含 miner.log 的文件，并搜索 power on 和 power off 的行
func searchMinerLogs(root string) (map[string][]string, error) {
	results := make(map[string][]string)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.Contains(info.Name(), "miner.log") {
			lines, err := readMinerLines(path)
			if err != nil {
				return err
			}
			if len(lines) > 0 {
				pathParts := strings.Split(path, "/")
				results[pathParts[3]] = append(results[pathParts[3]], lines...)
			}
		}
		return nil
	})

	return results, err
}

// readMinerLines 读取文件中包含 power on 和 power off 的行
func readMinerLines(filePath string) ([]string, error) {
	var lines []string

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "power on") || strings.Contains(line, "power off") {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// writeMinerResultsToFile 将 miner 结果写入指定文件
func writeMinerResultsToFile(filePath string, results map[string][]string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for file, lines := range results {
		fmt.Fprintf(writer, "文件: %s\n", file)
		for _, line := range lines {
			fmt.Fprintf(writer, "%s\n", line)
		}
		fmt.Fprintln(writer, "---------------------")
	}

	return writer.Flush()
}

// removeAll removes the directory and its contents.
func removeAll(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("failed to remove existing directory: %v", err)
	}
	return nil
}

func updateHandler(c *gin.Context) {
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

	//// 删除之前解压的文件
	//if err := deleteFilesInDir(DestPath); err != nil {
	//	fmt.Printf("err: %v\n", err)
	//	c.JSON(500, gin.H{"error": fmt.Sprintf("删除文件失败: %s", err.Error())})
	//	return
	//}

	srcPath := filepath.Join("uploads", json.FileName)
	destPath := DestPath

	switchResults, err := processFile(srcPath, destPath)
	if err != nil {
		fmt.Printf("err: %v", err)
		c.JSON(500, gin.H{"error": fmt.Sprintf("处理文件失败: %s", err.Error())})
		return
	}

	// 遍历目录，查找包含 miner.log 的文件，并搜索 power on 和 power off 的行
	minerResults, err := searchMinerLogs(DestPath)
	if err != nil {
		fmt.Printf("err: %v", err)
		c.JSON(500, gin.H{"error": fmt.Sprintf("搜索 miner.log 失败: %s", err.Error())})
		return
	}

	result := gin.H{
		"switchLog": switchResults,
		"powerLog":  minerResults,
	}

	// 将处理结果存入缓存
	resultCache.Store(json.FileName, result)

	c.JSON(200, gin.H{"message": "缓存已更新"})
}

func fetchHandler(c *gin.Context) {
	// 创建一个 map 来存储所有的缓存数据
	allResults := make(map[string]interface{})

	// 遍历 resultCache 中的所有条目
	resultCache.Range(func(key, value interface{}) bool {
		allResults[key.(string)] = value
		return true
	})

	// 将所有缓存数据以 JSON 格式返回
	c.JSON(200, allResults)
}

// parseLogTime 解析日志字符串中的时间部分
func parseLogTime(log string) (time.Time, error) {
	// 日志时间字符串的格式，例如 "Feb 22 05:55:04"
	timeLayout := "Jan 2 15:04:05"
	timeStr := log[:15]
	return time.Parse(timeLayout, timeStr)
}
