package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/pkg/errors"
)

func setCORS(w http.ResponseWriter) http.ResponseWriter {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	w.Header().Set("Access-Control-Expose-Headers", "*")
	return w
}

func getAuthFromHeader(r *http.Request) (string, string, error) {
	// <username>:<token>
	authVal := r.Header.Get("Authentication")
	auth := strings.Split(authVal, ":")
	if len(auth) != 2 || len(auth[0]) == 0 || len(auth[1]) == 0 {
		return "", "", errors.New("Invalid auth header")
	}
	return auth[0], auth[1], nil
}

func getSession(db *gorm.DB, r *http.Request) (*Session, error) {
	username, token, err := getAuthFromHeader(r)
	if err != nil {
		return nil, err
	}
	sess := Session{}
	where := "token = ? AND username = ? AND expiration > ?"
	err = db.Where(where, token, username, time.Now().Unix()).First(&sess).Error
	if isRecordNotFound(err) {
		err = errors.New("Invalid/Expired auth token, please login first to get new auth token")
	}
	return &sess, err
}

func getAccountInfoByName(db *gorm.DB, username string) (*Account, error) {
	var user Account
	err := db.Where(&Account{Username: username}).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func isRecordNotFound(err error) bool {
	return gorm.IsRecordNotFoundError(err)
}

func replyError(err error, code int, w http.ResponseWriter) {
	if err == nil {
		return
	}
	log.Println(err)
	//w.Header().Set("Content-Type", "application/json")
	w = setCORS(w)
	http.Error(w, err.Error(), code)
}

func reply(msg []byte, code int, w http.ResponseWriter) {
	w = setCORS(w)
	w.WriteHeader(code)
	w.Write(msg)
}

func replyJson(obj interface{}, code int, w http.ResponseWriter) {
	resp, err := json.Marshal(obj)
	if err != nil {
		code = http.StatusInternalServerError
		reply([]byte(err.Error()), code, w)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	reply(resp, code, w)
}
