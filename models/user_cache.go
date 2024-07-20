package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type LogData struct {
	SwitchLog map[string][]string `json:"switchLog"`
	PowerLog  map[string][]string `json:"powerLog"`
}

type UserCache struct {
	gorm.Model
	Username string         `gorm:"primaryKey"`
	Logs     datatypes.JSON `gorm:"type:json"`
}
