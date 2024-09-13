package models

import "time"

type BidVersion struct {
	BidID       uint           `gorm:"primaryKey"`
	Name        string         `gorm:"not null" json:"name"`
	Description string         `json:"description"`
	Status      BidStatus      `gorm:"type:bid_status;default:'CREATED'" json:"status"`
	TenderID    uint           `gorm:"not null" json:"tenderId"`
	AuthorType  AuthorBidsType `gorm:"not null" json:"author_type"`
	AuthorID    uint           `gorm:"not null" json:"author_id"`
	Version     int            `gorm:"default:1" json:"version"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (BidVersion) TableName() string {
	return "bid_versions"
}
