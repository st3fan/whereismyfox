package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/sessions"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

var store = sessions.NewCookieStore([]byte(gServerConfig.SessionCookie))

type PersonaResponse struct {
	Status   string `json: "status"`
	Email    string `json: "email"`
	Audience string `json: "audience"`
	Expires  int64  `json: "expires"`
	Issuer   string `json: "issuer"`
	Reason   string `json: "reason,omitempty"`
}

func IsLoggedIn(r *http.Request) bool {
	session, _ := store.Get(r, "persona-session")
	email := session.Values["email"]
	return email != nil
}

func GetLoginName(r *http.Request) string {
	session, _ := store.Get(r, "persona-session")
	email := session.Values["email"]

	if str, ok := email.(string); ok {
		return str
	}
	return ""
}

func loginCheckHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "persona-session")
	email := session.Values["email"]
	if email != nil {
		fmt.Fprintf(w, email.(string))
	} else {
		fmt.Fprintf(w, "")
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "persona-session")
	session.Values["email"] = nil
	session.Save(r, w)
	w.Write([]byte("OK"))
}

func appLoginHandler(w http.ResponseWriter, r *http.Request) {
	doLogin("https://firefoxos.persona.org/verify", w, r)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	doLogin("https://verifier.login.persona.org/verify", w, r)
}

func doLogin(verifierURL string, w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte("Bad Request."))
	}

	assertion := r.FormValue("assertion")

	if assertion == "" {
		w.WriteHeader(400)
		w.Write([]byte("Bad Request."))
	}

	data := url.Values{"assertion": {assertion}, "audience": {"http://" + gServerConfig.PersonaName}}

	resp, err := http.PostForm(verifierURL, data)
	if err != nil {
		log.Println(err)
		w.WriteHeader(400)
		w.Write([]byte("Bad Request."))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		w.WriteHeader(400)
		w.Write([]byte("Bad Request."))
	}

	pr := &PersonaResponse{}
	err = json.Unmarshal(body, pr)
	if err != nil {
		log.Println(err)
		w.WriteHeader(400)
		w.Write([]byte("Bad Request."))
	}

	if pr.Status != "okay" {
		log.Println("Persona failed to verify: " + pr.Reason)
		w.WriteHeader(400)
		w.Write([]byte("Bad Request"))
	}

	session, _ := store.Get(r, "persona-session")
	session.Values["email"] = pr.Email
	session.Save(r, w)

	w.Write(body)
}
