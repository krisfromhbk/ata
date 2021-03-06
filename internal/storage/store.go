package storage

// TODO maybe https://github.com/jackc/pgtype provides better types for current package as it includes binary encoding

import (
	"avito-trainee-assignment/internal/storage/zapadapter"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
	"time"
)

var (
	ErrUserExists        = errors.New("user already exists")
	ErrUserNotExist      = errors.New("user does not exist")
	ErrUserNotChatMember = errors.New("user is not chat member")
	ErrUserHasNoChats    = errors.New("user does not have chats")
	ErrChatExists        = errors.New("chat already exists")
	ErrChatBadUsers      = errors.New("bad users list")
	ErrChatNotExist      = errors.New("chat does not exist")
	ErrChatHasNoMessages = errors.New("chat does not have messages")
)

// Store defines fields used in db interaction processes
type Store struct {
	logger *zap.SugaredLogger
	db     *pgxpool.Pool
}

// NewStore constructs Store instance with configured logger and extends default pgxpool.Config with options.
// The underlying ConnConfig includes fields of pgconn.Config struct such as Host, Port, Database, etc.
// The comment before pgconn.ParseConfig has a list of environment variables that are parsed by default.
func NewStore(ctx context.Context, logger *zap.SugaredLogger, opts ...Option) (*Store, error) {
	if logger == nil {
		return nil, errors.New("no logger provided")
	}

	config, _ := pgxpool.ParseConfig("")
	for _, o := range opts {
		o.apply(config)
	}

	config.ConnConfig.Logger = zapadapter.NewLogger(logger.Desugar())

	pool, err := pgxpool.ConnectConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("cannot connect using config %+v: %w", config, err)
	}

	return &Store{
		logger: logger,
		db:     pool,
	}, nil
}

// Close closes all database connections in pool
func (s *Store) Close() {
	s.logger.Info("Closing store connections")
	s.db.Close()
}

// CreateUser creates user and returns its id.
func (s *Store) CreateUser(ctx context.Context, username string) (int64, error) {
	requestID, ok := zapadapter.IDFromContext(ctx)
	logger := s.logger
	if ok {
		logger = s.logger.With("request_id", requestID)
	}

	logger.Debugf("Creating user (%s)", username)

	// check if user exists to prevent error log during s.db.QueryRow insert call
	var i int8
	sql := "select 1 from users where username = $1"
	err := s.db.QueryRow(ctx, sql, username).Scan(&i)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return 0, err
		}
	}

	if i == 1 {
		return 0, ErrUserExists
	}

	var id int64
	sql = "insert into users (username, created_at) values ($1, $2) returning id"
	err = s.db.QueryRow(ctx, sql, username, time.Now()).Scan(&id)
	if err != nil {
		return 0, err
	}

	s.logger.Debugf("Created user (%s) with id %d", username, id)

	return id, nil
}

// CreateChat performs two-step transaction to create chat
// (1. insert chat record; 2. bulk insert on "chat-users" table) and returns its id
// TODO decide whether several chats with same users possible (different chat names)
func (s *Store) CreateChat(ctx context.Context, name string, users []int64) (int64, error) {
	s.logger.Debugf("Creating chat (%s) with users (%v)", name, users)

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	// error handling can be omitted for rollback according docs
	// see https://pkg.go.dev/github.com/jackc/pgx/v4?tab=doc#hdr-Transactions or any source comment on Rollback
	defer tx.Rollback(context.Background())

	// creating chat record
	var id int64
	sql := "insert into chats (name, created_at) values ($1, $2) returning id"
	err = tx.QueryRow(ctx, sql, name, time.Now()).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return 0, ErrChatExists
		}
		return 0, err
	}

	// preparing data for bulk insert
	var rows []chatRow
	for _, user := range users {
		rows = append(rows, chatRow{
			chatId: id,
			userId: user,
		})
	}

	// bulk insert
	_, err = tx.CopyFrom(ctx, pgx.Identifier{"chat_users"}, []string{"chat_id", "user_id"}, copyFromBulk(rows))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.ForeignKeyViolation {
			return 0, ErrChatBadUsers
		}
		return 0, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return 0, err
	}

	s.logger.Debugf("Created chat (%s) with id %d", name, id)

	return id, nil
}

// CreateMessage creates new message in database and returns its id
func (s *Store) CreateMessage(ctx context.Context, chat, author int64, text string) (int64, error) {
	s.logger.Debugf("Creating message from user (id: %d) in chat (id: %d)", author, chat)

	// check if chat exists
	var i int8
	sql := "select 1 from chats where id = $1"
	err := s.db.QueryRow(ctx, sql, chat).Scan(&i)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrChatNotExist
		}
		return 0, err
	}

	// check if user exists
	sql = "select 1 from users where id = $1"
	err = s.db.QueryRow(ctx, sql, author).Scan(&i)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrUserNotExist
		}
		return 0, err
	}

	var id int64
	sql = "insert into messages (chat_id, author_id, text, created_at) values ($1, $2, $3, $4) returning id"
	err = s.db.QueryRow(ctx, sql, chat, author, text, time.Now()).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.ForeignKeyViolation {
				return 0, ErrUserNotChatMember
			}
		}
		return 0, err
	}

	return id, nil
}

// ChatsByUserID returns a list of all chats with all fields, sorted by the time of the last message in the chat
//(from latest to oldest)
func (s *Store) ChatsByUserID(ctx context.Context, user int64) ([]Chat, error) {
	s.logger.Debugf("Retrieving chats for user (id: %d)", user)

	// check if user exists
	var i int8
	sql := "select 1 from users where id = $1"
	err := s.db.QueryRow(ctx, sql, user).Scan(&i)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotExist
		}
		return nil, err
	}

	// check if user has chats
	sql = "select 1 from chat_users where user_id = $1"
	err = s.db.QueryRow(ctx, sql, user).Scan(&i)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserHasNoChats
		}
		return nil, err
	}

	type retrievedChat struct {
		id        int64
		name      string
		users     pgtype.JSONBArray
		createdAt time.Time
	}

	// TODO update sql to retrieve chats without messages ordered by creation time
	sql = ` -- user chats ordered by last message
			with user_chats as (
				select chats.id, 
					   chats.name, 
					   chats.created_at, 
					   chat_users.user_id, 
					   min(age(clock_timestamp(), messages.created_at)) as time_since_message_creation
				  from chats
				  join chat_users 
					on chat_users.chat_id = chats.id
				  join messages
					on chats.id = messages.chat_id
				 group by chats.id, chats.name, chats.created_at, chat_users.user_id 
				having chat_users.user_id = $1 
				 order by time_since_message_creation
			), 
			
			users_per_chat as (
				select
					chat_id,
					array_agg(jsonb_build_object('id', users.id, 'username', trim(users.username), 'created_at', users.created_at)) as users
				from chat_users 
				join users 
				  on chat_users.user_id = users.id
			   group by chat_id
			   order by chat_id desc
			)
			
			select user_chats.id, 
				   trim(user_chats.name),
				   users_per_chat.users,
				   user_chats.created_at
			  from user_chats
			  join users_per_chat
				on user_chats.id = users_per_chat.chat_id`

	rows, err := s.db.Query(ctx, sql, user)
	if err != nil {
		return nil, err
	}

	var chats []Chat
	for rows.Next() {
		var c retrievedChat
		err = rows.Scan(&c.id, &c.name, &c.users, &c.createdAt)
		if err != nil {
			return nil, err
		}

		currentChat := Chat{
			ID:        c.id,
			Name:      c.name,
			Users:     make([]User, len(c.users.Elements)),
			CreatedAt: c.createdAt,
		}

		usersJSON := make([]string, len(c.users.Elements))
		err = c.users.AssignTo(&usersJSON)
		if err != nil {
			return nil, err
		}

		for i, v := range usersJSON {
			err = json.Unmarshal([]byte(v), &currentChat.Users[i])
			if err != nil {
				return nil, err
			}
		}

		chats = append(chats, currentChat)
	}

	if rows.Err() != nil {
		return nil, err
	}

	s.logger.Debugf("Retrieved %d chats", len(chats))

	return chats, nil
}

// MessagesByChatID returns list of all chat messages with all fields, sorted by message creation time
// (from earliest to latest)
func (s *Store) MessagesByChatID(ctx context.Context, chat int64) ([]Message, error) {
	s.logger.Debugf("Retrieving messages for chat (id: %d)", chat)

	// check if chat exists
	var i int8
	sql := "select 1 from chats where id = $1"
	err := s.db.QueryRow(ctx, sql, chat).Scan(&i)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrChatNotExist
		}
		return nil, err
	}

	// check if chat has messages
	sql = "select 1 from messages where chat_id = $1"
	err = s.db.QueryRow(ctx, sql, chat).Scan(&i)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrChatHasNoMessages
		}
		return nil, err
	}

	sql = `select messages.id, 
				  messages.chat_id, 
				  messages.author_id, 
				  messages.text, 
				  messages.created_at
			 from messages 
			where chat_id = $1 
			order by created_at asc`

	rows, err := s.db.Query(ctx, sql, chat)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		err = rows.Scan(&m.ID, &m.Chat, &m.Author, &m.Text, &m.CreatedAt)
		if err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	s.logger.Debugf("Retrieved %d messages", len(messages))

	return messages, nil
}
