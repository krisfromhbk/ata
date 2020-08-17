package storage

// TODO rewrite some tests to generate data they need during test instead of using predefined

import (
	mytesting "avito-trainee-assignment/internal/testing"
	"context"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"math/rand"
	"testing"
	"time"
)

var testUsers = []int64{39, 41, 42}

func bootstrap(t *testing.T) *Store {
	rand.Seed(time.Now().Unix())

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	s, err := NewStore(logger.Sugar(), TestConfig)
	require.NoError(t, err)

	return s
}

func TestNewStore(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	_, err = NewStore(logger.Sugar(), TestConfig)
	require.NoError(t, err)
}

func TestCreateUser(t *testing.T) {
	s := bootstrap(t)

	_, err := s.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)
}

func TestCreateUserExists(t *testing.T) {
	s := bootstrap(t)

	username := mytesting.RandString()
	_, err := s.CreateUser(context.Background(), username)
	require.NoError(t, err)
	_, err = s.CreateUser(context.Background(), username)
	require.Equal(t, ErrUserExists, err)
}

func TestCreateChat(t *testing.T) {
	s := bootstrap(t)

	_, err := s.CreateChat(context.Background(), mytesting.RandString(), testUsers)
	require.NoError(t, err)
}

func TestCreateChatExists(t *testing.T) {
	s := bootstrap(t)

	name := mytesting.RandString()
	_, err := s.CreateChat(context.Background(), name, testUsers)
	require.NoError(t, err)
	_, err = s.CreateChat(context.Background(), name, testUsers)
	require.Equal(t, ErrChatExists, err)
}

func TestCreateChatBadUsers(t *testing.T) {
	s := bootstrap(t)

	_, err := s.CreateChat(context.Background(), mytesting.RandString(), []int64{1, 2})
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

// TODO test not only by IDs but the whole chat rows
func TestChatsByUserID(t *testing.T) {
	s := bootstrap(t)
	// number of users
	n := 5

	// generating n users in database
	// test will retrieve chats for the first user
	userIDs := make([]int64, n)
	for i := range userIDs {
		id, err := s.CreateUser(context.Background(), mytesting.RandString())
		require.NoError(t, err)
		userIDs[i] = id
	}

	// creating chats between users [0,1], [0,2], [0,3], etc.
	chatIDs := make([]int64, n-1)
	for i, v := range mytesting.BatchUserIDs(userIDs) {
		id, err := s.CreateChat(context.Background(), mytesting.RandString(), v)
		require.NoError(t, err)
		chatIDs[i] = id
	}

	// creating 2 messages (author - first user) in each chat with 1 sec delay
	for _, v := range chatIDs {
		_, err := s.CreateMessage(context.Background(), v, userIDs[0], mytesting.RandString())
		require.NoError(t, err)
		time.Sleep(1 * time.Second)
		_, err = s.CreateMessage(context.Background(), v, userIDs[0], mytesting.RandString())
		require.NoError(t, err)
	}

	// retrieving chats by first userID
	chats, err := s.ChatsByUserID(context.Background(), userIDs[0])
	require.NoError(t, err)

	expected := mytesting.ReverseIDs(chatIDs)

	// extracting actual IDs
	actual := make([]int64, 0, len(chats))
	for _, v := range chats {
		actual = append(actual, v.ID)
	}

	require.Equal(t, expected, actual)
}

func TestChatsByUserIDNotExist(t *testing.T) {
	s := bootstrap(t)

	_, err := s.ChatsByUserID(context.Background(), 0)
	require.Equal(t, ErrUserNotExist, err)
}

// TODO test not only by IDs but the whole message rows
func TestMessagesByChatID(t *testing.T) {
	s := bootstrap(t)
	// number of messages
	n := 5

	userOneID, err := s.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)
	userTwoID, err := s.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)
	chatID, err := s.CreateChat(context.Background(), mytesting.RandString(), []int64{userOneID, userTwoID})
	require.NoError(t, err)

	messageIDs := make([]int64, n)
	for i := 0; i < n; i++ {
		id, err := s.CreateMessage(context.Background(), chatID, userTwoID, mytesting.RandString())
		require.NoError(t, err)
		messageIDs[i] = id
		require.NoError(t, err)
	}

	expected := messageIDs

	messages, err := s.MessagesByChatID(context.Background(), chatID)
	require.NoError(t, err)

	actual := make([]int64, 0, len(messages))
	for _, v := range messages {
		actual = append(actual, v.ID)
	}

	require.Equal(t, expected, actual)
}

func TestMessagesByChatIDNotExist(t *testing.T) {
	s := bootstrap(t)

	_, err := s.MessagesByChatID(context.Background(), 0)
	require.Equal(t, ErrChatNotExist, err)
}
