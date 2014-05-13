package main

import (
	"errors"
)

type Users interface {
	Add(User) error
	Delete(string) error
 	Exists(string) bool
	Info(string) (User, error)
}

type User struct {
	Id string
	Name string
	Avatar string
	Email string
	Link string
}

// For now Uid == Id
type UserDB struct {
	Uid map[string]User
}

func (udb UserDB) Add(u User) error {
	_, exists := udb.Uid[u.Id]
	if exists {
		return errors.New("Could not add user: ID aldready in the database.")
	}
	udb.Uid[u.Id] = u
	return nil
}

func (udb UserDB) Delete(id string) error {
	_, exists := udb.Uid[id]
        if exists {
                return errors.New("Could not delete user: not found in database.")
        }
        delete(udb.Uid, id)
	return nil
}

func (udb UserDB) Exists(id string) (exists bool) {
	_, exists = udb.Uid[id]
	return
}

func (udb UserDB) Info(id string) (User, error) {
	if ! udb.Exists(id) {
		return User{"", "", "", "", ""}, errors.New("Could not find user in database.")
	}
	return udb.Uid[id], nil
}










