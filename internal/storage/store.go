package storage

import (
	"context"
	"errors"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/zapadapter"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
	"time"
)

var (
	ErrUserExists       = errors.New("user already exists")
	ErrChatExists       = errors.New("chat already exists")
	ErrChatBadUsers     = errors.New("bad users list")
	ErrMessageBadChat   = errors.New("bad chat id")
	ErrMessageBadAuthor = errors.New("bad author id")
)

type Store struct {
	logger *zap.SugaredLogger
	db     *pgxpool.Pool
}

// New sets provided zap.Logger via zapadapter to pgxpool.Pool and returns instance of Store struct
func New(logger *zap.SugaredLogger) (*Store, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	config.ConnConfig.Logger = zapadapter.NewLogger(logger.Desugar())

	pool, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	return &Store{
		logger: logger,
		db:     pool,
	}, err
}

// CreateUser tries to insert user with specified username.
// TODO change id type to int64 as bigserial is 8 bytes (https://www.postgresql.org/docs/current/datatype-numeric.html)
// TODO think of using ON CONFLICT: https://postgrespro.ru/docs/postgresql/9.6/sql-insert#sql-on-conflict
// TODO maybe https://github.com/jackc/pgtype provides better types for current package as it includes binary encoding
func (s *Store) CreateUser(ctx context.Context, username string) (int, error) {
	s.logger.Debugf("Creating user (%s)", username)

	var id int
	sql := "insert into users (username, created_at) values ($1, $2) returning id"
	err := s.db.QueryRow(ctx, sql, username, time.Now()).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return 0, ErrUserExists
			}
		}
		return 0, err
	}

	s.logger.Debugf("Created user (%s) with id %d", username, id)

	return id, nil
}

// CreateChat performs two-step transaction: 1. insert chat record; 2. bulk insert on "chat-users" table
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
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				return 0, ErrChatExists
			default:
				return 0, err
			}
		}
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
	_, err = tx.CopyFrom(ctx, pgx.Identifier{"chat-users"}, []string{"chat_id", "user_id"}, copyFromBulk(rows))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23503":
				return 0, ErrChatBadUsers
			default:
				return 0, err
			}
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return 0, err
	}

	return 0, nil
}

func (s *Store) CreateMessage(ctx context.Context, chat, author int64, text string) (int64, error) {
	s.logger.Debugf("Creating message from user (id: %d) in chat (id: %d)", author, chat)

	var id int64
	sql := "insert into messages (chat_id, author_id, text, created_at) values ($1, $2, $3, $4)"
	err := s.db.QueryRow(ctx, sql, chat, author, text, time.Now()).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23503":
				switch pgErr.ConstraintName {
				case "messages_chat_id_fkey":
					return 0, ErrMessageBadChat
				case "messages_author_id_fkey":
					return 0, ErrMessageBadAuthor
				default:
					return 0, err
				}
			}
		}
	}

	return id, nil
}
