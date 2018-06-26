package main

import (
	"github.com/jinzhu/gorm"
)

type (
	Account struct {
		gorm.Model `json:"-"`
		Username   string `gorm:"unique_index;not null" json:"username"`
		Password   string `gorm:"not null" json:"password"`
	}

	TransactionRecord struct {
		gorm.Model `json:"-"`
		Type       string  `gorm:"index;not null" json:"type"`
		From       string  `gorm:"not null" json:"from"`
		To         string  `gorm:"not null" json:"to"`
		Amount     float32 `gorm:"not null" json:"amount"`
		Timestamp  int64   `gorm:"not null" json:"timestamp"`
	}

	AccountBalance struct {
		gorm.Model `json:"-"`
		Username   string  `json:"username"`
		Balance    float32 `json:"balance"`
	}

	Session struct {
		gorm.Model `json:"-"`
		Token      string `json:"token"`
		Username   string `json:"username"`
		Expiration int64  `json:"-"`
	}

	Transaction struct {
		Amount float32 `json:"amount"`
	}

	TransferTransaction struct {
		To     string  `json:"to"`
		Amount float32 `json:"amount"`
	}
)

const (
	// TransactionALL is a special transaction type and
	// only use as filter to get all transaction record from DB
	TransactionALL      = "*"
	TransactionDeposit  = "deposit"
	TransactionWithdraw = "withdraw"
	TransactionTransfer = "transfer"
)
