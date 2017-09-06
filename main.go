package main

import (
	"net/http"
	"sync"
	"math/rand"
	"time"
	"fmt"

	"github.com/go-zoo/bone"
)

var PORT string = ":4555"

type resetRequest struct {
	samaccountname string
	created time.Time 
}

func (r *resetRequest) expired() bool {
	return time.Now().Sub(r.created) > time.Duration(time.Hour * 5)
}

var resetRequestsMap = struct{
	sync.RWMutex
	m map[string]resetRequest // i.e, base64 secret maps to username
}{m: make(map[string]resetRequest)}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func main() {
	mux := bone.New()
	mux.Get("/:token", http.HandlerFunc(resetForm))
	mux.Post("/:token", http.HandlerFunc(resetPassword))

	go http.ListenAndServe(":4555", mux)
	go tidyRequestsMap()
	cliInterface()

}

func tidyRequestsMap() {
	resetRequestsMap.Lock() // Guards are wider than I like so I need to check 
	//that golang iteration can handle the map changing while iterating
	for k, v := range resetRequestsMap.m {
		if v.expired() {
			delete(resetRequestsMap.m, k)
		}
	}
	resetRequestsMap.Unlock()

	time.AfterFunc(time.Minute * 5, tidyRequestsMap)
}

func cliInterface() {
	for true {
		fmt.Print("Generate reset link for: ")
		var input string 
		fmt.Scanln(&input)
		token := RandString(64)

		resetRequestsMap.Lock()
		resetRequestsMap.m[token] = resetRequest{input, time.Now()}
		resetRequestsMap.Unlock()

		fmt.Printf("Generated link for `%s` at http://localhost%s/%s\n", input, PORT, token)

	}
}

func resetForm(rw http.ResponseWriter, r *http.Request) {
	token := bone.GetValue(r, "token")

	resetRequestsMap.Lock()
	defer resetRequestsMap.Unlock()
	if _, ok := resetRequestsMap.m[token]; ok {
		// return the form 
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	rw.Write([]byte(`<html>
	<head>
	<title></title>
	</head>
	<body>
	<form>
		Password:<input type="text" name="password">
		<input type="submit" value="Login">
	</form>
	</body>
	</html>`))
	return


	}
	
	http.NotFound(rw, r)

}

func resetPassword(rw http.ResponseWriter, r *http.Request) {
	token := bone.GetValue(r, "token")
	//newpassword := r.FormValue("password")

	resetRequestsMap.Lock()
	defer resetRequestsMap.Unlock()
	if _, ok := resetRequestsMap.m[token]; ok {
		// reset the password by calling the api async
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		rw.Write([]byte(`<html>
		`))

		delete(resetRequestsMap.m, token)
		return
	}

	http.NotFound(rw, r)
}

// from @icza's answer on SO
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandString(n int) string {
    b := make([]byte, n)
    for i := range b {
        b[i] = letterBytes[rand.Intn(len(letterBytes))]
    }
    return string(b)
}