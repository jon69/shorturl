// Модуль storage хранит информацию о URL.
package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"

	dbh "github.com/jon69/shorturl/internal/app/db"
)

// MyDelPair храние информацию о URL для удаления.
type MyDelPair struct {
	// uid - идентификатор.
	uid string
	// value - значение.
	value string
	// deleted - признак удаления.
	deleted bool
	// uid - идентификатор.
	uidI uint64
}

// StorageURL хранилище URL.
type StorageURL struct {
	// urls - множество URL.
	urls map[string]map[string]MyDelPair
	// mux - мьютекс для синхронизации.
	mux *sync.RWMutex
	// counter - счетчик всех URL.
	counter uint64
	// filePath - путь к фалу для хранения URL.
	filePath string
	// connDB - параметры подключения к БД.
	connDB string
	// restored - признак восстановления.
	restored bool
}

// NewStorage создает новое хранилище.
func NewStorage(filePath string, conndb string) *StorageURL {
	s := &StorageURL{}
	s.mux = &sync.RWMutex{}
	s.urls = make(map[string]map[string]MyDelPair)
	s.counter = 0
	s.filePath = filePath
	s.connDB = conndb
	s.restored = false
	if conndb != "" {
		dbh.CreateIfNotExist(conndb)
	}
	s.restoreFromDB()
	s.restoreFromFile()
	return s
}

func (h *StorageURL) put(user string, key string, v string, u string, del bool, uidi uint64) {
	_, ok := h.urls[user]
	if !ok {
		h.urls[user] = make(map[string]MyDelPair)
	}
	h.urls[user][key] = MyDelPair{value: v, uid: u, deleted: del, uidI: uidi}
}

func (h *StorageURL) del(user string, key string, u string) (bool, string, uint64) {
	_, ok := h.urls[user]
	if !ok {
		return false, "", 0
	}
	entry, ok2 := h.urls[user][key]
	if !ok2 {
		return false, "", 0
	}
	entry.deleted = true
	h.urls[user][key] = entry
	return true, entry.value, entry.uidI
}

func (h *StorageURL) getNewID() uint64 {
	log.Print(h.counter)
	for {
		val := atomic.LoadUint64(&h.counter)
		if atomic.CompareAndSwapUint64(&h.counter, val, val+1) {
			return h.counter
		}
	}
}

// EventDel храние информацию о удаляемых URL.
type EventDel struct {
	// User - идентификатор пользователя.
	User string `json:"user"`
	// Key - ключ.
	Key uint64 `json:"key"`
	// Value - удаляемое значение.
	Value string `json:"value"`
	// UID - индентификатор.
	UID string `json:"uid"`
	// DEL - признак удаления.
	DEL bool `json:"del"`
}

func max(value1 uint64, value2 uint64) uint64 {
	if value1 > value2 {
		return value1
	}
	return value2
}

func (h *StorageURL) restoreFromDB() {
	if h.restored {
		return
	}

	if h.connDB != "" {
		log.Print("reading urls from db...")

		data, ok := dbh.ReadURLS(h.connDB)
		if ok {
			h.restored = true
			var maxKey uint64
			maxKey = 0
			for _, url := range data {
				event := EventDel{}
				err := json.Unmarshal(url.DumpJSONURL, &event)
				event.DEL = url.Deleted
				if err == nil {
					maxKey = max(maxKey, event.Key)
					keyStr := fmt.Sprint(event.Key)
					log.Print("user  = " + event.User)
					log.Print("key   = " + keyStr)
					log.Print("value = " + event.Value)
					log.Print("uid   = " + event.UID)
					delStr := "false"
					if event.DEL {
						delStr = "true"
					}
					log.Print("del   = " + delStr)
					h.put(event.User, keyStr, event.Value, event.UID, event.DEL, event.Key)
				} else {
					log.Println("error unmarshal: " + err.Error())
				}
			}
			h.counter = max(h.counter, maxKey)
		} else {
			log.Println("can restore from db")
		}
	}
}

func (h *StorageURL) restoreFromFile() {
	if h.restored {
		return
	}
	if h.filePath != "" {
		log.Print("opening file...")
		file, err := os.OpenFile(h.filePath, os.O_RDONLY|os.O_CREATE, 0777)
		if err == nil {
			defer file.Close()
			log.Print("readin from file...")
			reader := bufio.NewReader(file)
			var maxKey uint64
			maxKey = 0
			for data, err := reader.ReadBytes('\n'); err == nil; data, err = reader.ReadBytes('\n') {
				event := EventDel{}
				err = json.Unmarshal(data, &event)
				if err == nil {
					maxKey = max(maxKey, event.Key)
					keyStr := fmt.Sprint(event.Key)
					log.Print("user  = " + event.User)
					log.Print("key   = " + keyStr)
					log.Print("value = " + event.Value)
					log.Print("uid   = " + event.UID)
					delStr := "false"
					if event.DEL {
						delStr = "true"
					}
					log.Print("del   = " + delStr)
					h.put(event.User, keyStr, event.Value, event.UID, event.DEL, event.Key)
				}
			}
			h.counter = max(h.counter, maxKey)
		} else {
			log.Print("can not open file to read: " + err.Error())
		}
	}
}

// PutUserURL сохраняет URL в хранилище.
func (h *StorageURL) PutUserURL(uid string, value string) (int, string) {
	user := "1"
	log.Print("StorageURL.PutURL user=", user)
	key := h.getNewID()

	var data []byte
	var errMarshal error

	event := EventDel{User: user, Key: key, Value: value, UID: uid, DEL: false}
	data, errMarshal = json.Marshal(&event)
	if errMarshal != nil {
		log.Print("can not json.marshal")
	} else {
		data = append(data, '\n')
	}

	strKey := fmt.Sprint(key)

	h.mux.Lock()
	defer h.mux.Unlock()

	iou := 1
	if h.connDB != "" && errMarshal == nil {
		log.Println("inserting into db...")
		var ok bool
		var su string
		ok, iou, su = dbh.InsertURL(h.connDB, data, value, strKey)
		if !ok {
			log.Println("eror insert into db")
		} else {
			strKey = su
		}
	}

	h.put(user, strKey, value, uid, false, key)

	if h.filePath != "" && errMarshal == nil && iou == 1 {
		log.Print("opening file...")
		file, err := os.OpenFile(h.filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
		if err == nil {
			defer file.Close()
			log.Print("writing to file...")
			log.Print(data)
			_, err = file.Write(data)
			if err != nil {
				log.Print("can not write to file")
			}
		} else {
			log.Print("can not open file to write: " + err.Error())
		}
	}

	return iou, strKey
}

// DelUserURL удаляет URL из хранилища.
func (h *StorageURL) DelUserURL(uid string, strKey string) bool {
	user := "1"
	log.Print("StorageURL.DelUserURL user=", user)

	go func() {

		var data []byte
		var errMarshal error

		h.mux.Lock()
		defer h.mux.Unlock()

		ok, value, key := h.del(user, strKey, uid)
		if !ok {
			log.Print("err DelUserURL can not find uid=" + uid + " strKey=" + strKey)
			return
		}

		event := EventDel{User: user, Key: key, Value: value, UID: uid, DEL: true}
		data, errMarshal = json.Marshal(&event)
		if errMarshal != nil {
			log.Print("can not json.marshal")
		} else {
			data = append(data, '\n')
		}
		if h.connDB != "" && errMarshal == nil {
			log.Println("deleting into db...")
			ok := dbh.DeleteURL(h.connDB, strKey)
			if !ok {
				log.Println("eror delete into db")
				return
			}
		}
		if h.filePath != "" && errMarshal == nil {
			log.Print("opening file...")
			file, err := os.OpenFile(h.filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
			if err == nil {
				defer file.Close()
				log.Print("writing to file...")
				log.Print(data)
				_, err = file.Write(data)
				if err != nil {
					log.Print("can not write to file")
				}
			} else {
				log.Print("can not open file to write: " + err.Error())
				return
			}
		}
	}()

	return true
}

// GetUserURL возвращает URL из хранилища на основе идентификатора.
func (h *StorageURL) GetUserURL(uid string, id string) (string, bool, bool) {
	user := "1"
	log.Print("StorageURL.GetURL user=", user)
	var val MyDelPair
	var ok bool
	ok = false

	h.mux.RLock()
	userURLS, isExist := h.urls[user]
	if isExist {
		val, ok = userURLS[id]
	}
	h.mux.RUnlock()
	return val.value, ok, val.deleted
}

// MyURLS представляет информацию о URL
type MyURLS struct {
	// ShortURL - краткая форма URL
	ShortURL string `json:"short_url"`
	// OriginalURL - исходная длинная форма URL
	OriginalURL string `json:"original_url"`
}

// GetUserURLS возвращает множество URL из хранилища.
func (h *StorageURL) GetUserURLS(uid string, url string) ([]MyURLS, []byte, bool) {
	user := "1"
	log.Print("StorageURL.GetURLS user=", user)
	h.mux.RLock()

	var urls []MyURLS
	var urlsJSON []byte
	retOK := true

	userURLS, ok := h.urls[user]
	if ok {
		for key, element := range userURLS {
			if uid == element.uid {
				urls = append(urls, MyURLS{ShortURL: url + "/" + key, OriginalURL: element.value})
			}
		}
	}

	if len(urls) != 0 {
		var err error
		urlsJSON, err = json.Marshal(urls)
		if err != nil {
			log.Print("Marshal all urls fail ", err.Error())
			retOK = false
		}
	}

	h.mux.RUnlock()
	return urls, urlsJSON, retOK
}

// PutURL сохраняет URL в хранилище.
func (h *StorageURL) PutURL(value string) (int, string) {
	return h.PutUserURL("1", value)
}

// GetURL возвращает URL из хранилища.
func (h *StorageURL) GetURL(id string) (string, bool, bool) {
	return h.GetUserURL("1", id)
}

// DelURL удаляет URL из хранилища.
func (h *StorageURL) DelURL(id string) bool {
	return h.DelUserURL("1", id)
}

// GetURLS возвращает множества URL из хранилища.
func (h *StorageURL) GetURLS(url string) ([]MyURLS, []byte, bool) {
	return h.GetUserURLS("1", url)
}
