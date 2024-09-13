package models

import "time"

type TenderVersion struct {
	ID          uint   `gorm:"primaryKey"`
	TenderID    uint   `gorm:"not null"`
	Name        string `gorm:"not null"`
	Description string
	ServiceType string
	Status      TenderStatus
	Version     int       `gorm:"not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

func (TenderVersion) TableName() string {
	return "tender_versions"
}
