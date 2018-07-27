package main

import (
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

type User struct {
	Created          float32 `json:"created"`
	Created_utc      float32 `json:"created_utc"`
	Has_mail         bool    `json:"has_mail"`
	Id               string  `json:"id"`
	Is_mod           bool    `json:"is_mod"`
	Name             string  `json:"name"`
	Level            int     `json:"level"`
	Active           bool    `json:"active"`
	Activation_token string  `json:"activation_token"`
	Created_at       string  `json:"created_at"`
	Auth             ServerAuth
	IP               string
}

type DocInstance struct {
	Level        int       `json:"level"`
	Text         string    `json:"text"`
	UserName     string    `json:"name"`
	ChatRoomName string    `json:"room_name"`
	Timestamp    time.Time `json:"timestamp,omitempty"`
}

type DBDocInstance struct {
	DocHash string // id
	Json    []byte
}

type ServerAuth struct {
	Access_token string `json:"access_token"`
	Token_type   string `json:"token_type"`
	Expires_in   int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

// env
var DOMAIN = "192.168.1.47"
var MPORT = "5555"
var SERVER_ADDRESS = "http://192.168.1.47:5555"
var COOKIE_NAME = "dalext"
var PROJ_ROOT = ""
var SIGN_KEY = []byte("secret")

// mem
var users map[string]User
var AuthorizedIps []string
var MessageChannel chan DBDocInstance

// Declare a global variable to store the Redis connection pool.
var POOL *redis.Pool

func init() {
	// set root directory
	ROOT, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	PROJ_ROOT = ROOT
	// Establish a pool of 5 Redis connections to the Redis server
	POOL = newPool("localhost:6379")
	// set JWT key
	SIGN_KEY = []byte(os.Getenv("SIGN_KEY"))
}

func newPool(addr string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     5,
		IdleTimeout: 240 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", addr) },
	}
}

func main() {
	MessageChannel = make(chan DBDocInstance, 256)
	// a goroutine for saving messages
	go saveDocInstances(&MessageChannel)
	// for keeping track of users in memory
	users = make(map[string]User)
	r := mux.NewRouter()
	hub := newHub()
	go hub.run()
	// auth
	r.HandleFunc("/login", login)
	r.HandleFunc("/register", register)
	// files
	r.HandleFunc("/convert/{type}", convert)
	// editor
	r.HandleFunc("/history/{docHash}", docHistory)
	r.HandleFunc("/instances/{emailHash}", clientInstances)
	r.HandleFunc("/collab_socket/{docHash}",
		func(w http.ResponseWriter, r *http.Request) {
			collabSockets(hub, w, r)
		})
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir(PROJ_ROOT+"/icons"))))
	// server
	srv := &http.Server{
		Handler:      r,
		Addr:         ":" + MPORT,
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
	}
	err := srv.ListenAndServe()
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func getRandomString(n int) string {
	b := make([]byte, n)
	src := rand.NewSource(time.Now().UnixNano())
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
