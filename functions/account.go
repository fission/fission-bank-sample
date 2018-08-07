package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

const (
	ErrAccountExists = "Account already exists"
	Salt             = "this-is-just-dummy-salt"
)

func encodePassword(password string) string {
	h := sha256.New()
	h.Write([]byte(password))
	return hex.EncodeToString(h.Sum([]byte(Salt)))
}

func createAccount(db *gorm.DB, username, password string) error {
	// check for empty string
	if len(username) == 0 || len(password) == 0 {
		return errors.New("username or password should not be empty")
	}

	// check does account name already exist
	_, err := getAccountInfoByName(db, username)
	if err == nil {
		return errors.New(ErrAccountExists)
	}

	if !isRecordNotFound(err) {
		return err
	}

	h := sha256.New()
	h.Write([]byte(password))

	acc := Account{
		Username: username,
		Password: encodePassword(password),
	}

	err = db.Create(&acc).Error
	if err != nil {
		return err
	}

	bal := AccountBalance{
		Username: username,
		Balance:  0,
	}

	err = db.Create(&bal).Error
	if err != nil {
		return err
	}

	return nil
}

func getAccountBalance(db *gorm.DB, username string) (float32, error) {
	var bal AccountBalance

	err := db.Where("username = ?", username).First(&bal).Error
	if err != nil {
		return 0, err
	}

	return bal.Balance, nil
}

func AccountCreateHandler(w http.ResponseWriter, r *http.Request) {
	// POST /accounts/
	// POST /accounts

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		err = errors.Wrap(err, "Error reading request body")
		replyError(err, http.StatusBadRequest, w)
		return
	}

	log.Printf("body -> %v", string(body))

	acc := Account{}
	err = json.Unmarshal(body, &acc)
	if err != nil {
		err = errors.Wrap(err, "Error decoding request body")
		replyError(err, http.StatusBadRequest, w)
		return
	}

	//  Retrieve record from database
	record, err := getAccountInfoByName(getDB(), acc.Username)
	if err != nil && !isRecordNotFound(err) {
		err = errors.Wrap(err, "Error checking account existence")
		replyError(err, http.StatusInternalServerError, w)
		return
	}

	if record != nil {
		err = errors.New(fmt.Sprintf("Account %v already exists", acc.Username))
		replyError(err, http.StatusConflict, w)
		return
	}

	err = createAccount(getDB(), acc.Username, acc.Password)
	if err != nil {
		err = errors.Wrap(err, "Error creating user account")
		replyError(err, http.StatusInternalServerError, w)
		return
	}

	reply([]byte("Account created successfully"), http.StatusOK, w)
}

func AccountLoginHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		err = errors.Wrap(err, "Error reading request body")
		replyError(err, http.StatusBadRequest, w)
		return
	}

	acc := Account{}
	err = json.Unmarshal(body, &acc)
	if err != nil {
		err = errors.Wrap(err, "Error decoding request body")
		replyError(err, http.StatusBadRequest, w)
		return
	}

	acc.Password = encodePassword(acc.Password)

	err = getDB().Where("username = ? AND password = ?", acc.Username, acc.Password).Find(&acc).Error
	if err != nil {
		httpCode := http.StatusInternalServerError
		if isRecordNotFound(err) {
			httpCode = http.StatusUnauthorized
		}
		err = errors.Wrap(err, "Error loginning user account")
		replyError(err, httpCode, w)
		return
	}

	sess := Session{
		Token:      uuid.NewV4().String(),
		Username:   acc.Username,
		Expiration: time.Now().Add(10 * time.Minute).Unix(),
	}

	err = getDB().Create(&sess).Error
	if err != nil {
		err = errors.Wrap(err, "Error login")
		replyError(err, http.StatusInternalServerError, w)
		return
	}

	resp, err := json.Marshal(sess)
	if err != nil {
		err = errors.Wrap(err, "Error encoding session info")
		replyError(err, http.StatusInternalServerError, w)
		return
	}

	reply(resp, http.StatusOK, w)
}

func AccountGetHanlder(w http.ResponseWriter, r *http.Request) {
	session, err := getSession(getDB(), r)
	if err != nil {
		code := http.StatusInternalServerError
		if isRecordNotFound(err) {
			code = http.StatusUnauthorized
		}
		err = errors.Wrap(err, "Error getting user account information")
		replyError(err, code, w)
		return
	}

	var accBal AccountBalance
	err = getDB().Where("username = ?", session.Username).Find(&accBal).Error

	resp, err := json.Marshal(accBal)
	if err != nil {
		err = errors.Wrap(err, "Error encoding account balance info")
		replyError(err, http.StatusInternalServerError, w)
		return
	}

	reply(resp, http.StatusOK, w)
}
