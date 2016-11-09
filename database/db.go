package database

import (
	"encoding/json"
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
	AddNewUser(clid string, clidb string)
	LoadUserFromDB() map[string][]byte
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

//Close ,closes db con
func (db *DB) Close() {
	db.conn.Close()
}

//GetUser , gets users
func (db *DB) GetUser(clidb string) ([]byte, error) {
	var data []byte
	err := db.conn.View(func(tx *bolt.Tx) error {
		var err error
		b := tx.Bucket([]byte("room"))
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

//LoadUserFromDB , load users from db
func (db *DB) LoadUserFromDB() map[string][]byte {
	users := make(map[string][]byte)
	err := db.conn.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte("users"))

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			key := string(k)
			users[key] = v

		}

		return nil
	})
	if err != nil {
		return nil
	}
	log.Println("Wczytano", len(users))
	return users
}

//AddNewUser , adduser to db
func (db *DB) AddNewUser(clid string, clidb string) {
	err := db.conn.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return err
		}

		moves := struct {
			Number    int
			SinceMove time.Time
			Warnings  int
		}{
			0,
			time.Now(),
			0,
		}

		user := struct {
			Clidb   string
			Clid    string
			Moves   interface{}
			Perm    int
			IsAdmin bool
		}{
			clid,
			clidb,
			moves,
			1,
			false,
		}

		data, errx := marshalJSON(user)
		if errx != nil {
			return errx
		}
		err = bucket.Put([]byte(user.Clidb), data)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
}

func marshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
