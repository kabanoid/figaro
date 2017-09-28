package figaro

import (
	"time"
)

// User represents Slack user
type User struct {
	ID       string
	Name     string
	FullName string
	Email    string
}

// Message represents Slack message
type Message struct {
	UserID    string
	ChannelID string
	CreatedAt time.Time
	Text      string
	Type      string
	Name      string // For example for a new channel name
}

// Channel represents Slack channel
type Channel struct {
	ID       string
	Name     string
	Ok       bool
	Archived bool
	Messages []*Message
}

// ChannelPair Contains bad and good channels
type ChannelPair struct {
	Bad []*Channel
	Ok  []*Channel
}
