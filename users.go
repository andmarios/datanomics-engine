/*
Datanomics™ — A web sink for your sensors
Copyright (C) 2014, Marios Andreopoulos

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

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
	Id     string
	Name   string
	Avatar string
	Email  string
	Link   string
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
	if !udb.Exists(id) {
		return User{"", "", "", "", ""}, errors.New("Could not find user in database.")
	}
	return udb.Uid[id], nil
}
