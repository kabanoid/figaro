package figaro

import (
	"fmt"
	nlopesslack "github.com/nlopes/slack"
	"log"
	"strconv"
	"time"
)

// Slack fetches Users, Messages and Channels from Slack
type Slack struct {
	api       *nlopesslack.Client
	messageCh chan *Message
}

// NewSlack creates a new slack service
func NewSlack(token string) *Slack {
	s := &Slack{}
	s.api = nlopesslack.New(token)
	// s.api.SetDebug(true)
	s.messageCh = make(chan *Message)
	go s.serveRTM()
	return s
}

// MessageCh channel returns Slack RTM messages
func (s *Slack) MessageCh() <-chan *Message {
	return s.messageCh
}

func (s *Slack) serveRTM() {
	rtm := s.api.NewRTM()
	go rtm.ManageConnection()
	for rtmMsg := range rtm.IncomingEvents {
		fmt.Print("Event Received: ")
		switch ev := rtmMsg.Data.(type) {
		case *nlopesslack.HelloEvent:
			log.Println("Slack RTM says Hello")
		case *nlopesslack.MessageEvent:
			apiMsg := rtmMsg.Data.(*nlopesslack.MessageEvent).Msg
			msg := &Message{}
			msg.UserID = apiMsg.User
			msg.ChannelID = apiMsg.Channel
			msg.CreatedAt = strToTime(apiMsg.Timestamp)
			msg.Text = apiMsg.Text
			msg.Type = apiMsg.SubType
			msg.Name = apiMsg.Name
			s.messageCh <- msg
		case *nlopesslack.RTMError:
			log.Printf("Slack RTM Error: %s\n", ev.Error())
		case *nlopesslack.InvalidAuthEvent:
			log.Println("Slack RTM Error: Invalid auth")
		}
	}
}

// GetUsers returns all slack users
func (s *Slack) GetUsers() ([]*User, error) {
	apiUsers, err := s.api.GetUsers()
	if err != nil {
		return nil, err
	}
	users := make([]*User, 0, len(apiUsers))
	for _, apiUser := range apiUsers {
		user := &User{}
		user.ID = apiUser.ID
		user.Name = apiUser.Name
		user.FullName = apiUser.RealName
		user.Email = apiUser.Profile.Email
		users = append(users, user)
	}
	return users, nil
}

// ProcMsgs is a type which describes a funciton which process messages
// a portion of messages received from GetMessages.
type ProcMsgs func(messages []*Message) error

// GetMessages gets all messages starting from specified timestamp
// (not including)
func (s *Slack) GetMessages(chID string, ts time.Time, process ProcMsgs) error {
	if ts.IsZero() {
		ts = time.Unix(1, 0)
	}
	query := nlopesslack.HistoryParameters{
		Latest: timeToStr(time.Now()),
		Oldest: timeToStr(ts),
		Count:  1000,
	}
	for {
		history, err := s.api.GetChannelHistory(chID, query)
		if err != nil {
			log.Println("Cannot get messages from Slack API:", err)
			return err
		}
		messages := make([]*Message, 0, len(history.Messages))
		for _, apiMsg := range history.Messages {
			if apiMsg.Type != "message" {
				continue
			}
			msg := &Message{}
			msg.UserID = apiMsg.User
			msg.ChannelID = chID
			msg.CreatedAt = strToTime(apiMsg.Timestamp)
			msg.Text = apiMsg.Text
			msg.Type = apiMsg.SubType
			msg.Name = apiMsg.Name
			messages = append(messages, msg)
		}
		if err := process(messages); err != nil {
			log.Println("Cannot process messages:", err)
			return err
		}
		if !history.HasMore {
			break
		}
		query.Oldest = history.Latest
	}
	log.Printf("Channel %v is processed.\n", chID)
	return nil
}

// GetChannels returns all slack channels without messages
func (s *Slack) GetChannels() ([]*Channel, error) {
	apiChannels, err := s.api.GetChannels(false)
	if err != nil {
		return nil, err
	}
	channels := make([]*Channel, 0, len(apiChannels))
	for _, apiCh := range apiChannels {
		channel := &Channel{}
		channel.ID = apiCh.ID
		channel.Name = apiCh.Name
		channel.Archived = apiCh.IsArchived
		channels = append(channels, channel)
	}
	return channels, nil
}

func timeToStr(t time.Time) string {
	return fmt.Sprintf("%010d.%06d", t.Unix(), t.Nanosecond()/1e3)
}

func strToTime(s string) time.Time {
	seconds, _ := strconv.ParseInt(s[0:10], 10, 64)
	microseconds, _ := strconv.ParseInt(s[11:17], 10, 64)
	return time.Unix(seconds, microseconds*1e3)
}
