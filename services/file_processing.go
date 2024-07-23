package services

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const DestPath = "extracted"
const Result = "result"

// ProcessFile processFile 解压 tar 文件
func ProcessFile(src, dest string) (map[string][]string, error) {
	// 获取压缩文件的名字（不含扩展名）
	fileName := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	// 创建解压后的目标目录
	destPath := filepath.Join(dest, fileName)

	// 检查目标目录是否存在，如果存在则删除
	if _, err := os.Stat(destPath); !os.IsNotExist(err) {
		if err := os.RemoveAll(destPath); err != nil {
			return nil, fmt.Errorf("删除已有文件夹失败: %v", err)
		}
	}

	// 创建新的目标目录
	if err := os.MkdirAll(destPath, os.ModePerm); err != nil {
		return nil, fmt.Errorf("创建解压目录失败: %v", err)
	}

	// 在目标目录中解压文件
	if err := untar(src, destPath); err != nil {
		fmt.Printf("untar err\n")
		//return nil, err
	}

	results, err := SearchFiles(destPath)
	if err != nil {
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
