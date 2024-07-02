package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/go-playground/validator"
)

type Comment struct {
	Message  string `validate:"required,min=1,max=140"`
	UserName string `validate:"required,min=1,max=15"`
}

func main() {
	var mutex = &sync.RWMutex{}
	comments := make([]Comment, 0, 100)

	http.HandleFunc("/comments", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case http.MethodGet:
			mutex.RLock() // 読み取り時に書き込みがあることを考慮してロック

			if err := json.NewEncoder(w).Encode(comments); err != nil {
				http.Error(w, fmt.Sprintf(`{"status":"%s"}`, err), http.StatusInternalServerError)
				return
			}
			mutex.RUnlock()

			err := r.ParseForm()
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			word := r.FormValue("searchWord")
			log.Printf("searchWord = %s\n", word)

			words, ok := r.Form["searchWord"]
			log.Printf("search words = %v has values %v\n", words, ok)

			log.Print("all queries")
			for key, values := range r.Form {
				log.Printf(" %s: %v\n", key, values)
			}
		case http.MethodPost:
			var c Comment
			if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
				http.Error(w, fmt.Sprintf(`{"status":"%s}`, err), http.StatusInternalServerError)
				return
			}
			validate := validator.New()
			if err := validate.Struct(c); err != nil {
				var out []string
				var ve validator.ValidationErrors
				if errors.As(err, &ve) {
					for _, fe := range ve {
						switch fe.Field() {
						case "Message":
							out = append(out, fmt.Sprintln("Messageは1~140文字です"))
						case "UserName":
							out = append(out, fmt.Sprintln("UserNameは1~15文字です"))
						}
					}
				}
				http.Error(w, fmt.Sprintf(`{"status":"%s"}`, strings.Join(out, ",")), http.StatusBadRequest)
				return
			}

			mutex.Lock()
			comments = append(comments, c)
			mutex.Unlock() // 同時に複数アクセスを防ぐためにロック

			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"status": "created"}`))
		default:
			http.Error(w, `{"status":"permits only GET or POST"}`, http.StatusMethodNotAllowed)
		}
	})
	http.ListenAndServe(":8080", nil)
}
