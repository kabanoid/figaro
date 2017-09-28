package main

import (
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/adyatlov/backend/figaro"
	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var delay = int(*flag.Uint("delay", 3, "delay between messages"))
var nOkChannels = int(*flag.Uint("ok_channels", 25, "number of OK channels"))
var nBadChannels = int(*flag.Uint("bad_channels", 25, "number of bad channels"))
var nMessages = int(*flag.Uint("messages", 3, "number of channels"))
var maxTextLength = int(*flag.Uint("maxtext", 256, "max text length"))
var upgrader = websocket.Upgrader{}

func generateUser() figaro.User {
	return figaro.User{
		ID:       randomdata.PostalCode("SE"),
		Name:     "@" + strings.ToLower(randomdata.SillyName()),
		FullName: randomdata.FullName(randomdata.RandomGender),
		Email:    randomdata.Email(),
	}
}

func generateMessage() figaro.Message {
	createdAt, _ := time.Parse("Monday 2 Jan 2006", randomdata.FullDate())
	return figaro.Message{
		User:      generateUser(),
		CreatedAt: createdAt,
		Text:      generateText(),
	}
}

func generateChannel(ok bool) figaro.Channel {
	channel := figaro.Channel{
		ID: randomdata.PostalCode("SE"),
		Name: "#" +
			strings.ToLower(randomdata.Country(randomdata.FullCountry)),
		URL:      "http://some-long-url.com/",
		Ok:       ok,
		Messages: make([]figaro.Message, int(nMessages)),
	}
	for i := 0; i < nMessages; i++ {
		channel.Messages[i] = generateMessage()
	}
	return channel
}

func generateText() string {
	messageLength := 1 + rand.Intn(maxTextLength)
	text := ""
	for len(text) < messageLength {
		text += randomdata.Paragraph()
	}
	return text[:messageLength]
}

type channelPair struct {
	Bad []figaro.Channel
	Ok  []figaro.Channel
}

func serve(w http.ResponseWriter, r *http.Request) {
	allowAllOrigin := func(r *http.Request) bool { return true }
	upgrader.CheckOrigin = allowAllOrigin
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		var pair channelPair
		pair.Bad = make([]figaro.Channel, nBadChannels)
		for i := 0; i < nBadChannels; i++ {
			pair.Bad[i] = generateChannel(false)
		}
		pair.Ok = make([]figaro.Channel, nOkChannels)
		for i := 0; i < nOkChannels; i++ {
			pair.Ok[i] = generateChannel(true)
		}
		buff, _ := json.Marshal(pair)
		err = c.WriteMessage(websocket.TextMessage, buff)
		if err != nil {
			log.Println("write:", err)
			break
		}
		time.Sleep(time.Duration(delay) * time.Second)
	}
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/", serve)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
