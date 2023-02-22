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

type MyDelPair struct {
	uid     string
	value   string
	deleted bool
	uidI    uint64
}

type StorageURL struct {
	urls     map[string]map[string]MyDelPair
	mux      *sync.RWMutex
	counter  uint64
	filePath string
	connDB   string
	restored bool
}

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
	_, ok2 := h.urls[user][key]
	if !ok2 {
		return false, "", 0
	}
	return true, h.urls[user][key].value, h.urls[user][key].uidI
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

type EventDel struct {
	User  string `json:"user"`
	Key   uint64 `json:"key"`
	Value string `json:"value"`
	UID   string `json:"uid"`
	DEL   bool   `json:"del"`
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
				err := json.Unmarshal(url.DumpJsonURL, &event)
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

func (h *StorageURL) DelUserURL(uid string, strKey string) bool {
	user := "1"
	log.Print("StorageURL.DelUserURL user=", user)

	var data []byte
	var errMarshal error

	h.mux.Lock()
	defer h.mux.Unlock()

	ok, value, key := h.del(user, strKey, uid)
	if !ok {
		log.Print("err DelUserURL can not find uid=" + uid + " strKey=" + strKey)
		return false
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
		var ok bool
		ok = dbh.DeleteURL(h.connDB, strKey)
		if !ok {
			log.Println("eror delete into db")
			return false
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
			return false
		}
	}

	return true
}

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

type MyURLS struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func (h *StorageURL) GetUserURLS(uid string, url string) ([]byte, bool) {
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
	return urlsJSON, retOK
}

func (h *StorageURL) PutURL(value string) (int, string) {
	return h.PutUserURL("1", value)
}

func (h *StorageURL) GetURL(id string) (string, bool, bool) {
	return h.GetUserURL("1", id)
}

func (h *StorageURL) DelURL(id string) bool {
	return h.DelUserURL("1", id)
}

func (h *StorageURL) GetURLS(url string) ([]byte, bool) {
	return h.GetUserURLS("1", url)
}
