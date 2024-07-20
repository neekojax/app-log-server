package services

import (
	"antalpha-service/models"
	"encoding/json"
	"fmt"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	var err error
	DB, err = gorm.Open(sqlite.Open("cache.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	DB.AutoMigrate(&models.UserCache{})
}

type UserCacheService struct {
	db *gorm.DB
}

func NewUserCacheService(db *gorm.DB) *UserCacheService {
	return &UserCacheService{db: db}
}

func (s *UserCacheService) SaveOrUpdate(username, filename string, switchLog, powerLog map[string][]string) error {
	// 构建日志数据
	logData := models.LogData{
		SwitchLog: switchLog,
		PowerLog:  powerLog,
	}

	// 将日志数据序列化为 JSON
	logDataJSON, err := json.Marshal(logData)
	if err != nil {
		return fmt.Errorf("序列化 logData 失败: %w", err)
	}

	var userCache models.UserCache
	// 查找现有记录
	err = s.db.Where("username = ?", username).First(&userCache).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建新的记录
			logs := map[string]datatypes.JSON{filename: logDataJSON}
			logsJSON, _ := json.Marshal(logs)
			userCache = models.UserCache{
				Username: username,
				Logs:     logsJSON,
			}
			if err := s.db.Create(&userCache).Error; err != nil {
				fmt.Printf("用户 %v 创建记录失败: %v\n", username, err)
				return fmt.Errorf("创建记录失败: %w", err)
			}
		} else {
			fmt.Printf("用户 %v 查询数据库失败: %v\n", username, err)
			return fmt.Errorf("查询数据库失败: %w", err)
		}
	} else {
		// 更新现有记录
		var logs map[string]datatypes.JSON
		if err := json.Unmarshal(userCache.Logs, &logs); err != nil {
			fmt.Printf("用户 %v 获取数据库失败: 反序列化 Logs 失败\n", username)
			return fmt.Errorf("反序列化 Logs 失败: %w", err)
		}
		logs[filename] = logDataJSON
		logsJSON, _ := json.Marshal(logs)
		userCache.Logs = datatypes.JSON(logsJSON)
		if err := s.db.Save(&userCache).Error; err != nil {
			fmt.Printf("用户 %v 更新记录失败: %v\n", username, err)
			return fmt.Errorf("更新记录失败: %w", err)
		}
	}

	return nil
}

func (s *UserCacheService) FetchUserCacheByUsername(username string) (map[string]interface{}, error) {
	var userCache models.UserCache
	err := s.db.Where("username = ?", username).First(&userCache).Error
	if err != nil {
		return nil, err
	}

	// 反序列化 Logs 字段
	var logs map[string]datatypes.JSON
	if err := json.Unmarshal(userCache.Logs, &logs); err != nil {
		fmt.Printf("用户 %v 获取数据库失败: 反序列化 Logs 失败\n", username)
		return nil, fmt.Errorf("反序列化 Logs 失败: %w", err)
	}

	// 创建一个 map 存储反序列化后的日志数据
	result := make(map[string]interface{})

	// 遍历 logs 并进行反序列化
	for filename, logDataJSON := range logs {
		var logData models.LogData
		if err := json.Unmarshal(logDataJSON, &logData); err != nil {
			fmt.Printf("用户 %v 获取数据库失败: 反序列化 logData 失败\n", username)
			return nil, fmt.Errorf("反序列化 logData 失败: %w", err)
		}
		// 将 logData 转换为 map
		logDataMap := map[string]map[string][]string{
			"switchLog": logData.SwitchLog,
			"powerLog":  logData.PowerLog,
		}

		result[filename] = logDataMap
	}

	return result, nil
}
