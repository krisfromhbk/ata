package server

import (
	"avito-trainee-assignment/internal/storage"
	mytesting "avito-trainee-assignment/internal/testing"
	"bytes"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fastjson"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func bootstrapHandler(t *testing.T) *handler {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	store, err := storage.NewStore(logger.Sugar(), storage.TestConfig)
	require.NoError(t, err)

	h := &handler{
		logger: logger.Sugar(),
		store:  store,
		parsers: parsers{
			createChatPool:       fastjson.ParserPool{},
			createMessagePool:    fastjson.ParserPool{},
			chatsByUserIDPool:    fastjson.ParserPool{},
			messagesByChatIDPool: fastjson.ParserPool{},
		},
	}

	return h
}

func statusOkHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestEnforcePOSTJSON(t *testing.T) {
	t.Parallel()

	payload := bytes.NewBuffer([]byte(`{"username":"` + mytesting.RandString() + `"}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(statusOkHandler))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestEnforcePOSTJSON_NotPOST(t *testing.T) {
	t.Parallel()

	payload := bytes.NewBuffer([]byte(`{"username":"` + mytesting.RandString() + `"}`))
	req, err := http.NewRequest("GET", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(statusOkHandler))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusMethodNotAllowed, rr.Code)
	require.Equal(t, http.StatusText(http.StatusMethodNotAllowed)+"\n", rr.Body.String())
}

func TestEnforcePOSTJSON_MalformedContentType(t *testing.T) {
	t.Parallel()

	payload := bytes.NewBuffer([]byte(`{"username":"` + mytesting.RandString() + `"}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "1:2\n+/-")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(statusOkHandler))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Malformed Content-Type header\n", rr.Body.String())
}

func TestEnforcePOSTJSON_UnsupportedContentType(t *testing.T) {
	t.Parallel()

	payload := bytes.NewBuffer([]byte(`{"username":"` + mytesting.RandString() + `"}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "text/plain")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(statusOkHandler))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnsupportedMediaType, rr.Code)
	require.Equal(t, "Content-Type header must be application/json\n", rr.Body.String())
}

func TestEnforcePOSTJSON_BlankContentType(t *testing.T) {
	t.Parallel()

	payload := bytes.NewBuffer([]byte(`{"username":"` + mytesting.RandString() + `"}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(statusOkHandler))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestEnforcePOSTJSON_NoContentType(t *testing.T) {
	t.Parallel()

	payload := bytes.NewBuffer([]byte(`{"username":"` + mytesting.RandString() + `"}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	//req.Header.Set("Content-Type", "")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(statusOkHandler))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestEnforcePOSTJSON_MalformedJSON(t *testing.T) {
	t.Parallel()

	// missing opening quotation mark after colon
	payload := bytes.NewBuffer([]byte(`{"username":` + mytesting.RandString() + `"}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(statusOkHandler))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Malformed JSON\n", rr.Body.String())
}

func TestCreateUser(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"username":"` + mytesting.RandString() + `"}`))
	req, err := http.NewRequest("POST", "/users/add", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.createUser)

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	body, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err)

	// validating response JSON
	var p fastjson.Parser
	v, err := p.ParseBytes(body)
	require.NoError(t, err)
	idValue := v.Get("id")
	_, err = idValue.Int64()
	require.NoError(t, err)
}

func TestCreateUserNoUsernameField(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"alice":"bob"}`))
	req, err := http.NewRequest("POST", "/users/add", payload)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.createUser)

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Missing Field \"username\"\n", rr.Body.String())
}

func TestCreateUserBlankUsername(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"username":""}`))
	req, err := http.NewRequest("POST", "/users/add", payload)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.createUser)

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Field \"username\" must be a string and have non-zero length\n", rr.Body.String())
}

func TestCreateUserNullUsername(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"username":null}`))
	req, err := http.NewRequest("POST", "/users/add", payload)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.createUser)

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Field \"username\" must be a string and have non-zero length\n", rr.Body.String())
}

func TestCreateUserAlreadyExists(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	username := mytesting.RandString()
	_, err := h.store.CreateUser(context.Background(), username)
	require.NoError(t, err)

	payload := bytes.NewBuffer([]byte(`{"username":"` + username + `"}`))
	req, err := http.NewRequest("POST", "/users/add", payload)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.createUser)

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "User already exists\n", rr.Body.String())
}

func TestCreateUserInternalOnCreateUserCall(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"username":"` + mytesting.RandString() + `"}`))
	req, err := http.NewRequest("POST", "/users/add", payload)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.createUser)

	h.store.Close()

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestCreateChat(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

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
		id, err := h.store.CreateUser(context.Background(), username)
		require.NoError(t, err)

		userIDs = append(userIDs, id)
	}

	// test chat creation handler
	type request struct {
		Name  string  `json:"name"`
		Users []int64 `json:"users"`
	}

	payload := request{
		Name:  mytesting.RandString(),
		Users: userIDs,
	}

	encodedPayload, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/chats/add", bytes.NewBuffer(encodedPayload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.createChat)

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	body, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err)

	// validating response JSON
	var p fastjson.Parser
	v, err := p.ParseBytes(body)
	require.NoError(t, err)
	idValue := v.Get("id")
	_, err = idValue.Int64()
	require.NoError(t, err)
}

func TestCreateChatMalformedJSON(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	// missing opening quotation mark after colon
	payload := bytes.NewBuffer([]byte(`{"variable":` + mytesting.RandString() + `"}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createChat))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Malformed JSON\n", rr.Body.String())
}

func TestCreateChatNoNameField(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"users":[1,2]}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createChat))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Missing Field \"name\"\n", rr.Body.String())
}

func TestCreateChatNameFieldNotString(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"name":1,"users":[1,2]}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createChat))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Field \"name\" must be a string\n", rr.Body.String())
}

func TestCreateChatBlankName(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"name":"","users":[1,2]}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createChat))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Field \"name\" must have non-zero length\n", rr.Body.String())
}

func TestCreateChatNoUsersField(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"name":"` + mytesting.RandString() + `"}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createChat))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Missing Field \"users\"\n", rr.Body.String())
}

func TestCreateChatUsersFieldNotArray(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"name":"` + mytesting.RandString() + `","users":"1,2,3"}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createChat))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Field \"users\" must be an array\n", rr.Body.String())
}

func TestCreateChatUsersFieldNotEachInteger(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"name":"` + mytesting.RandString() + `","users":[1,"2",3]}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createChat))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Each item in \"users\" array field must be a 64-bit integer value\n", rr.Body.String())
}

func TestCreateChatUsersFieldInvalidUserID(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"name":"` + mytesting.RandString() + `","users":[1,-2,3]}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createChat))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Each integer in \"users\" array must be a valid user id grater than zero\n", rr.Body.String())
}

func TestCreateChatAlreadyExists(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	userOneID, err := h.store.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)
	userTwoID, err := h.store.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)
	chatName := mytesting.RandString()
	_, err = h.store.CreateChat(context.Background(), chatName, []int64{userOneID, userTwoID})
	require.NoError(t, err)

	// test chat creation handler
	type request struct {
		Name  string  `json:"name"`
		Users []int64 `json:"users"`
	}

	payload := request{
		Name:  chatName,
		Users: []int64{userOneID, userTwoID},
	}

	encodedPayload, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/", bytes.NewBuffer(encodedPayload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createChat))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Chat already exists\n", rr.Body.String())
}

func TestCreateChatInternalOnCreateChatCall(t *testing.T) {
	h := bootstrapHandler(t)

	userOneID, err := h.store.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)
	userTwoID, err := h.store.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)

	// test chat creation handler
	type request struct {
		Name  string  `json:"name"`
		Users []int64 `json:"users"`
	}

	payload := request{
		Name:  mytesting.RandString(),
		Users: []int64{userOneID, userTwoID},
	}

	encodedPayload, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/", bytes.NewBuffer(encodedPayload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createChat))

	h.store.Close()

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestCreateMessage(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	userOneID, err := h.store.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)
	userTwoID, err := h.store.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)

	chatID, err := h.store.CreateChat(context.Background(), mytesting.RandString(), []int64{userOneID, userTwoID})
	require.NoError(t, err)

	// test message creation handler
	type request struct {
		Chat   int64  `json:"chat"`
		Author int64  `json:"author"`
		Text   string `json:"text"`
	}

	payload := request{
		Chat:   chatID,
		Author: userOneID,
		Text:   mytesting.RandString(),
	}

	encodedPayload, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/messages/add", bytes.NewBuffer(encodedPayload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.createMessage)

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	body, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err)

	// validating response JSON
	var p fastjson.Parser
	v, err := p.ParseBytes(body)
	require.NoError(t, err)
	idValue := v.Get("id")
	_, err = idValue.Int64()
	require.NoError(t, err)
}

func TestCreateMessageMalformedJSON(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	// missing opening quotation mark after colon
	payload := bytes.NewBuffer([]byte(`{"variable":` + mytesting.RandString() + `"}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createMessage))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Malformed JSON\n", rr.Body.String())
}

func TestCreateMessageNoChatField(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"author":1,"text":"` + mytesting.RandString() + `"}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createMessage))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Missing Field \"chat\"\n", rr.Body.String())
}

func TestCreateMessageChatFieldInvalidID(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"chat":-1,"author":1,"text":"` + mytesting.RandString() + `"}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createMessage))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Field \"chat\" must be a valid chat id grater than zero\n", rr.Body.String())
}

func TestCreateMessageChatFieldNotInteger(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"chat":"1","author":1,"text":"` + mytesting.RandString() + `"}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createMessage))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Field \"chat\" must be a 64-bit integer value\n", rr.Body.String())
}

func TestCreateMessageNoAuthorField(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"chat":1,"text":"` + mytesting.RandString() + `"}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createMessage))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Missing Field \"author\"\n", rr.Body.String())
}

func TestCreateMessageAuthorFieldNotInteger(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"chat":1,"author":"2","text":"` + mytesting.RandString() + `"}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createMessage))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Field \"author\" must be a 64-bit integer value\n", rr.Body.String())
}

func TestCreateMessageAuthorFieldInvalidUserID(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"chat":1,"author":-1,"text":"` + mytesting.RandString() + `"}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createMessage))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Field \"author\" must be a valid user id grater than zero\n", rr.Body.String())
}

func TestCreateMessageNoTextField(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"chat":1,"author":2}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createMessage))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Missing Field \"text\"\n", rr.Body.String())
}

func TestCreateMessageTextFieldNotString(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"chat":1,"author":2,"text":["n","i","c","e"]}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createMessage))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Field \"text\" must be a string\n", rr.Body.String())
}

func TestCreateMessageBlankTextField(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"chat":1,"author":2,"text":""}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createMessage))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Field \"text\" must have non-zero length\n", rr.Body.String())
}

func TestCreateMessageChatNotExist(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	userID, err := h.store.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)

	// let's assume that test database will never has such sequence number in bigserial
	payload := bytes.NewBuffer([]byte(`{"chat":9223372036854775807,"author":` + strconv.FormatInt(userID, 10) + `,"text":"Hi!"}`))
	req, err := http.NewRequest("POST", "/", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createMessage))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Chat with provided id does not exist\n", rr.Body.String())
}

func TestCreateMessageAuthorNotExist(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	userOneID, err := h.store.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)
	userTwoID, err := h.store.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)

	chatID, err := h.store.CreateChat(context.Background(), mytesting.RandString(), []int64{userOneID, userTwoID})
	require.NoError(t, err)

	type request struct {
		Chat   int64  `json:"chat"`
		Author int64  `json:"author"`
		Text   string `json:"text"`
	}

	// let's assume that test database will never has such sequence number in bigserial
	payload := request{
		Chat:   chatID,
		Author: 9223372036854775807,
		Text:   "Hi!",
	}

	encodedPayload, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/", bytes.NewBuffer(encodedPayload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createMessage))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Author with provided id does not exist\n", rr.Body.String())
}

func TestCreateMessageAuthorNotChatMember(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	userOneID, err := h.store.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)
	userTwoID, err := h.store.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)
	userThreeID, err := h.store.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)

	chatID, err := h.store.CreateChat(context.Background(), mytesting.RandString(), []int64{userOneID, userTwoID})
	require.NoError(t, err)

	type request struct {
		Chat   int64  `json:"chat"`
		Author int64  `json:"author"`
		Text   string `json:"text"`
	}

	// let's assume that test database will never has such sequence number in bigserial
	payload := request{
		Chat:   chatID,
		Author: userThreeID,
		Text:   "Hi!",
	}

	encodedPayload, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/", bytes.NewBuffer(encodedPayload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createMessage))

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Author is not chat member\n", rr.Body.String())
}

func TestCreateMessageInternalOnCreateMessageCall(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	userOneID, err := h.store.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)
	userTwoID, err := h.store.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)

	chatID, err := h.store.CreateChat(context.Background(), mytesting.RandString(), []int64{userOneID, userTwoID})
	require.NoError(t, err)

	type request struct {
		Chat   int64  `json:"chat"`
		Author int64  `json:"author"`
		Text   string `json:"text"`
	}

	payload := request{
		Chat:   chatID,
		Author: userOneID,
		Text:   "Hi!",
	}

	encodedPayload, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/", bytes.NewBuffer(encodedPayload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := enforcePOSTJSON(http.HandlerFunc(h.createMessage))

	h.store.Close()

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestChatsByUserID(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	// number of users
	n := 5

	// generating n users in database
	// test will retrieve chats for the first user
	userIDs := make([]int64, n)
	for i := range userIDs {
		id, err := h.store.CreateUser(context.Background(), mytesting.RandString())
		require.NoError(t, err)
		userIDs[i] = id
	}

	// creating chats between users [0,1], [0,2], [0,3], etc.
	chatIDs := make([]int64, n-1)
	for i, v := range mytesting.BatchUserIDs(userIDs) {
		id, err := h.store.CreateChat(context.Background(), mytesting.RandString(), v)
		require.NoError(t, err)
		chatIDs[i] = id
	}

	// creating 2 messages (author - first user) in each chatValue with 1 sec delay
	for _, v := range chatIDs {
		_, err := h.store.CreateMessage(context.Background(), v, userIDs[0], mytesting.RandString())
		require.NoError(t, err)
		time.Sleep(1 * time.Second)
		_, err = h.store.CreateMessage(context.Background(), v, userIDs[0], mytesting.RandString())
		require.NoError(t, err)
	}

	payload := bytes.NewBuffer([]byte(`{"user":` + strconv.FormatInt(userIDs[0], 10) + `}`))

	req, err := http.NewRequest("POST", "/chats/get", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.chatsByUserID)

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	body, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err)

	expected := mytesting.ReverseIDs(chatIDs)

	// extracting actual chatValue ids
	chatsValue, err := fastjson.ParseBytes(body)
	require.NoError(t, err)
	chatValues, err := chatsValue.Array()
	require.NoError(t, err)

	actual := make([]int64, 0, len(chatValues))
	for _, chatValue := range chatValues {
		chatIDValue := chatValue.Get("id")
		chatID, err := chatIDValue.Int64()
		require.NoError(t, err)

		actual = append(actual, chatID)
	}

	require.Equal(t, expected, actual)
}

func TestChatsByUserID_MalformedJson(t *testing.T) {
	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"user":"1}`))

	req, err := http.NewRequest("POST", "/chats/get", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.chatsByUserID)

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Malformed JSON\n", rr.Body.String())
}

func TestChatsByUserID_NoUserField(t *testing.T) {
	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{}`))

	req, err := http.NewRequest("POST", "/chats/get", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.chatsByUserID)

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Missing Field \"user\"\n", rr.Body.String())
}

func TestChatsByUserID_UserFieldNotInteger(t *testing.T) {
	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"user":"1"}`))

	req, err := http.NewRequest("POST", "/chats/get", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.chatsByUserID)

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Field \"user\" must be a 64-bit integer value\n", rr.Body.String())
}

func TestChatsByUserID_UserFieldInvalidUserID(t *testing.T) {
	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"user":-1}`))

	req, err := http.NewRequest("POST", "/chats/get", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.chatsByUserID)

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "Field \"user\" must be a valid user id grater than zero\n", rr.Body.String())
}

func TestChatsByUserID_UserNotExist(t *testing.T) {
	h := bootstrapHandler(t)

	// let's assume that test database will never has such sequence number in bigserial
	payload := bytes.NewBuffer([]byte(`{"user":9223372036854775807}`))

	req, err := http.NewRequest("POST", "/chats/get", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.chatsByUserID)

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "User does not exist\n", rr.Body.String())
}

func TestChatsByUserID_InternalOnChatsByUserIDCall(t *testing.T) {
	h := bootstrapHandler(t)

	payload := bytes.NewBuffer([]byte(`{"user":1}`))

	req, err := http.NewRequest("POST", "/chats/get", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.chatsByUserID)

	h.store.Close()

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestMessagesByChatID(t *testing.T) {
	t.Parallel()

	h := bootstrapHandler(t)

	// number of messages
	n := 5

	userOneID, err := h.store.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)
	userTwoID, err := h.store.CreateUser(context.Background(), mytesting.RandString())
	require.NoError(t, err)
	chatID, err := h.store.CreateChat(context.Background(), mytesting.RandString(), []int64{userOneID, userTwoID})
	require.NoError(t, err)

	messageIDs := make([]int64, n)
	for i := 0; i < n; i++ {
		id, err := h.store.CreateMessage(context.Background(), chatID, userTwoID, mytesting.RandString())
		require.NoError(t, err)
		messageIDs[i] = id
		require.NoError(t, err)
	}

	payload := bytes.NewBuffer([]byte(`{"chat":` + strconv.FormatInt(chatID, 10) + `}`))

	req, err := http.NewRequest("POST", "/messages/get", payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.messagesByChatID)

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	body, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err)

	expected := messageIDs

	// extracting actual message ids
	messagesValue, err := fastjson.ParseBytes(body)
	require.NoError(t, err)
	messageValues, err := messagesValue.Array()
	require.NoError(t, err)

	actual := make([]int64, 0, len(messageValues))
	for _, messageValue := range messageValues {
		messageIDValue := messageValue.Get("id")
		messageID, err := messageIDValue.Int64()
		require.NoError(t, err)

		actual = append(actual, messageID)
	}

	require.Equal(t, expected, actual)
}
