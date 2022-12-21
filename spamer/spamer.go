package spamer

import (
	"log"
	"net/http"
	"time"
)

func DoSpam() {

	for {
		time.Sleep(time.Second * 2)
		go spam()
		go spam()
		go spam()
	}

}

func spam() {
	_, err := http.Get("http://localhost:3030")
	if err != nil {
		log.Fatal(err)
	}
}
