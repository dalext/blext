package main

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
)

// Unused
func templates(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	w = setCors(w)
}

// basic template is saved as default when creating new docs
var basicTemplate = []byte("{\"doc\":{\"type\":\"doc\",\"content\":[{\"type\":\"paragraph\",\"content\":[{\"type\":\"text\",\"text\":\"Empty document\"}]}]},\"users\":1,\"version\":3104,\"comments\":[],\"commentVersion\":39}")

// Set and Get a document instance
func docHistory(w http.ResponseWriter, r *http.Request) {
	var err error
	vars := mux.Vars(r)
	conn := POOL.Get()
	defer conn.Close()
	w = setCors(w)
	if r.Method == "GET" { // get doc instance
		log.Println("GET history for: " + vars["docHash"])
		docJson, err := redis.Bytes(conn.Do("HGET", vars["docHash"], "json"))
		// if document is not found
		if err != nil {
			// create new document for this hash
			_, err = conn.Do("HSET", vars["docHash"], "json", basicTemplate)
			if err != nil {
				log.Println(err)
			}
			w.Write(basicTemplate)
			log.Println(err)
		} else {
			w.Write(docJson)
		}
	} else if r.Method == "PUT" { // set doc instance
		log.Println("Put request for: " + vars["docHash"])
		docJson, _ := ioutil.ReadAll(r.Body)
		_, err = conn.Do("HSET", vars["docHash"], "json", docJson)
		if err != nil {
			log.Println(err)
		}
		w.Write([]byte(docJson))
	}
}

// Channel to save messages to the database
func saveDocInstances(m *chan DBDocInstance) {
	for {
		message, ok := <-*m
		if !ok {
			log.Println("Error when trying to save")
			return
		}
		saveDocInstance(&message)
	}
}

// saves a document instance to the database
func saveDocInstance(msg *DBDocInstance) {
	var err error
	conn := POOL.Get()
	if err != nil {
		log.Println(err)
	}
	log.Println("Got doc instance, saving")
	defer conn.Close()
	_, err = conn.Do("HSET", msg.DocHash, "json", msg.Json)
	if err != nil {
		log.Println(err)
	}
}

// set and get doc instances ids / hashes for email hashes
// a@a.com (hashed) => hash => eyJhbGciOiJIUzI1N... => ["eyJ...", "zI1N..."]
//
// 2 way hash? Yes (for now)
//
// accepts GET, POST, PUT and CORS preflight
//
// example:
// Get all client's docs
// GET base/instances/eyJhbGciOiJIUzI1N...
// returns => ["eyJ...", "zI1N..."]
// Add a client doc (new doc)
// POST base/instances/eyJhbGciOiJIUzI1N...
// returns => ["eyJ...", "zI1N..."]
//
// TODO: Avoid dupes by checking that the document hash is not already
// in the list
func clientInstances(w http.ResponseWriter, r *http.Request) {
	w = setCors(w)
	vars := mux.Vars(r)
	conn := POOL.Get()
	defer conn.Close()
	email := vars["emailHash"]
	// double hash (comes hashed)
	emailHash := fmt.Sprintf("%x", sha1.Sum([]byte(email)))
	// set / update doc hashes for a client, passing a new doc hash
	// in the form
	if r.Method == "POST" {
		newDocHash := r.FormValue("docHash")
		log.Println("POST new client doc with hash " + newDocHash)
		// get the current document hash list for the client
		docList, err := redis.Bytes(conn.Do("HGET", string(emailHash), "docs"))
		// list found
		if err == nil {
			var docs []string
			// get client docs as a list of strings in docs
			err := json.Unmarshal(docList, &docs)
			// append new docHash
			docs = append(docs, newDocHash)
			docsJson, err := json.Marshal(docs)
			if err != nil {
				log.Println(err)
			}
			// save it
			_, err = conn.Do("HSET", string(emailHash), "docs", docsJson)
			w.Write([]byte(docsJson))
		} else {
			// client doesn't have docs, create a new list with the doc hash as the
			// only value
			//
			// no need to initialize it because it will be initialized
			// when you request the docHash
			//
			// save
			docBytes := []byte(fmt.Sprintf("[\"%s\"]", newDocHash))
			_, err := conn.Do("HSET", string(emailHash), "docs", docBytes)
			if err != nil {
				log.Println(err)
			}
			w.Write(docBytes)
		}
	}
	// get client doc hashes
	if r.Method == "GET" {
		// todo
		log.Println("GET client docs")
		docList, err := redis.Bytes(conn.Do("HGET", string(emailHash), "docs"))
		if err != nil {
			// This client doesn't have any docs yet
			w.Write([]byte("[]"))
		} else {
			w.Write(docList)
		}
	}
}
