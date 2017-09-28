package figaro

// Schema
const queryCreateSchema = `--Creates Figaro's schema
CREATE SCHEMA IF NOT EXISTS figaro;

--Creates table for Slack users
CREATE TABLE IF NOT EXISTS figaro.users (
	user_id		VARCHAR,
	name 		VARCHAR,
	full_name	VARCHAR,
	email 		VARCHAR	
);
CREATE UNIQUE INDEX IF NOT EXISTS users_user_id_idx ON figaro.users (user_id);

--Creates table for Slack Message
CREATE TABLE IF NOT EXISTS figaro.messages (
	user_id			VARCHAR,
	channel_id		VARCHAR,
	created_at		TIMESTAMP,
	message_text	TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS messages_user_id_channel_id_created_at_idx 
	ON figaro.messages (user_id, channel_id, created_at);
CREATE INDEX IF NOT EXISTS messages_created_at_idx 
	ON figaro.messages (created_at);

--Creates table for Slack Channels 
CREATE TABLE IF NOT EXISTS figaro.channels (
	channel_id	VARCHAR,
	name		VARCHAR,
	ok			BOOLEAN,
	archived	BOOLEAN
);
CREATE UNIQUE INDEX IF NOT EXISTS channels_channel_id_idx
	ON figaro.channels (channel_id);
CREATE INDEX IF NOT EXISTS channels_name_idx
	ON figaro.channels (name);
`

// Queries
const queryUpdateUser = `--Creates user, if user exists, then update
INSERT INTO figaro.users VALUES ($1, $2, $3, $4) 
ON CONFLICT(user_id) DO UPDATE SET (name, full_name, email)=($2, $3, $4);
`

const queryGetUsers = `--Returns users by user IDs
SELECT * FROM figaro.users WHERE user_id IN $1;
`

const queryCountUsers = `--Counts users
SELECT COUNT(*) FROM figaro.users;
`

const queryUpdateMessage = `--Creates message, if message with the same user_id, 
--channel_id and careated_at exists, then update text
INSERT INTO figaro.messages VALUES ($1, $2, $3, $4)
ON CONFLICT(user_id, channel_id, created_at) DO UPDATE SET message_text = $4;
`

const queryGetMessagesByChannel = `--Returns limited amount of messages for a 
--channel sorted descendingly by created_at.
SELECT * FROM figaro.messages
WHERE figaro.messages.channel_id = $1
ORDER BY figaro.messages.created_at DESC LIMIT $2;
`

const queryGetLastMessageTS = `--Returns a timestamp of the channel's last message.
SELECT created_at FROM figaro.messages
WHERE channel_id = $1 ORDER BY created_at DESC LIMIT 1;
`

const queryCountMessages = `--Counts messages.
SELECT COUNT(*) FROM figaro.messages;
`

const queryUpdateChannel = `--Creates channel, if channel exists, then update.
INSERT INTO figaro.channels VALUES($1, $2, FALSE, $3)
ON CONFLICT(channel_id) DO UPDATE SET (name, archived) = ($2, $3);
`

const queryUpdateChannelStatus = `--Updates channel status.
UPDATE figaro.channels SET ok = $2 WHERE channel_id = $1;
`

const queryUpdateChannelArch = `--Archives or unarchives a channel.
UPDATE figaro.channels SET archived = $2 WHERE channel_id = $1;
`

const queryUpdateChannelName = `--Renames a channel.
UPDATE figaro.channels SET name = $2 WHERE channel_id = $1;
`

const queryGetChannel = `--Returns channel by its ID.
SELECT * FROM figaro.channels WHERE channel_id = $1
`

const queryGetChannels = `--Returns all.
SELECT * FROM figaro.channels;
`

const queryCountChannels = `--Counts channels
SELECT COUNT(*) FROM figaro.channels;
`
