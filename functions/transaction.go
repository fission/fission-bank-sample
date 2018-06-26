package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

func getTransaction(db *gorm.DB, username string, transType string) ([]TransactionRecord, error) {
	_, err := getAccountInfoByName(db, username)
	if err != nil {
		log.Printf("Error getting account information when trying to get transactions for user %v: %v", username, err)
		return nil, err
	}

	if len(transType) == 0 {
		transType = TransactionALL
	}

	var where string

	switch transType {
	case TransactionALL:
		where = fmt.Sprintf("\"from\" = '%v' OR \"to\" = '%v'", username, username)
	case TransactionWithdraw, TransactionDeposit:
		where = fmt.Sprintf("\"type\" = '%v' AND \"to\" = '%v'", transType, username)
	case TransactionTransfer:
		where = fmt.Sprintf("\"type\" = '%v' AND (\"from\" = '%v' OR \"to\" = '%v')", transType, username, username)
	}

	var trans []TransactionRecord

	err = db.Where(where).Find(&trans).Error
	if err != nil {
		log.Printf("Error getting transactions for user %s", username)
		return nil, err
	}

	return trans, nil
}

func deposit(db *gorm.DB, username string, amount float32) error {
	if amount <= 0 {
		return errors.New("Amount should be higher than 0")
	}

	_, err := getAccountInfoByName(db, username)
	if err != nil {
		return errors.Wrap(err, "Error getting account information")
	}

	// Atomic operation
	tx := db.Begin()
	defer func() {
		// recover from panic and rollback transaction just in case.
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	trans := TransactionRecord{
		Type:      TransactionDeposit,
		To:        username,
		Amount:    amount,
		Timestamp: time.Now().Unix(),
	}

	// write transaction log
	err = tx.Create(&trans).Error
	if err != nil {
		tx.Rollback()
		log.Printf("Error creating transaction record: %v", err)
		return err
	}

	err = tx.Model(AccountBalance{}).Where("username = ?", username).UpdateColumn("balance", gorm.Expr("balance + ?", amount)).Error
	if err != nil {
		tx.Rollback()
		log.Printf("Error depositing money to user %v account: %v", username, err)
		return err
	}

	return tx.Commit().Error
}

func withdraw(db *gorm.DB, username string, amount float32) error {
	if amount <= 0 {
		return errors.New("Amount should be higher than 0")
	}

	_, err := getAccountInfoByName(db, username)
	if err != nil {
		if isRecordNotFound(err) {
			return errors.New(fmt.Sprintf("Error withdrawing money from non-exist user %s account", username))
		}
		return errors.Wrap(err, "Error getting account information")
	}

	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	bal, err := getAccountBalance(db, username)
	if err != nil {
		return errors.Wrap(err, "Error getting account balance")
	}

	if bal <= 0 || bal-amount < 0 {
		return errors.New("No enough balance to withdraw, please check the account has enough balance")
	}

	trans := TransactionRecord{
		Type:      TransactionWithdraw,
		To:        username,
		Amount:    amount,
		Timestamp: time.Now().Unix(),
	}

	err = tx.Create(&trans).Error
	if err != nil {
		tx.Rollback()
		log.Printf("Error creating transaction record: %v", err)
		return err
	}

	err = tx.Model(AccountBalance{}).Where("username = ?", username).
		UpdateColumn("balance", gorm.Expr("balance - ?", amount)).Error
	if err != nil {
		tx.Rollback()
		log.Printf("Error withdrawing money to user %v account: %v", username, err)
		return err
	}

	return tx.Commit().Error
}

func transfer(db *gorm.DB, from string, to string, amount float32) error {
	if amount <= 0 {
		return errors.New("Amount should be higher than 0")
	}

	if len(from) == 0 || len(to) == 0 {
		return errors.New("From/To user account is empty")
	}

	_, err := getAccountInfoByName(db, from)
	if err != nil {
		if isRecordNotFound(err) {
			return errors.New(fmt.Sprintf("Error transfering money from non-exist user %s account", from))
		}
		return errors.Wrap(err, "Error getting account information")
	}

	_, err = getAccountInfoByName(db, to)
	if err != nil {
		if isRecordNotFound(err) {
			return errors.New(fmt.Sprintf("Error transfering money to non-exist user %s account", to))
		}
		return errors.Wrap(err, "Error getting account information")
	}

	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	bal, err := getAccountBalance(db, from)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "Error getting account balance")
	}

	if bal <= 0 || bal-amount < 0 {
		tx.Rollback()
		return errors.New("No enough balance to transfer, please check the account has enough balance")
	}

	trans := TransactionRecord{
		Type:      TransactionTransfer,
		From:      from,
		To:        to,
		Amount:    amount,
		Timestamp: time.Now().Unix(),
	}

	err = tx.Create(&trans).Error
	if err != nil {
		tx.Rollback()
		log.Printf("Error creating transaction record: %v", err)
		return err
	}

	err = tx.Model(AccountBalance{}).Where("username = ?", from).
		UpdateColumn("balance", gorm.Expr("balance - ?", amount)).Error
	if err != nil {
		tx.Rollback()
		log.Printf("Error transfering money from user %v account: %v", from, err)
		return err
	}

	err = tx.Model(AccountBalance{}).Where("username = ?", to).
		UpdateColumn("balance", gorm.Expr("balance + ?", amount)).Error
	if err != nil {
		tx.Rollback()
		log.Printf("Error transfering money to user %v account: %v", to, err)
		return err
	}

	return tx.Commit().Error
}

func TransactionDepositHandler(w http.ResponseWriter, r *http.Request) {
	session, err := getSession(getDB(), r)
	if err != nil {
		replyError(err, http.StatusBadRequest, w)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		err = errors.Wrap(err, "Error reading request body")
		replyError(err, http.StatusBadRequest, w)
		return
	}

	trans := Transaction{}
	err = json.Unmarshal(body, &trans)
	if err != nil {
		err = errors.Wrap(err, "Error decoding request body")
		replyError(err, http.StatusBadRequest, w)
		return
	}

	err = deposit(getDB(), session.Username, trans.Amount)
	if err != nil {
		err = errors.Wrap(err, "Error depositing money")
		replyError(err, http.StatusInternalServerError, w)
		return
	}

	msg := fmt.Sprintf("Despoit %.3f to account %v successfully", trans.Amount, session.Username)
	reply([]byte(msg), http.StatusOK, w)
}

func TransactionWithdrawHandler(w http.ResponseWriter, r *http.Request) {
	session, err := getSession(getDB(), r)
	if err != nil {
		replyError(err, http.StatusBadRequest, w)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		err = errors.Wrap(err, "Error reading request body")
		replyError(err, http.StatusBadRequest, w)
		return
	}

	trans := Transaction{}
	err = json.Unmarshal(body, &trans)
	if err != nil {
		err = errors.Wrap(err, "Error decoding request body")
		replyError(err, http.StatusBadRequest, w)
		return
	}

	err = withdraw(getDB(), session.Username, trans.Amount)
	if err != nil {
		err = errors.Wrap(err, "Error withdrawing money")
		replyError(err, http.StatusInternalServerError, w)
		return
	}

	msg := fmt.Sprintf("Withdraw %.3f from account %v successfully", trans.Amount, session.Username)
	reply([]byte(msg), http.StatusOK, w)
}

func TransactionTransferHandler(w http.ResponseWriter, r *http.Request) {
	session, err := getSession(getDB(), r)
	if err != nil {
		replyError(err, http.StatusBadRequest, w)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		err = errors.Wrap(err, "Error reading request body")
		replyError(err, http.StatusBadRequest, w)
		return
	}

	trans := TransferTransaction{}
	err = json.Unmarshal(body, &trans)
	if err != nil {
		err = errors.Wrap(err, "Error decoding request body")
		replyError(err, http.StatusBadRequest, w)
		return
	}

	err = transfer(getDB(), session.Username, trans.To, trans.Amount)
	if err != nil {
		err = errors.Wrap(err, "Error transfering money")
		replyError(err, http.StatusInternalServerError, w)
		return
	}

	msg := fmt.Sprintf("Transfer %.3f from account %v to account %v successfully", trans.Amount, session.Username, trans.To)
	reply([]byte(msg), http.StatusOK, w)
}

func TransactionGetHandler(w http.ResponseWriter, r *http.Request) {
	transType := r.URL.Query().Get("type")
	switch transType {
	case TransactionDeposit, TransactionTransfer, TransactionWithdraw:
		break
	case "":
		transType = TransactionALL
	default:
		err := errors.New("Invalid transaction type")
		replyError(err, http.StatusBadRequest, w)
		return
	}

	session, err := getSession(getDB(), r)
	if err != nil {
		replyError(err, http.StatusBadRequest, w)
		return
	}

	trans, err := getTransaction(getDB(), session.Username, transType)
	if err != nil {
		err = errors.Wrap(err, "Error retrieving transaction records")
		replyError(err, http.StatusInternalServerError, w)
		return
	}

	replyJson(trans, http.StatusOK, w)
}
