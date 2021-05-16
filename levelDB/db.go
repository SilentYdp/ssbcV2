package levelDB

import (
	"github.com/cloudflare/cfssl/log"
	"github.com/syndtr/goleveldb/leveldb"
)

var db *leveldb.DB
var err error

func InitDB(path string) {
	db, err = leveldb.OpenFile("db/path/"+path, nil)
	if err != nil {
		log.Error("db init err", err)
	}
}

func DBGet(key string) []byte {
	data, err := db.Get([]byte(key), nil)
	if err != nil {
		log.Error("db get err", err)
		return nil
	}
	return data
}

func DBPut(key string, value []byte) {
	err = db.Put([]byte(key), value, nil)
	if err != nil {
		log.Error("db put err", err)
	}
}

func DBDelete(key string) {
	err = db.Delete([]byte(key), nil)
	if err != nil {
		log.Error("db delete err", err)
	}
}
