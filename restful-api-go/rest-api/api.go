// Due to golang plugin mechanism,
// the package of function handler must be main package
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/textproto"
	"strconv"
	"time"

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

// GetPathValue g, "id"et url path parameter from the request header.
// For example, if a RESTful API is exposed with path `/guestbook/{id}`
// The fission router will extract and attach the value of `id`
// to the request header with key `X-Fission-Params-Id`.
// Then, the function can retrieve the value of `id` from the header with key.
// ** Notice **:
// The case of first letter after each dash(-) will be transform to lowercase.
// For example: 'id' -> 'Id', 'fooBAR' -> 'Foobar', 'foo-BAR' -> 'Foo-Bar'.
func GetPathValue(r *http.Request, param string) string {
	// transform text case
	// For example: 'id' -> 'Id', 'fooBAR' -> 'Foobar', 'foo-BAR' -> 'Foo-Bar'.
	param = textproto.CanonicalMIMEHeaderKey(param)

	// generate header key for accessing request header value
	key := fmt.Sprintf("%s-%s", URL_PARAMS_PREFIX, param)

	// get header value
	return r.Header.Get(key)
}

// GetQueryString get query value from the request query string.
// Unlike url path, the fission router leaves request query intact,
// which means that the case of query string will not change nor
// need to access it from request header.
// For example: If a request come with uri '/guestbook/123?start=123&end=456',
// the function will see `start=123&end=456`.
func GetQueryString(r *http.Request, query string) string {
	return r.URL.Query().Get(query)
}

// getMessageId get message id from request
func getMessageId(r *http.Request) (uint, error) {
	msgIdStr := GetPathValue(r, "id")
	if len(msgIdStr) > 0 {
		id, err := strconv.ParseUint(msgIdStr, 10, 64)
		if err != nil {
			return 0, err
		}
		return uint(id), nil
	}
	return 0, nil
}

// getDate get date from query string
func getDate(r *http.Request, query string) (*time.Time, error) {
	str := GetQueryString(r, query)

	if len(str) == 0 {
		return nil, nil
	}

	second, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return nil, err
	}

	t := time.Unix(second, 0)

	return &t, nil
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
	msg.Timestamp = time.Now().Unix()

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
	// GET guestbook/messages/?start=ooo&end=xxx

	msgId, err := getMessageId(r)
	if err != nil {
		err = errors.Wrap(err, "Error parsing message id")
		log.Println(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	// extract 'start' and 'end' from query string `start=xxx&end=ooo`
	start, err := getDate(r, "start")
	if err != nil {
		err = errors.Wrap(err, "Error parsing start value")
	}

	end, err := getDate(r, "end")
	if err != nil {
		err = errors.Wrap(err, "Error parsing end value")
	}

	var respObj interface{}

	if msgId != 0 && start == nil && end == nil {
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

		if start == nil && end == nil {
			err = GetDB().Find(&msgs).Error
		} else {
			err = GetDB().Where("timestamp >= ? AND timestamp <= ?", start.Unix(), end.Unix()).
				Find(&msgs).Error
		}

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

	msgId, err := getMessageId(r)
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
	// DELETE guestbook/messages/{id}

	msgId, err := getMessageId(r)
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
