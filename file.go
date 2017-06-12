// For file operations
//
// Upload files
// Convert files
// Return user files
// Delete files
// Rename files
//
// Each user has a folder name as zbase32 of their email address
//

package main

import (
	// "crypto/sha1"
	// "encoding/json"
	"fmt"
	"github.com/tv42/zbase32"
	"log"
	// "github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	"io"
	// "log"
	"bytes"
	"net/http"
	"os"
	"os/exec"
	"strings"
	// "path/filepath"
)

// Should respond with the filename... ?
//
// Examples
//
// POST /files/{WordType} ->
// File is converted from WordType to MarkDown
// File is converted from MarkDown to prosemirror
// Prosemirror file is loaded in the document
//
func convert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fmt.Println("method:", r.Method)
	// get request for this route prints FileType parameter
	if r.Method == "GET" {
		w.Write([]byte(vars["type"] + "\n"))
	} else {
		email := r.FormValue("email")
		filename := r.FormValue("filename")
		// email is used for finding the user folder
		if email == "" {
			http.Error(w, fmt.Sprintf("{ \"error\": \"%s\" }",
				"Email not found"), 401)
			return
		}
		// only supports .md files for now
		if strings.Split(filename, ".")[1] != "md" {
			http.Error(w, fmt.Sprintf("{ \"error\": \"%s\" }",
				"File not supported"), 401)
			return
		}
		// parse request body as multipart/form-data
		r.ParseMultipartForm(32 << 20)
		// read file from the form
		file, _, err := r.FormFile("file")
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()
		// write to w, as %v, file handler header
		// w is the http.ResponseWriter
		// fmt.Fprintf(w, "%v", handler.Header) // ??
		// log.Printf("%v", handler.Header)
		// generate user folder path
		// each user has a folder with the name as zbase32 email encoded
		userFolderPath := PROJ_ROOT + "/users/" +
			zbase32.EncodeToString([]byte(email)) + "/"
		// create/open empty file (at location)
		// first check if the user has their folder already created
		if _, err := os.Stat(userFolderPath); os.IsNotExist(err) {
			// if not, create it
			fmt.Println("Creating folder for user at: " + userFolderPath)
			os.Mkdir(userFolderPath, 0666)
		}
		// open/create an empty file at the user folder with the file name
		// passed in the request
		f, err := os.OpenFile(userFolderPath+filename,
			os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer f.Close()
		// save file
		tempFileName := userFolderPath + filename + ".pdf"
		resultFileName := userFolderPath + strings.Split(filename, ".")[0] + ".pdf"
		io.Copy(f, file)
		// compose file conversion command
		cmd := exec.Command("pandoc", userFolderPath+filename,
			"-s", "-o", tempFileName)
		var out bytes.Buffer
		cmd.Stdout = &out
		// Run command and assign output of command to &out
		err = cmd.Run()
		if err != nil {
			log.Println(err)
			return
		}
		// rename file to the converted filetype
		err = os.Rename(tempFileName, resultFileName)
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Printf("File converted!")
		// respond with the filename as json
		w.Write([]byte(fmt.Sprintf("{ \"filename\": \"%s\" }", filename)))
	}
}
