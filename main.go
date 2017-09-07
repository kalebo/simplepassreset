package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/go-zoo/bone"
)

const (
	PORT string        = ":4555"
	TTL  time.Duration = time.Duration(time.Hour * 5)
)

type resetRequest struct {
	accountname string
	created     time.Time
}

func (r *resetRequest) expired() bool {
	return time.Now().Sub(r.created) > TTL
}

var resetRequestsMap = struct {
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

	time.AfterFunc(time.Minute*5, tidyRequestsMap)
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
	if val, ok := resetRequestsMap.m[token]; ok {
		// return the form
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		rw.Write([]byte(fmt.Sprintf(`
<html>
<head>
<title>Password Reset</title>
</head>
<body>
<form class="pure-form" method="POST">
<fieldset>
	<legend>Reset password for '%s'</legend>

	<input type="password" placeholder="Password" id="password" required>
	<input type="password" placeholder="Confirm Password" id="confirm_password" required>

	<button type="submit" class="pure-button pure-button-primary">Confirm</button>
	</br>
	</br>
	<small style="color:grey"> This form will expire on %s </small>
</fieldset>
</form>
<script>
	var password = document.getElementById("password")
	, confirm_password = document.getElementById("confirm_password");

	function validatePassword(){
	if(password.value != confirm_password.value) {
		confirm_password.setCustomValidity("Passwords Don't Match");
	} else {
		confirm_password.setCustomValidity('');
	}
	}

	password.onchange = validatePassword;
	confirm_password.onkeyup = validatePassword;
</script>
</body>
</html>`, val.accountname, time.Now().Add(TTL).Format("2006-01-02T15:04:05 MST"))))
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
		rw.Write([]byte(`
<html>
<head>
<title>Password Reset</title>
</head>
<body>
Password has been reset!
</body>
</html>
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
