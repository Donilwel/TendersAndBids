package models

import "time"

type OrganizationResponsible struct {
	ID             uint      `gorm:"primaryKey"`
	OrganizationID uint      `gorm:"not null"`
	UserID         uint      `gorm:"not null"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
}

func (OrganizationResponsible) TableName() string {
	return "organization_responsible"
}
