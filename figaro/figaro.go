package figaro

import (
	"bytes"
	"encoding/json"
	"log"
	"sort"
	"strings"
	"sync"
	"time"
)

// Figaro is a main component. It
// * Updates the storage with data from slack.
// * Exposes data from storage to clients via HTTP and WebSocket.
type Figaro struct {
	sl                   *Slack
	st                   *Storage
	pu                   *PushService
	channelPattern       string
	messageLimit         uint
	domains              []string
	lastChannelPairBytes []byte
}

// NewFigaro creates main component.
// It updates data from Slack to the storage. It returns error if it fails
// to update.
func NewFigaro(sl *Slack, st *Storage, pu *PushService, channelPattern string,
	messageLimit uint, domains []string) (*Figaro, error) {
	log.Println("Starting Figaro...")
	f := &Figaro{
		sl:             sl,
		st:             st,
		pu:             pu,
		channelPattern: channelPattern,
		messageLimit:   messageLimit,
		domains:        domains,
	}
	if err := f.updateStorage(); err != nil {
		log.Println("Cannot update Storage during startup:", err)
		return nil, err
	}
	go f.serve()
	log.Println("Figaro started.")
	return f, nil
}

func (f *Figaro) serve() {
	tickCh := time.Tick(time.Hour)
	for {
		select {
		case <-tickCh:
			if err := f.updateStorage(); err != nil {
				log.Println("Cannot update Storage during periodical update:", err)
			}
		case msg := <-f.sl.MessageCh():
			f.processMessages([]*Message{msg})
		}
		f.notifyUsers()
	}
}

func (f *Figaro) notifyUsers() {
	channels, err := f.st.GetChannelsByRegex(f.channelPattern, f.messageLimit)
	if err != nil {
		log.Println("Cannot notify users:", err)
	}
	ids := make([]string, 0, len(channels))
	for _, channel := range channels {
		ids = append(ids, channel.Messages[0].UserID)
	}
	users, err := f.st.GetUsers(ids)
	if err != nil {
		log.Println("Cannot notify users:", err)
	}
	idToEmail := make(map[string]string)
	for _, user := range users {
		idToEmail[user.ID] = user.Email
	}
	channelPair := ChannelPair{}
	for _, channel := range channels {
		id := channel.Messages[0].UserID
		email := idToEmail[id]
		if isInDomains(email, f.domains) {
			channelPair.Ok = append(channelPair.Ok, channel)
		} else {
			channelPair.Bad = append(channelPair.Bad, channel)
		}
	}
	sortChannelsByLastMessageTime(channelPair.Ok)
	sortChannelsByLastMessageTime(channelPair.Bad)
	channelPairBytes, err := json.Marshal(channelPair)
	if err != nil {
		log.Fatalln("Cannot marshal channel pair:", err)
	}
	if !bytes.Equal(channelPairBytes, f.lastChannelPairBytes) {
		f.lastChannelPairBytes = channelPairBytes
		f.pu.In() <- channelPairBytes
	}
}

func isInDomains(email string, domains []string) bool {
	for _, domain := range domains {
		if strings.HasSuffix(email, "@"+domain) {
			return true
		}
	}
	return false
}

func sortChannelsByLastMessageTime(channels []*Channel) {
	sort.Slice(channels, func(i, j int) bool {
		return channels[i].Messages[0].CreatedAt.UnixNano() <
			channels[j].Messages[0].CreatedAt.UnixNano()
	})
}

func (f *Figaro) updateStorage() error {
	log.Println("Updating storage...")
	if err := f.updateUsers(); err != nil {
		log.Println("Error occurred when update users:", err)
		return err
	}
	if err := f.updateChannels(); err != nil {
		log.Println("Error occurred when update channels:", err)
		return err
	}
	if err := f.updateMessages(); err != nil {
		log.Println("Error occurred when update messages:", err)
		return err
	}
	log.Println("Storage updated.")
	return nil
}

func (f *Figaro) updateUsers() error {
	log.Println("Update users...")
	users, err := f.sl.GetUsers()
	if err != nil {
		log.Println("Cannot get users from Slack:", err)
		return err
	}

	err = f.st.UpdateUsers(users)
	if err != nil {
		log.Println("Cannot update users in Storage:", err)
		return err
	}
	log.Println("Users updated")
	return nil
}

func (f *Figaro) updateChannels() error {
	log.Println("Updating channels...")
	channels, err := f.sl.GetChannels()
	if err != nil {
		log.Println("Cannot get channels from Slack:", err)
		return err
	}
	err = f.st.UpdateChannels(channels)
	if err != nil {
		log.Println("Cannot update channels in Storage:", err)
		return err
	}
	log.Println("Channels updated")
	return nil
}

func (f *Figaro) processMessages(messages []*Message) error {
	// Collect text messages here to update them all in once
	txtMessages := make([]*Message, 0, len(messages))
	for _, m := range messages {
		// See the full list of message subtypes here:
		// https://api.slack.com/events/message
		switch m.Type {
		case "":
			txtMessages = append(txtMessages, m)
		case "channel_archive":
			if err := f.st.UpdateChannelArch(m.ChannelID, true); err != nil {
				log.Println("Cannot archive channel:", err)
			}
			log.Println("Channel archived:", m.ChannelID)
		case "channel_unarchive":
			if err := f.st.UpdateChannelArch(m.ChannelID, false); err != nil {
				log.Println("Cannot unarchive channel:", err)
			}
			log.Println("Channel unarchived:", m.ChannelID)
		case "channel_name":
			if err := f.st.UpdateChannelName(m.ChannelID, m.Name); err != nil {
				log.Println("Cannot rename channel:", err)
			}
			log.Printf("Channel %s renamed to %s\n", m.ChannelID, m.Name)
		}
	}
	return f.st.UpdateMessages(txtMessages)
}

func (f *Figaro) updateMessages() error {
	log.Println("Updating messages...")
	channels, err := f.sl.GetChannels()
	if err != nil {
		log.Println("Cannot get channels from Slack:", err)
		return err
	}

	if len(channels) == 0 {
		log.Println("No channels found. No messages to store.")
		return nil
	}

	wg := sync.WaitGroup{}
	for _, channel := range channels {
		wg.Add(1)
		go func(channel *Channel) {
			defer wg.Done()
			ts, err := f.st.GetLastMessageTS(channel.ID)
			if err != nil {
				log.Println("Cannot get last message TS:", err)
				return
			}
			err = f.sl.GetMessages(channel.ID, ts, f.processMessages)
			if err != nil {
				log.Println("Cannot store message:", err)
				return
			}
		}(channel)
	}
	wg.Wait()
	log.Println("Messages updated")
	return nil
}

// Close releases all acquired resources.
func (f *Figaro) Close() error {
	return nil
}
