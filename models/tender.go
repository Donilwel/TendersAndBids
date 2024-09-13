package models

import (
	"time"
)

type TenderStatus string

const (
	CREATED   TenderStatus = "CREATED"
	PUBLISHED TenderStatus = "PUBLISHED"
	CLOSED    TenderStatus = "CLOSED"
)

type Tender struct {
	ID              uint   `gorm:"primaryKey"`
	Name            string `gorm:"not null"`
	Description     string
	ServiceType     string
	Status          TenderStatus `gorm:"type:tender_status;default:'CREATED'"`
	OrganizationID  uint         `gorm:"not null"`
	CreatorUsername string       `gorm:"not null"`
	CreatedAt       time.Time    `gorm:"autoCreateTime"`
	UpdatedAt       time.Time    `gorm:"autoUpdateTime"`
	Version         int          `gorm:"default:1"`
}

func (Tender) TableName() string {
	return "tenders"
}
