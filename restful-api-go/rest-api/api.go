// Due to golang plugin mechanism,
// the package of function handler must be main package
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

const (
	URL_PARAMS_PREFIX = "X-Fission-Params"
)

var (
	dbConn *gorm.DB
)

func init() {
	if dbConn == nil {
		dbUrl := "postgresql://root@cockroachdb.guestbook:26257/guestbook?sslmode=disable"
		dbName := "guestbook"
		dbConn = ConnectDB(dbUrl, dbName)
	}
}

func GetDB() *gorm.DB {
	return dbConn
}

func GetMessageID(r *http.Request) (uint, error) {
	msgIdStr := r.Header.Get(fmt.Sprintf("%s-%s", URL_PARAMS_PREFIX, "Id"))
	if len(msgIdStr) > 0 {
		id, err := strconv.ParseUint(msgIdStr, 10, 64)
		if err != nil {
			return 0, err
		}
		return uint(id), nil
	}
	return 0, nil
}

func MessagePostHandler(w http.ResponseWriter, r *http.Request) {
	// POST guestbook/messages

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		err = errors.Wrap(err, "Error reading request body")
		log.Println(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	msg := Message{}
	err = json.Unmarshal(body, &msg)
	if err != nil {
		err = errors.Wrap(err, "Error decoding json body")
		log.Println(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	// Database will assign id to the id field automatically,
	// remove user assigned value.
	if msg.ID != 0 {
		msg.ID = 0
	}

	err = GetDB().Create(&msg).Error
	if err != nil {
		err = errors.Wrap(err, "Error inserting message to database")
		log.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func MessageGetHandler(w http.ResponseWriter, r *http.Request) {
	// GET guestbook/messages
	// GET guestbook/messages/{id}

	msgId, err := GetMessageID(r)
	if err != nil {
		err = errors.Wrap(err, "Error parsing message id")
		log.Println(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	var respObj interface{}

	if msgId != 0 {
		msg := Message{}
		err := GetDB().First(&msg, msgId).Error
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("Error finding message with message id %v", msgId))
			log.Println(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		respObj = msg
	} else {
		msgs := []Message{}
		err := GetDB().Find(&msgs).Error
		if err != nil {
			err = errors.Wrap(err, "Error retrieving messages")
			log.Println(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		respObj = msgs
	}

	body, err := json.Marshal(respObj)
	if err != nil {
		err = errors.Wrap(err, "Error encoding response body")
		log.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

func MessageUpdateHandler(w http.ResponseWriter, r *http.Request) {
	// PUT guestbook/messages/{id}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		err = errors.Wrap(err, "Error reading request body")
		log.Println(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	msg := Message{}
	err = json.Unmarshal(body, &msg)
	if err != nil {
		err = errors.Wrap(err, "Error decoding json body")
		log.Println(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	// Database will assign id to the id field automatically,
	// remove user assigned value.
	if msg.ID != 0 {
		msg.ID = 0
	}

	msgId, err := GetMessageID(r)
	if err != nil {
		err = errors.Wrap(err, "Error parsing message id")
		log.Println(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	err = GetDB().Model(&Message{}).Where("id = ?", msgId).Update("message", msg.Message).Error
	if err != nil {
		err = errors.Wrap(err, "Error updating message")
		log.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func MessageDeleteHandler(w http.ResponseWriter, r *http.Request) {
	msgId, err := GetMessageID(r)
	if err != nil {
		err = errors.Wrap(err, "Error parsing message id")
		log.Println(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	err = GetDB().Where("id = ?", msgId).Delete(&Message{}).Error
	if err != nil {
		err = errors.Wrap(err, "Error deleting message")
		log.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
}
