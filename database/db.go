package db

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
	AddNewUser()
	LoadUserFromDB() map[string]*UserDB
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

//UserDB , ...
type UserDB struct {
	Clidb   string
	Clid    string
	Moves   *Moves
	Perm    int
	IsAdmin bool
}

//Moves , how much time user moved
type Moves struct {
	Number    int
	SinceMove time.Time
	Warnings  int
}

//LoadUserFromDB , load users from db
func (db *DB) LoadUserFromDB() map[string]*UserDB {
	users := make(map[string]*UserDB)
	err := db.conn.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte("users"))

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			user := &UserDB{}
			key := string(k)
			err := user.unMarshalJSON(v)
			if err != nil {
				return err
			}
			users[key] = user

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
func (db *DB) AddNewUser() {
	err := db.conn.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return err
		}

		user := &UserDB{
			Clidb: "666",
			Clid:  "55",
			Moves: &Moves{
				Number:    0,
				SinceMove: time.Now(),
				Warnings:  0,
			},
			Perm:    111,
			IsAdmin: false,
		}
		data, errx := user.marshalJSON()
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

func (u *UserDB) unMarshalJSON(data []byte) error {
	return json.Unmarshal(data, &u)
}

func (u *UserDB) marshalJSON() ([]byte, error) {
	return json.Marshal(&u)
}
