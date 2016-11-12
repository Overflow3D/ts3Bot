package database

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/boltdb/bolt"
)

//DB , database struct
type DB struct {
	conn *bolt.DB
}

//Datastore , db interface
type Datastore interface {
	CreateBuckets() error
	AddRoom(cid []byte, data []byte) error
	ReadRooms() (map[string][]byte, error)
	AddNewUser(clidb string, v interface{})
	GetUser(clidb string) ([]byte, error)
	DeleteUser(clidb string) error
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

//In case if database is deleted
func (db *DB) CreateBuckets() error {
	err := db.conn.View(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte("rooms"))
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

//Close ,closes db con
func (db *DB) Close() {
	db.conn.Close()
}

//GetUser , gets users
func (db *DB) GetUser(clidb string) ([]byte, error) {
	var data []byte
	err := db.conn.View(func(tx *bolt.Tx) error {
		var err error
		b := tx.Bucket([]byte("users"))
		k := []byte(clidb)
		data = b.Get(k)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (db *DB) DeleteUser(clidb string) error {
	err := db.conn.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		b.Delete([]byte(clidb))
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

//AddNewUser , adduser to db
func (db *DB) AddNewUser(clidb string, v interface{}) {
	err := db.conn.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return err
		}

		data, errx := marshalJSON(v)
		if errx != nil {
			return errx
		}
		err = bucket.Put([]byte(clidb), data)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

}

//AddRoom , adds room to database
func (db *DB) AddRoom(cid []byte, data []byte) error {
	err := db.conn.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("rooms"))
		if err != nil {
			return err
		}
		err = bucket.Put(cid, data)
		if err != nil {
			return err
		}
		return nil
	})

	return err
}

//GetRoom , returns room with certain cid
func (db *DB) GetRoom(cid []byte) ([]byte, error) {
	var buffer bytes.Buffer
	err := db.conn.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("rooms"))
		if bucket == nil {
			return fmt.Errorf("Bucket rooms")
		}

		buffer.Write(bucket.Get(cid))

		return nil
	})
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

//ReadRooms , reads all room from database
func (db *DB) ReadRooms() (map[string][]byte, error) {
	start := time.Now()
	defer func() {
		log.Println(time.Since(start))
	}()
	channels := make(map[string][]byte)
	err := db.conn.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("rooms"))

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			channels[string(k)] = v
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return channels, nil
}

func marshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
