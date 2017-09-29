package main

import (
	"log"
	"os"
	"strings"

	"github.com/adyatlov/figaro/figaro"
	"github.com/kelseyhightower/envconfig"
)

type configuration struct {
	Dbaddr      string `desc:"DB connection string" required:"true"`
	Wsaddr      string `desc:"web socket service address" default:"localhost:8080"`
	Slacktoken  string `desc:"slack token" required:"true"`
	Slackteam   string `desc:"slack team" required:"true"`
	Domains     string `desc:"comma-separated organization domains" required:"true"`
	Delay       uint   `desc:"delay between db updates in seconds" default:"30"`
	Nmessages   uint   `desc:"max number of last messages to show" default:"3"`
	Ncharacters uint   `desc:"max number of first characters to show for each message" default:"256"`
	Pattern     string `desc:"channel name regex pattern" default:".*"`
}

func main() {
	log.Println("Starting Figaro")
	var conf configuration
	if err := envconfig.Process("FIGARO", &conf); err != nil {
		log.Println(err.Error())
		envconfig.Usage("FIGARO", &conf)
		os.Exit(1)
	}
	domains := strings.Split(conf.Domains, ",")
	for i, domain := range domains {
		domains[i] = strings.TrimSpace(domain)
	}
	log.Println("Domains:", domains)
	st, err := figaro.NewStorage(conf.Dbaddr)
	if err != nil {
		log.Fatalln("Cannot create Storage service", err)
	}
	defer st.Close()
	sl := figaro.NewSlack(conf.Slacktoken)
	if err != nil {
		log.Fatalln("Cannot create Slack service", err)
	}
	pu := figaro.NewPushService()
	f, err := figaro.NewFigaro(sl, st, pu, conf.Pattern, conf.Nmessages, domains)
	if err != nil {
		log.Println("Cannot create Figaro service:", err)
	}
	defer f.Close()
	wait := make(chan struct{})
	<-wait
}
