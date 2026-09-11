package storage

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// TUSRouteHandler exposes the TUSManager over HTTP. Routes are dispatched by
// method and path prefix following the server's stdlib ServeMux conventions.
type TUSRouteHandler struct {
	Manager *TUSManager
}

type tusCreateRequest struct {
	Size        int64  `json:"size"`
	Checksum    string `json:"checksum"`
	ContentType string `json:"content_type"`
}

type tusCreateResponse struct {
	ID string `json:"id"`
}

type tusAppendRequest struct {
	Offset int64  `json:"offset"`
	Data   string `json:"data"` // base64-encoded chunk
}

type tusOffsetResponse struct {
	ID          string `json:"id"`
	Size        int64  `json:"size"`
	Offset      int64  `json:"offset"`
	ContentType string `json:"content_type"`
	Checksum    string `json:"checksum"`
}

type tusErrorResponse struct {
	Error string `json:"error"`
}

func (h TUSRouteHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if h.Manager == nil {
		writeTUSJSON(writer, http.StatusServiceUnavailable, tusErrorResponse{Error: "upload service unavailable"})
		return
	}
	path := request.URL.Path
	if !strings.HasPrefix(path, "/uploads") {
		writeTUSJSON(writer, http.StatusNotFound, tusErrorResponse{Error: "not found"})
		return
	}
	remainder := strings.TrimPrefix(path, "/uploads")
	switch {
	case remainder == "" && request.Method == http.MethodPost:
		h.handleCreate(writer, request)
	case remainder == "" && request.Method != http.MethodPost:
		writer.Header().Set("Allow", "POST")
		writeTUSJSON(writer, http.StatusMethodNotAllowed, tusErrorResponse{Error: "POST is required"})
	case strings.HasSuffix(remainder, "/chunks") && request.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(remainder, "/"), "/chunks")
		h.handleAppend(writer, request, id)
	case strings.HasSuffix(remainder, "/offset") && request.Method == http.MethodGet:
		id := strings.TrimSuffix(strings.TrimPrefix(remainder, "/"), "/offset")
		h.handleOffset(writer, request, id)
	case strings.HasSuffix(remainder, "/complete") && request.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(remainder, "/"), "/complete")
		h.handleComplete(writer, request, id)
	case strings.HasSuffix(remainder, "/abort") && request.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(remainder, "/"), "/abort")
		h.handleAbort(writer, request, id)
	default:
		writeTUSJSON(writer, http.StatusNotFound, tusErrorResponse{Error: "not found"})
	}
}

func (h TUSRouteHandler) handleCreate(writer http.ResponseWriter, request *http.Request) {
	var req tusCreateRequest
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		writeTUSJSON(writer, http.StatusBadRequest, tusErrorResponse{Error: "invalid JSON request"})
		return
	}
	id, err := h.Manager.Create(request.Context(), req.Size, req.Checksum, req.ContentType)
	if err != nil {
		writeTUSJSON(writer, http.StatusBadRequest, tusErrorResponse{Error: err.Error()})
		return
	}
	writeTUSJSON(writer, http.StatusCreated, tusCreateResponse{ID: id})
}

func (h TUSRouteHandler) handleAppend(writer http.ResponseWriter, request *http.Request, id string) {
	var req tusAppendRequest
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		writeTUSJSON(writer, http.StatusBadRequest, tusErrorResponse{Error: "invalid JSON request"})
		return
	}
	data, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		writeTUSJSON(writer, http.StatusBadRequest, tusErrorResponse{Error: "base64 data is invalid"})
		return
	}
	next, err := h.Manager.Append(request.Context(), id, req.Offset, data)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrUploadNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, ErrUploadOffsetMismatch) {
			status = http.StatusConflict
		}
		writeTUSJSON(writer, status, tusErrorResponse{Error: err.Error()})
		return
	}
	writeTUSJSON(writer, http.StatusOK, map[string]int64{"offset": next})
}

func (h TUSRouteHandler) handleOffset(writer http.ResponseWriter, request *http.Request, id string) {
	session, err := h.Manager.Offset(request.Context(), id)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrUploadNotFound) {
			status = http.StatusNotFound
		}
		writeTUSJSON(writer, status, tusErrorResponse{Error: err.Error()})
		return
	}
	writeTUSJSON(writer, http.StatusOK, tusOffsetResponse{
		ID: session.ID, Size: session.Size, Offset: session.Offset,
		ContentType: session.ContentType, Checksum: session.Checksum,
	})
}

func (h TUSRouteHandler) handleComplete(writer http.ResponseWriter, request *http.Request, id string) {
	object, err := h.Manager.Complete(request.Context(), id)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrUploadNotFound) {
			status = http.StatusNotFound
		}
		writeTUSJSON(writer, status, tusErrorResponse{Error: err.Error()})
		return
	}
	writeTUSJSON(writer, http.StatusOK, map[string]string{
		"key":          object.Key,
		"content_type": object.ContentType,
	})
}

func (h TUSRouteHandler) handleAbort(writer http.ResponseWriter, request *http.Request, id string) {
	if err := h.Manager.Abort(request.Context(), id); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrUploadNotFound) {
			status = http.StatusNotFound
		}
		writeTUSJSON(writer, status, tusErrorResponse{Error: err.Error()})
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func writeTUSJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}

// Compile-time check that TUSRouteHandler satisfies http.Handler.
var _ http.Handler = TUSRouteHandler{}
