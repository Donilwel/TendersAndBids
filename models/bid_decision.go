package models

type BidDecision struct {
	ID            uint   `gorm:"primaryKey"`
	BidID         uint   `gorm:"not null"`
	ResponsibleID uint   `gorm:"not null"`
	Decision      string `gorm:"not null"`
}

func (BidDecision) TableName() string {
	return "bid_decisions"
}
