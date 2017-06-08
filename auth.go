package main

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/garyburd/redigo/redis"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"strings"
	"time"
)

// gets you a token if you pass the right credentials
func login(w http.ResponseWriter, r *http.Request) {
	var err error
	w = setCors(w)
	if r.Method != "POST" {
		http.Error(w, fmt.Sprintf("{ \"error\": \"%s\" }", "Forbidden request"), 403)
		return
	}
	conn := POOL.Get()
	defer conn.Close()
	email := strings.ToLower(r.FormValue("email"))
	password, err := redis.Bytes(conn.Do("GET", email))
	if err == nil {
		// compare passwords
		err = bcrypt.CompareHashAndPassword(password, []byte(r.FormValue("password")))
		// if it doesn't match
		if err != nil {
			http.Error(w, fmt.Sprintf("{ \"error\": \"%s\" }", "Wrong password"), 401)
			return
		}
		token := jwt.New(jwt.SigningMethodHS256)
		claims := token.Claims.(jwt.MapClaims)
		claims["admin"] = false
		claims["email"] = email
		// 24 hour token
		claims["exp"] = time.Now().Add(time.Hour * 24).Unix()
		tokenString, _ := token.SignedString(SIGN_KEY)
		w.Write([]byte(fmt.Sprintf("{ \"access_token\": \"%s\" }", tokenString)))
	} else {
		// email not found
		http.Error(w, fmt.Sprintf("{ \"error\": \"%s\" }", "Email not found"), 401)
		return
	}
}

// register a new user, gives you a token if the email -> password
// is not registered already
func register(w http.ResponseWriter, r *http.Request) {
	var err error
	w = setCors(w)
	if r.Method != "POST" {
		http.Error(w, fmt.Sprintf("{ \"error\": \"%s\" }", "Forbidden request"), 403)
		return
	}
	conn := POOL.Get()
	defer conn.Close()
	email := strings.ToLower(r.FormValue("email"))
	// check if the user is already registered
	exists, err := redis.Bool(conn.Do("EXISTS", email))
	if exists {
		w.Write([]byte(fmt.Sprintf("{ \"error\": \"%s\" }", "Email taken")))
		return
	}
	// get password from the post request form value
	password := r.FormValue("password")
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password),
		bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	// Set user -> password in redis
	_, err = conn.Do("SET", email, string(hashedPassword[:]))
	if err != nil {
		log.Println(err)
	}
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["admin"] = false
	claims["email"] = email
	// 24 hour token
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()
	tokenString, _ := token.SignedString(SIGN_KEY)
	w.Write([]byte(fmt.Sprintf("{ \"access_token\": \"%s\" }", tokenString)))
}

func setCors(w http.ResponseWriter) http.ResponseWriter {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers",
		"Accept, 0, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	w.Header().Set("Content-Type", "application/json")
	return w
}
