package models

import "time"

type BidFeedback struct {
	ID        uint      `gorm:"primaryKey"`
	BidID     uint      `gorm:"primaryKey"`
	Username  string    `gorm:"not null"`
	Feedback  string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (BidFeedback) TableName() string {
	return "bid_feedback"
}
