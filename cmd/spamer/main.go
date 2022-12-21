package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func main() {

	for {
		time.Sleep(time.Second * 1)
		go spam()
		go spam()
		go spam()
	}

}

func spam() {
	resp, err := http.Get("http://localhost:3030")
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(string(body))
}
