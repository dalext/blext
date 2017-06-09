package main

import (
	// "crypto/sha1"
	// "encoding/json"
	"fmt"
	"log"
	// "github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	"io"
	// "log"
	"bytes"
	"net/http"
	"os"
	"os/exec"
)

// Should respond with the URL for the uploaded document after being processed
//
// Examples
//
// POST /files/{WordType} ->
// File is converted from WordType to MarkDown
// File is converted from MarkDown to prosemirror
// Prosemirror file is loaded in the document
//
func upload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fmt.Println("method:", r.Method)
	// get request for this route prints filetype parameter
	if r.Method == "GET" {
		w.Write([]byte(vars["type"] + "\n"))
	} else {
		r.ParseMultipartForm(32 << 20)
		file, handler, err := r.FormFile("file")
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()
		fmt.Fprintf(w, "%v", handler.Header)
		// create empty file
		f, err := os.OpenFile("./test/"+r.FormValue("filename"),
			os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer f.Close()
		// write file to empty file
		io.Copy(f, file)
		cmd := exec.Command("pandoc", "./test/"+r.FormValue("filename"),
			"-s", "-o", "./test/"+r.FormValue("filename")+".pdf")
		var out bytes.Buffer
		cmd.Stdout = &out
		err = cmd.Run()
		if err != nil {
			log.Println(err)
		}
		fmt.Printf("File converted")
	}
}
