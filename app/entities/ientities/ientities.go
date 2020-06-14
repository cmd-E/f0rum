package ientities

import (
	"log"
	"net/http"

	"github.com/IrvinIrvin/forum/app/contentmanager/threadsmanager"
	"github.com/IrvinIrvin/forum/app/contentmanager/usermanager"
)

// IndexStruct - struct for index page
type IndexStruct struct {
	Name, Email string                  // users credentials
	Threads     []threadsmanager.Thread // threads from db
}

const tag = "indexentities"

// SetIndexStruct - sets struct for index page
func SetIndexStruct(r *http.Request) IndexStruct {
	log.Printf(tag + " /// SetIndexFunc")
	user := usermanager.GetUser(r)
	//log.Printf("User in setindexstruct: %v", user)
	Threads := threadsmanager.GetThreads()
	indexStruct := IndexStruct{Name: user.Name, Email: user.Email, Threads: Threads}
	return indexStruct
}
