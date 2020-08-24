package storage

import (
	mytesting "avito-trainee-assignment/internal/testing"
	"context"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"testing"
	"time"
)

func bootstrap(t *testing.T) *Store {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	s, err := NewStore(context.Background(), logger.Sugar())
	require.NoError(t, err)

	return s
}

func TestNewStore(t *testing.T) {
	t.Parallel()

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	_, err = NewStore(context.Background(), logger.Sugar())
	require.NoError(t, err)
}

func TestCreateUser(t *testing.T) {
	t.Parallel()

	s := bootstrap(t)

	_, err := s.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)
}

func TestCreateUserExists(t *testing.T) {
	t.Parallel()

	s := bootstrap(t)

	username := mytesting.RandString()
	_, err := s.CreateUser(context.Background(), username)
	require.NoError(t, err)
	_, err = s.CreateUser(context.Background(), username)
	require.Equal(t, ErrUserExists, err)
}

func TestCreateChat(t *testing.T) {
	t.Parallel()

	s := bootstrap(t)

	// number of users
	n := 3

	// generating usernames
	usernames := make([]string, 0, n)
	for i := 0; i < n; i++ {
		usernames = append(usernames, mytesting.RandString())
	}

	// creating users in database
	userIDs := make([]int64, 0, n)
	for _, username := range usernames {
		id, err := s.CreateUser(context.Background(), username)
		require.NoError(t, err)

		userIDs = append(userIDs, id)
	}

	_, err := s.CreateChat(context.Background(), mytesting.RandString(), userIDs)
	require.NoError(t, err)
}

func TestCreateChatExists(t *testing.T) {
	t.Parallel()

	s := bootstrap(t)

	// number of users
	n := 3

	// generating usernames
	usernames := make([]string, 0, n)
	for i := 0; i < n; i++ {
		usernames = append(usernames, mytesting.RandString())
	}

	// creating users in database
	userIDs := make([]int64, 0, n)
	for _, username := range usernames {
		id, err := s.CreateUser(context.Background(), username)
		require.NoError(t, err)

		userIDs = append(userIDs, id)
	}

	name := mytesting.RandString()
	_, err := s.CreateChat(context.Background(), name, userIDs)
	require.NoError(t, err)
	_, err = s.CreateChat(context.Background(), name, userIDs)
	require.Equal(t, ErrChatExists, err)
}

func TestCreateChatBadUsers(t *testing.T) {
	t.Parallel()

	s := bootstrap(t)

	_, err := s.CreateChat(context.Background(), mytesting.RandString(), []int64{0, 1})
	require.Equal(t, ErrChatBadUsers, err)
}

func TestCreateMessage(t *testing.T) {
	t.Parallel()

	s := bootstrap(t)

	userOneID, err := s.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)
	userTwoID, err := s.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)

	chatID, err := s.CreateChat(context.Background(), mytesting.RandString(), []int64{userOneID, userTwoID})
	require.NoError(t, err)

	_, err = s.CreateMessage(context.Background(), chatID, userOneID, mytesting.RandString())
	require.NoError(t, err)
}

func TestCreateMessageChatNotExist(t *testing.T) {
	t.Parallel()

	s := bootstrap(t)

	userID, err := s.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)

	_, err = s.CreateMessage(context.Background(), 0, userID, "Hi There!")
	require.Equal(t, ErrChatNotExist, err)
}

func TestCreateMessageUserNotExist(t *testing.T) {
	t.Parallel()

	s := bootstrap(t)

	userOneID, err := s.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)
	userTwoID, err := s.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)

	chatID, err := s.CreateChat(context.Background(), mytesting.RandString(), []int64{userOneID, userTwoID})
	require.NoError(t, err)

	_, err = s.CreateMessage(context.Background(), chatID, 0, "Hi There!")
	require.Equal(t, ErrUserNotExist, err)
}

func TestCreateMessageUserNotChatMember(t *testing.T) {
	t.Parallel()

	s := bootstrap(t)

	userOneID, err := s.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)
	userTwoID, err := s.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)
	userThreeID, err := s.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)

	chatID, err := s.CreateChat(context.Background(), mytesting.RandString(), []int64{userOneID, userTwoID})
	require.NoError(t, err)

	_, err = s.CreateMessage(context.Background(), chatID, userThreeID, "Hi There!")
	require.Equal(t, ErrUserNotChatMember, err)
}

// TODO test not only by IDs but the whole chat rows
func TestChatsByUserID(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	s := bootstrap(t)

	_, err := s.ChatsByUserID(context.Background(), 0)
	require.Equal(t, ErrUserNotExist, err)
}

// TODO test not only by IDs but the whole message rows
func TestMessagesByChatID(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	s := bootstrap(t)

	_, err := s.MessagesByChatID(context.Background(), 0)
	require.Equal(t, ErrChatNotExist, err)
}
