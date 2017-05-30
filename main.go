package main

import (
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
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
var DOMAIN = "52.58.76.202"
var MPORT = "5555"
var SERVER_ADDRESS = "http://52.58.76.202:5555"
var COOKIE_NAME = "backmirror"
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

// Checks the cookie in the request, if the cookie is not found or the value
// is not found in the server memory map, then return 403. (TODO)
func collab(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	conn := POOL.Get()
	defer conn.Close()
	// Doc instances, fix this
	DocInstances, err := redis.Strings(conn.Do("SMEMBERS", "doc_hashes"))
	if err != nil {
		log.Println(err)
	}
	// sort.Strings(DocInstances)
	// cookie, err := r.Cookie(COOKIE_NAME)
	if err != nil {
		log.Println(err)
	}
	template.Must(
		template.New("collab.html").ParseFiles(
			PROJ_ROOT+"/collab.html")).Execute(w, struct {
		CookieName      string
		ServerAddr      string
		Username        string
		CollabInstances []string
	}{COOKIE_NAME, SERVER_ADDRESS, "Jimmy", DocInstances})
}

func index(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	w.Write([]byte("You shouldn't be here..."))
}

// Unused at the moment
func templates(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers",
		"Accept, 0, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	w.Write([]byte("{\"doc\":{\"type\":\"doc\",\"content\":[{\"type\":\"paragraph\",\"content\":[{\"type\":\"text\",\"text\":\"asd\"}]}]},\"users\":1,\"version\":3104,\"comments\":[],\"commentVersion\":39}"))
}

// Unused at the moment
func loginCallback(w http.ResponseWriter, r *http.Request) {
	err := r.FormValue("error")
	if err != "" {
		log.Println(err)
	}
	user := &User{
		Created:          1495185151,
		Created_utc:      1495185151,
		Has_mail:         true,
		Id:               getRandomString(10),
		Is_mod:           true,
		Name:             "Timmy",
		Level:            10,
		Active:           true,
		Activation_token: getRandomString(200),
		Created_at:       "Fri May 19 09:13:53 UTC 2017",
	}
	if user.Name == "" {
		http.Redirect(w, r, SERVER_ADDRESS+"/", 302)
		return
	}
	clientIp := strings.Split(r.RemoteAddr, ":")[0]
	AuthorizedIps = append(AuthorizedIps, clientIp)
	user.IP = clientIp
	// store reddit auth data in the map, Username -> RedditAuth data
	users[user.Name] = *user
	expire := time.Now().AddDate(0, 0, 1)
	cookie := &http.Cookie{
		Expires: expire,
		MaxAge:  86400,
		Name:    COOKIE_NAME,
		Value:   user.Name,
		Path:    "/",
		Domain:  DOMAIN,
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, SERVER_ADDRESS+"/chat", 302)
}

// Set and Get a document instance
func docState(w http.ResponseWriter, r *http.Request) {
	var err error
	vars := mux.Vars(r)
	conn := POOL.Get()
	defer conn.Close()
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers",
		"Accept, 0, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "GET" { // get doc instance
		log.Println("Get request for: " + vars["docHash"])
		docJson, err := redis.Bytes(conn.Do("HGET", vars["docHash"], "json"))
		if err != nil {
			log.Println(err)
		}
		w.Write(docJson)
	}
	if r.Method == "PUT" { // set doc instance
		log.Println("Put request for: " + vars["docHash"])
		docJson, _ := ioutil.ReadAll(r.Body)
		_, err = conn.Do("HSET", vars["docHash"], "json", docJson)
		if err != nil {
			log.Println(err)
		}
		w.Write([]byte(docJson))
	}
	// docHash := vars["docHash"]
	// hashed := fmt.Sprintf("%x", sha1.Sum([]byte(full_query)))
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

func main() {
	MessageChannel = make(chan DBDocInstance, 256)
	// a goroutine for saving messages
	go saveDocInstances(&MessageChannel)
	//for keeping track of users in memory
	users = make(map[string]User)
	r := mux.NewRouter()
	hub := newHub()
	go hub.run()
	r.HandleFunc("/", index)
	r.HandleFunc("/login", login)
	r.HandleFunc("/collab", collab)
	r.HandleFunc("/register", register)
	r.HandleFunc("/templates", templates)
	r.HandleFunc("/history/{docHash}", docState)
	r.HandleFunc("/collab_socket/{docHash}",
		func(w http.ResponseWriter, r *http.Request) {
			collabSockets(hub, w, r)
		})
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir(PROJ_ROOT+"/icons"))))
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
