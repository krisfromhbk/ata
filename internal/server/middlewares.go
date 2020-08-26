package server

import (
	"avito-trainee-assignment/internal/storage/zapadapter"
	"bytes"
	"github.com/rs/xid"
	"github.com/valyala/fastjson"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
)

// enforcePostJson is a middleware pre-processing each HTTP request
// it checks for POST method, application/json Content-Type header and valid json body
// it also sets blank Content-Type header to application/json
func enforcePostJson(next http.Handler) http.Handler {
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
		var bodyBuf bytes.Buffer
		bodyReader := io.TeeReader(r.Body, &bodyBuf)
		body, err := ioutil.ReadAll(bodyReader)
		if err != nil {
			http.Error(w, "Can not read request body", http.StatusBadRequest)
			return
		}

		if len(body) == 0 {
			http.Error(w, "No body provided", http.StatusBadRequest)
			return
		}

		err = fastjson.ValidateBytes(body)
		if err != nil {
			http.Error(w, "Malformed JSON", http.StatusBadRequest)
			return
		}

		r.Body = ioutil.NopCloser(&bodyBuf)

		next.ServeHTTP(w, r)
	})
}

func log(next http.Handler, logger *zap.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := xid.New().String()

		ctx := zapadapter.NewContextWithID(r.Context(), id)
		rwID := r.WithContext(ctx)

		logger.Info("incoming http request",
			zap.String("id", id),
			zap.String("method", r.Method),
			zap.String("uri", r.URL.RequestURI()),
			zap.String("ip", r.RemoteAddr),
		)

		next.ServeHTTP(w, rwID)
	})
}
