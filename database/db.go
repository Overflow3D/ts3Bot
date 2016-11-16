package database

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/boltdb/bolt"
)

//DB , database struct
type DB struct {
	conn *bolt.DB
}

//Datastore , db interface
type Datastore interface {
	CreateBuckets(bucket string) error
	CreateSubBuckets(mBucket, sBucket string) error
	AddRecord(bucket, key string, v interface{}) error
	GetRecord(bucket, key string) ([]byte, error)
	DeleteRecord(bucket, key string) error

	AddRecordSubBucket(mainBucket, subBucket, key string, v interface{}) error
	GetRecordSubBucket(mainBucket, subBucket, key string) ([]byte, error)
	DeleteRecordSubBucket(mainBucket, subBucket, key string) error

	Close()
}

//NewConn , create new db connection
func NewConn() (*DB, error) {
	db, err := bolt.Open("./ts3.db", 0664, nil)
	if err != nil {
		return nil, err
	}
	return &DB{conn: db}, nil
}

//CreateBuckets , in case if database is deleted
func (db *DB) CreateBuckets(bucket string) error {
	err := db.conn.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

//CreateSubBuckets , creates sub bucket of bucket
func (db *DB) CreateSubBuckets(mBucket, sBucket string) error {
	err := db.conn.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(mBucket))
		if err != nil {
			return err
		}
		b.CreateBucketIfNotExists([]byte(sBucket))
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

//Close ,closes db con
func (db *DB) Close() {
	db.conn.Close()
}

//AddRecord ,  adds new key to database
func (db *DB) AddRecord(bucket, key string, v interface{}) error {
	err := db.conn.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}

		data, errx := marshalJSON(v)
		if errx != nil {
			return errx
		}
		err = bucket.Put([]byte(key), data)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

//GetRecord , gets key from database
func (db *DB) GetRecord(bucket, key string) ([]byte, error) {
	var buffer bytes.Buffer
	err := db.conn.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucket))
		if bucket == nil {
			return fmt.Errorf("Bucket doesn't exists")
		}

		buffer.Write(bucket.Get([]byte(key)))

		return nil
	})
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

//DeleteRecord , deletes key from database
func (db *DB) DeleteRecord(bucket, key string) error {
	err := db.conn.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		b.Delete([]byte(key))
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

//AddRecordSubBucket , adds record to sub bucket in database
func (db *DB) AddRecordSubBucket(mainBucket, subBucket, key string, v interface{}) error {
	err := db.conn.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(mainBucket))
		if err != nil {
			return err
		}

		data, errx := marshalJSON(v)
		if errx != nil {
			return errx
		}
		b, e := bucket.CreateBucketIfNotExists([]byte(subBucket))
		if e != nil {
			return e
		}
		err = b.Put([]byte(key), data)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

//GetRecordSubBucket , get key from sub bucket
func (db *DB) GetRecordSubBucket(mainBucket, subBucket, key string) ([]byte, error) {
	var buffer bytes.Buffer
	err := db.conn.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(mainBucket))
		if bucket == nil {
			return fmt.Errorf("MainBucket doesn't exists")
		}
		b := bucket.Bucket([]byte(subBucket))
		if b == nil {
			return fmt.Errorf("SubBucket doesn't exists")
		}
		buffer.Write(b.Get([]byte(key)))

		return nil
	})
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

//DeleteRecordSubBucket , get key from sub bucket
func (db *DB) DeleteRecordSubBucket(mainBucket, subBucket, key string) error {
	err := db.conn.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(mainBucket))
		if bucket == nil {
			return fmt.Errorf("MainBucket doesn't exists")
		}
		b := bucket.Bucket([]byte(subBucket))
		if b == nil {
			return fmt.Errorf("SubBucket doesn't exists")
		}
		b.Delete([]byte(key))

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func marshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
