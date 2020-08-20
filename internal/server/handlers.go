package server

import (
	"avito-trainee-assignment/internal/storage"
	"encoding/json"
	"errors"
	"github.com/valyala/fastjson"
	"go.uber.org/zap"
	"io/ioutil"
	"mime"
	"net/http"
	"strconv"
)

// TODO maybe omit error handling after call to parser.Parse as provided body already contains valid JSON

type parsers struct {
	createChatPool       fastjson.ParserPool
	createMessagePool    fastjson.ParserPool
	chatsByUserIDPool    fastjson.ParserPool
	messagesByChatIDPool fastjson.ParserPool
}

type handler struct {
	logger  *zap.SugaredLogger
	store   *storage.Store
	parsers parsers
}

// enforcePOSTJSON is a middleware pre-processing each HTTP request
// it checks for POST method, application/json Content-Type header and valid json body
// it also sets blank Content-Type header to application/json
func enforcePOSTJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.Header().Set("Allow", "POST")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		// check "Content-Type" header
		contentType := r.Header.Get("Content-Type")
		if contentType != "" {
			mt, _, err := mime.ParseMediaType(contentType)
			if err != nil {
				http.Error(w, "Malformed Content-Type header", http.StatusBadRequest)
				return
			}

			if mt != "application/json" {
				http.Error(w, "Content-Type header must be application/json", http.StatusUnsupportedMediaType)
				return
			}
		} else {
			r.Header.Set("Content-Type", "application/json")
		}

		// check if provided request body is valid JSON
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Can not read request body", http.StatusBadRequest)
			return
		}

		err = fastjson.ValidateBytes(body)
		if err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// createUser handles HTTP requests on "/users/add" endpoint
func (h *handler) createUser(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	if !fastjson.Exists(body, "username") {
		http.Error(w, "Missing Field \"username\"", http.StatusBadRequest)
		return
	}

	username := fastjson.GetString(body, "username")
	if len(username) == 0 {
		http.Error(w, "Field \"username\" must be a string and have non-zero length", http.StatusBadRequest)
		return
	}

	id, err := h.store.CreateUser(r.Context(), username)
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			http.Error(w, "User already exists", http.StatusBadRequest)
			return
		}
		h.logger.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	payload := []byte(`{"id":` + strconv.FormatInt(id, 10) + `}`)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write(payload)
	if err != nil {
		h.logger.Errorf("writing marshaled data to ResponseWriter: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

// createChat handles HTTP requests on "/chats/add" endpoint
func (h *handler) createChat(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)

	parser := h.parsers.createChatPool.Get()
	defer h.parsers.createChatPool.Put(parser)
	v, err := parser.ParseBytes(body)
	if err != nil {
		http.Error(w, "Malformed JSON object", http.StatusBadRequest)
		return
	}

	// retrieving chat name
	if !v.Exists("name") {
		http.Error(w, "Missing Field \"name\"", http.StatusBadRequest)
		return
	}

	nameValue := v.Get("name")
	if nameValue.Type() != fastjson.TypeString {
		http.Error(w, "Field \"name\" must be a string", http.StatusBadRequest)
		return
	}

	name := string(nameValue.MarshalTo(nil))
	if len(name) == 0 {
		http.Error(w, "Field \"name\" must have non-zero length", http.StatusBadRequest)
		return
	}

	// retrieving users array
	if !v.Exists("users") {
		http.Error(w, "Missing Field \"users\"", http.StatusBadRequest)
		return
	}

	userValues, err := v.Get("users").Array()
	if err != nil {
		http.Error(w, "Field \"users\" must be an array", http.StatusBadRequest)
		return
	}

	userIDs := make([]int64, 0, len(userValues))
	for _, v := range userValues {
		userID, err := v.Int64()
		if err != nil {
			http.Error(w, "Each item in \"users\" array field must be a 64-bit integer value", http.StatusBadRequest)
			return
		}

		if userID < 1 {
			http.Error(w, "Each integer in \"users\" array must be a valid user id grater than zero", http.StatusBadRequest)
			return
		}
		userIDs = append(userIDs, userID)
	}

	h.parsers.createChatPool.Put(parser)

	// creating chat
	id, err := h.store.CreateChat(r.Context(), name, userIDs)
	if err != nil {
		switch err {
		case storage.ErrChatExists:
			http.Error(w, "Chat already exists", http.StatusBadRequest)
			return
		case storage.ErrChatBadUsers:
			http.Error(w, "Bad user list", http.StatusBadRequest)
			return
		default:
			h.logger.Error(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	// returning id
	payload := []byte(`{"id":` + strconv.FormatInt(id, 10) + `}`)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write(payload)
	if err != nil {
		h.logger.Errorf("writing marshaled data to ResponseWriter: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

// createMessage handles HTTP requests on "/messages/add" endpoint
func (h *handler) createMessage(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)

	parser := h.parsers.createMessagePool.Get()
	defer h.parsers.createMessagePool.Put(parser)
	v, err := parser.ParseBytes(body)
	if err != nil {
		http.Error(w, "Malformed JSON object", http.StatusBadRequest)
		return
	}

	// retrieving chat id
	chatValue := v.Get("chat")
	chatID, err := chatValue.Int64()
	if err != nil {
		http.Error(w, "Field \"chat\" must be a 64-bit integer value", http.StatusBadRequest)
		return
	}

	if chatID < 0 {
		http.Error(w, "Field \"chat\" must be a valid chat id grater than zero", http.StatusBadRequest)
		return
	}

	// retrieving author id
	authorValue := v.Get("author")
	authorID, err := authorValue.Int64()
	if err != nil {
		http.Error(w, "Field \"author\" must be a 64-bit integer value", http.StatusBadRequest)
		return
	}

	if authorID < 0 {
		http.Error(w, "Field \"author\" must be a valid chat id grater than zero", http.StatusBadRequest)
		return
	}

	// retrieving text
	textValue := v.Get("text")
	if textValue.Type() != fastjson.TypeString {
		http.Error(w, "Field \"text\" must be a string", http.StatusBadRequest)
		return
	}

	text := string(textValue.MarshalTo(nil))
	if len(text) < 0 {
		http.Error(w, "Field \"text\" must have non-zero length", http.StatusBadRequest)
		return
	}

	h.parsers.createMessagePool.Put(parser)

	// creating message
	id, err := h.store.CreateMessage(r.Context(), chatID, authorID, text)
	if err != nil {
		switch err {
		case storage.ErrChatNotExist:
			http.Error(w, "Chat with provided id does not exist", http.StatusBadRequest)
			return
		case storage.ErrUserNotExist:
			http.Error(w, "Author with provided id does not exist", http.StatusBadRequest)
			return
		default:
			h.logger.Error(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	// returning id
	payload := []byte(`{"id":` + strconv.FormatInt(id, 10) + `}`)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write(payload)
	if err != nil {
		h.logger.Errorf("writing marshaled data to ResponseWriter: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

// chatsByUserID handles HTTP requests on "/chats/get" endpoint
func (h *handler) chatsByUserID(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)

	parser := h.parsers.chatsByUserIDPool.Get()
	defer h.parsers.chatsByUserIDPool.Put(parser)
	v, err := parser.ParseBytes(body)
	if err != nil {
		http.Error(w, "Malformed JSON object", http.StatusBadRequest)
		return
	}

	userIDValue := v.Get("user")
	userID, err := userIDValue.Int64()
	if err != nil {
		http.Error(w, "Field \"user\" must be a 64-bit integer value", http.StatusBadRequest)
		return
	}

	h.parsers.chatsByUserIDPool.Put(parser)

	chats, err := h.store.ChatsByUserID(r.Context(), userID)
	if err != nil {
		switch err {
		case storage.ErrUserNotExist:
			http.Error(w, "User does not exist", http.StatusBadRequest)
			return
		default:
			h.logger.Error(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	payload, err := json.Marshal(chats)
	if err != nil {
		h.logger.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(payload)
	if err != nil {
		h.logger.Errorf("writing marshaled data to ResponseWriter: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

// messagesByChatID handles HTTP requests on "/messages/get" endpoint
func (h *handler) messagesByChatID(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)

	parser := h.parsers.messagesByChatIDPool.Get()
	defer h.parsers.messagesByChatIDPool.Put(parser)
	v, err := parser.ParseBytes(body)
	if err != nil {
		http.Error(w, "Malformed JSON object", http.StatusBadRequest)
		return
	}

	chatIDValue := v.Get("chat")
	chatID, err := chatIDValue.Int64()
	if err != nil {
		http.Error(w, "Field \"chat\" must be a 64-bit integer value", http.StatusBadRequest)
		return
	}

	h.parsers.messagesByChatIDPool.Put(parser)

	messages, err := h.store.MessagesByChatID(r.Context(), chatID)
	if err != nil {
		switch err {
		case storage.ErrChatNotExist:
			http.Error(w, "Chat does not exist", http.StatusBadRequest)
			return
		default:
			h.logger.Error(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	payload, err := json.Marshal(messages)
	if err != nil {
		h.logger.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(payload)
	if err != nil {
		h.logger.Errorf("writing marshaled data to ResponseWriter: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
