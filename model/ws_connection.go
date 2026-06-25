package model

import (
	"time"

	"gorm.io/gorm"
)

type WsConnection struct {
	ID         uint   `gorm:"primaryKey;autoIncrement"`
	TerminalId string `gorm:"type:varchar(20)"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}
