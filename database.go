package uptime

import (
	"log"
	"time"

	"go.etcd.io/bbolt"
)

var (
	//DB     *bbolt.DB
	old    string
	Tables = []Table{
		{Name: "status", Path: []string{"status"}},
	}
)

type Table struct {
	Name string
	Path []string
}

func OpenDB(file string) (*bbolt.DB, error) {
	//var err error
	//if err := CloseDB(); err != nil {
	//return nil, err
	//}
	DB, err := bbolt.Open(file, 0666, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, err
	}
	//old = file
	log.Println("loaded db file", file)
	//if err := createTables(Tables); err != nil {
	//return fmt.Errorf("create tables %w", err)
	//}
	return DB, nil
}

func createBucket(path []string, tx *bbolt.Tx) *bbolt.Bucket {
	if path == nil {
		return nil
	}
	bucket, err := tx.CreateBucketIfNotExists([]byte(path[0]))
	if err != nil {
		log.Println("create root bucket", path[0], err)
		return nil
	}
	for _, name := range path[1:] {
		bucket, err = tx.CreateBucket([]byte(name))
		if err != nil {
			log.Println("create nested bucket", name, err)
			return nil
		}
	}
	return bucket
}

func AddKey(db *bbolt.DB, name string, path []string, value []byte) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := createBucket(path, tx)
		return bucket.Put([]byte(name), value)
	})
}
