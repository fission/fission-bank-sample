package main

import "time"

type (
	Message struct {
		// field from gorm.Model
		ID        uint       `json:"id";gorm:"primary_key"`
		CreatedAt time.Time  `json:"-"`
		UpdatedAt time.Time  `json:"-"`
		DeletedAt *time.Time `json:"-";sql:"index"`

		Message string `json:"message"`
	}
)
