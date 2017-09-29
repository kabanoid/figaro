package figaro

import (
	"database/sql"
	"log"
	"regexp"
	"time"

	"github.com/lib/pq" // _ is a common practice for the database/sql package
)

const maxDBConn = 10

// Storage is our DB backend
type Storage struct {
	db *sql.DB
}

// NewStorage creates a new storage with the URL connection like
// postgres://user:password@db.host.com:5439/figaro
func NewStorage(connURL string) (*Storage, error) {
	db, err := sql.Open("postgres", connURL)
	if err != nil {
		return nil, err
	}
	s := &Storage{}
	s.db = db
	s.db.SetMaxOpenConns(maxDBConn)
	log.Println("Storage: starting Storage...")
	if err := s.createSchema(); err != nil {
		return nil, err
	}
	log.Println("Storage: Storage started.")
	return s, nil
}

func (s *Storage) createSchema() error {
	log.Println("Storage: creating schema...")
	if _, err := s.db.Exec(queryCreateSchema); err != nil {
		return err
	}
	log.Println("Storage: schema created.")
	return nil
}

// Close closes db connections of the storage. Makes the storage unusable.
func (s *Storage) Close() error {
	err := s.db.Close()
	return err
}

// UpdateUser Saves user.
// If the user doesn't exist, then create a new one.
func (s *Storage) UpdateUser(user *User) error {
	_, err := s.db.Exec(queryUpdateUser,
		user.ID, user.Name, user.FullName, user.Email)
	return err
}

// UpdateUsers updates users in bulk.
// If the user doesn't exist, then create a new one.
func (s *Storage) UpdateUsers(users []*User) error {
	// I use a transaction here, because it works faster than db.Prepare()
	// prepared statement.
	txn, err := s.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := txn.Prepare(queryUpdateUser)
	if err != nil {
		txn.Rollback()
		return err
	}
	// Commit or Rollback should close it, but I'm not sure if pq library
	// obeys it.
	defer stmt.Close()

	for _, user := range users {
		_, err = stmt.Exec(user.ID, user.Name, user.FullName, user.Email)
		if err != nil {
			txn.Rollback()
			return err
		}
	}

	err = txn.Commit()
	if err != nil {
		log.Println("Cannot commit transaction:", err)
		return err
	}
	return nil
}

// GetUsers Gets users by IDs
func (s *Storage) GetUsers(ids []string) ([]*User, error) {
	rows, err := s.db.Query(queryGetUsers, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*User
	for rows.Next() {
		user := &User{}
		if err := rows.Scan(
			user.ID,
			user.Name,
			user.FullName,
			user.Email); err != nil {
			continue
		}
		users = append(users, user)
	}
	return users, nil
}

// CountUsers returns total amount of users in the storage
func (s *Storage) CountUsers() (n int64, err error) {
	row := s.db.QueryRow(queryCountUsers)
	err = row.Scan(&n)
	if err != nil {
		log.Println("Cannot count users:", err)
	}
	return
}

// UpdateMessage updates or creates a message.
// If message with the same user ID, channel Id and timestamp exists,
// then update the text, otherwise creates a new message.
func (s *Storage) UpdateMessage(message *Message) error {
	_, err := s.db.Exec(queryUpdateMessage,
		message.UserID, message.ChannelID, message.CreatedAt.UTC(), message.Text)
	return err
}

// UpdateMessages updates or creates messages in bulk.
// If message with the same user ID, channel Id and timestamp exists,
// then update the text, otherwise creates a new message.
func (s *Storage) UpdateMessages(messages []*Message) error {
	// I use a transaction here, because it works faster than db.Prepare()
	// prepared statement.
	txn, err := s.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := txn.Prepare(queryUpdateMessage)
	if err != nil {
		txn.Rollback()
		return err
	}
	// Commit or Rollback should close it, but I'm not sure if pq library
	// obeys it.
	defer stmt.Close()

	for _, m := range messages {
		_, err = stmt.Exec(m.UserID, m.ChannelID, m.CreatedAt.UTC(), m.Text)
		if err != nil {
			txn.Rollback()
			return err
		}
	}

	err = txn.Commit()
	if err != nil {
		log.Println("Cannot commit transaction:", err)
		return err
	}
	return nil
}

// GetMessagesByChannel returns limited amount of messages by channel
// sorted descendingly by creation time.
func (s *Storage) GetMessagesByChannel(channelID string,
	limit uint) ([]*Message, error) {
	rows, err := s.db.Query(queryGetMessagesByChannel, channelID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var messages []*Message
	for rows.Next() {
		message := &Message{}
		if err := rows.Scan(
			&message.UserID,
			&message.ChannelID,
			&message.CreatedAt,
			&message.Text); err != nil {
			continue
		}
		messages = append(messages, message)
	}
	return messages, nil
}

// CountMessages returns total amount of messages in the storage
func (s *Storage) CountMessages() (n int64, err error) {
	row := s.db.QueryRow(queryCountMessages)
	err = row.Scan(&n)
	if err != nil {
		log.Println("Cannot count messages:", err)
	}
	return
}

// UpdateChannel updates channel.
// If the channel doesn't exist, then creates a new one.
func (s *Storage) UpdateChannel(channel *Channel) error {
	_, err := s.db.Exec(queryUpdateChannel,
		channel.ID, channel.Name, channel.Archived)
	if err != nil {
		return err
	}
	return nil
}

// UpdateChannels updates channels in bulk.
// If the channel doesn't exist, then creates a new one.
func (s *Storage) UpdateChannels(channels []*Channel) error {
	// I use a transaction here, because it works faster than db.Prepare()
	// prepared statement.
	txn, err := s.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := txn.Prepare(queryUpdateChannel)
	if err != nil {
		txn.Rollback()
		return err
	}
	// Commit or Rollback should close it, but I'm not sure if pq library
	// obeys it.
	defer stmt.Close()

	for _, ch := range channels {
		if _, err := stmt.Exec(ch.ID, ch.Name, ch.Archived); err != nil {
			txn.Rollback()
			return err
		}
	}

	err = txn.Commit()
	if err != nil {
		log.Println("Cannot commit transaction:", err)
		return err
	}
	return nil
}

// UpdateChannelStatus updates a channel status.
func (s *Storage) UpdateChannelStatus(id string, ok bool) error {
	_, err := s.db.Exec(queryUpdateChannelStatus, id, ok)
	return err
}

// UpdateChannelArch archives or unarchives a channel.
func (s *Storage) UpdateChannelArch(id string, archived bool) error {
	_, err := s.db.Exec(queryUpdateChannelArch, id, archived)
	return err
}

// UpdateChannelName renames a channel.
func (s *Storage) UpdateChannelName(id string, name string) error {
	_, err := s.db.Exec(queryUpdateChannelName, id, name)
	return err
}

// GetChannel returns channel by its ID.
func (s *Storage) GetChannel(chID string) (channel *Channel, err error) {
	channel = &Channel{}
	row := s.db.QueryRow(queryGetChannel, chID)
	err = row.Scan(&channel.ID,
		&channel.Name, &channel.Ok, &channel.Archived)
	return
}

// GetChannelsByRegex returns channels which names match the given regex with
// the last lim messages. It doesn't return channels which don't have messages.
func (s *Storage) GetChannelsByRegex(pattern string, lim uint) ([]*Channel, error) {
	rows, err := s.db.Query(queryGetChannels, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var channels []*Channel
	for rows.Next() {
		channel := &Channel{}
		if err := rows.Scan(&channel.ID,
			&channel.Name, &channel.Ok, &channel.Archived); err != nil {
			return nil, err
		}
		if matched, err := regexp.MatchString(pattern, channel.Name); err != nil {
			if !matched {
				continue
			}
		} else {
			return nil, err
		}
		if channel.Messages, err = s.GetMessagesByChannel(channel.ID, lim); err != nil {
			return nil, err
		}
		if len(channel.Messages) == 0 {
			continue
		}
		channels = append(channels, channel)
	}
	return channels, nil
}

// GetLastMessageTS returns a timestamp of a last message in a channel and
// zero time if channel is empty.
func (s *Storage) GetLastMessageTS(chID string) (t time.Time, err error) {
	err = s.db.QueryRow(queryGetLastMessageTS, chID).Scan(&t)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	return
}

// CountChannels returns total amount of channels in the storage
func (s *Storage) CountChannels() (n int64, err error) {
	row := s.db.QueryRow(queryCountChannels)
	err = row.Scan(&n)
	if err != nil {
		log.Println("Cannot count messages:", err)
	}
	return
}
