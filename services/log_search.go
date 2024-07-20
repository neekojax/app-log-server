package services

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SearchFiles searchFiles 递归遍历目录，查找包含字符串"message"的文件，并读取包含"Stratum+tcp"的行
func SearchFiles(root string) (map[string][]string, error) {
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

// SearchMinerLogs searchMinerLogs 遍历目录，查找包含 miner.log 的文件，并搜索 power on 和 power off 的行
func SearchMinerLogs(root string) (map[string][]string, error) {
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
				fmt.Printf("power path: %v\n", path)
				pathParts := strings.Split(path, "/")
				results[pathParts[3]] = append(results[pathParts[3]], lines...)
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

// parseLogTime 解析日志字符串中的时间部分
func parseLogTime(log string) (time.Time, error) {
	// 日志时间字符串的格式，例如 "Feb 22 05:55:04"
	timeLayout := "Jan 2 15:04:05"
	timeStr := log[:15]
	return time.Parse(timeLayout, timeStr)
}
