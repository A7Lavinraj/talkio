package main

import (
	"log"
	"net/http"
)

func main() {
	server := http.FileServer(http.Dir("public"))

	http.Handle("/", http.StripPrefix("/", server))

	log.Println("Server started on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
