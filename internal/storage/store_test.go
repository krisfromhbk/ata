package storage

import (
	"context"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"math/rand"
	"strings"
	"testing"
	"time"
)

var testUsers = []int64{39, 41, 42}

func randString() string {
	rand.Seed(time.Now().Unix())

	var out strings.Builder
	charSet := "abcdedfghijklmnopqrstABCDEFGHIJKLMNOP"
	length := 10
	for i := 0; i < length; i++ {
		random := rand.Intn(len(charSet))
		randomChar := charSet[random]
		out.WriteString(string(randomChar))
	}
	return out.String()
}

func bootstrap(t *testing.T) *Store {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	s, err := New(logger.Sugar())
	require.NoError(t, err)

	return s
}

func TestCreateUser(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	s, err := New(logger.Sugar())
	require.NoError(t, err)

	_, err = s.CreateUser(context.Background(), randString())
	require.NoError(t, err)
}

func TestCreateUserExists(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	s, err := New(logger.Sugar())
	require.NoError(t, err)

	username := randString()
	_, err = s.CreateUser(context.Background(), username)
	require.NoError(t, err)
	_, err = s.CreateUser(context.Background(), username)
	require.Equal(t, ErrUserExists, err)
}

func TestCreateChat(t *testing.T) {
	s := bootstrap(t)

	_, err := s.CreateChat(context.Background(), randString(), testUsers)
	require.NoError(t, err)
}

func TestCreateChatExists(t *testing.T) {
	s := bootstrap(t)

	name := randString()
	_, err := s.CreateChat(context.Background(), name, testUsers)
	require.NoError(t, err)
	_, err = s.CreateChat(context.Background(), name, testUsers)
	require.Equal(t, ErrChatExists, err)
}

func TestCreateChatViolationFK(t *testing.T) {
	s := bootstrap(t)

	_, err := s.CreateChat(context.Background(), randString(), []int64{1, 2})
	require.Equal(t, ErrChatBadUsers, err)
}

func TestCreateMessage(t *testing.T) {
	s := bootstrap(t)

	_, err := s.CreateMessage(context.Background(), 4, 39, "Hi There!")
	require.NoError(t, err)
}

func TestCreateMessageBadChat(t *testing.T) {
	s := bootstrap(t)

	_, err := s.CreateMessage(context.Background(), 1, 39, "Hi There!")
	require.Equal(t, ErrMessageBadChat, err)
}

func TestCreateMessageBadAuthor(t *testing.T) {
	s := bootstrap(t)

	_, err := s.CreateMessage(context.Background(), 4, 1, "Hi There!")
	require.Equal(t, ErrMessageBadAuthor, err)
}
