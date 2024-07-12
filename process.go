package main

import (
	"archive/tar"
	"bufio"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const DestPath = "extracted"
const Result = "result"

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

	// 删除之前解压的文件
	if err := deleteFilesInDir(DestPath); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("删除文件失败: %s", err.Error())})
		return
	}

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
		c.JSON(500, gin.H{"error": fmt.Sprintf("搜索 miner.log 失败: %s", err.Error())})
		return
	}

	c.JSON(200, gin.H{
		"switchLog": switchResults,
		"powerLog":  minerResults,
	})
	// 将结果返回给前端
	//c.JSON(200, results)
}

// processFile 解压 tar 文件
func processFile(src, dest string) (map[string][]string, error) {
	if err := untar(src, dest); err != nil {
		return nil, err
	}

	results, err := searchFiles(DestPath)
	if err != nil {
		return nil, err
	}

	// 将结果写入文件
	outputFilePath := filepath.Join(Result, "search_results.txt")
	if err := writeResultsToFile(outputFilePath, results); err != nil {
		//c.JSON(500, gin.H{"error": fmt.Sprintf("写入文件失败: %s", err.Error())})
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
				pathParts := strings.Split(path, "/")
				results[pathParts[2]] = append(results[pathParts[2]], lines...)
			}
		}
		return nil
	})

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
				results[pathParts[2]] = append(results[pathParts[2]], lines...)
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
