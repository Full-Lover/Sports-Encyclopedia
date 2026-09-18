package webexperience

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/full-lover/sports-encyclopedia/internal/publishedatlas"
)

var requestSequence atomic.Uint64

type errorEnvelope struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"requestId"`
	} `json:"error"`
}

func registerAtlasRoutes(mux *http.ServeMux, reader publishedatlas.Reader) {
	mux.HandleFunc("GET /_atlas/snapshots/{snapshotId}/map", func(response http.ResponseWriter, request *http.Request) {
		snapshotID, err := publishedatlas.ParseSnapshotID(request.PathValue("snapshotId"))
		if err != nil {
			writeAtlasError(response, http.StatusBadRequest, "INVALID_REQUEST", "The request is invalid.")
			return
		}

		result, err := reader.Read(request.Context(), publishedatlas.ReadRequest{
			Kind:       publishedatlas.ReadMap,
			SnapshotID: snapshotID,
		})
		if err != nil {
			var fault *publishedatlas.ReadFault
			if errors.As(err, &fault) && fault.Code == publishedatlas.FaultSnapshotMissing {
				writeAtlasError(response, http.StatusNotFound, "SNAPSHOT_NOT_FOUND", "The requested snapshot is unavailable.")
				return
			}
			writeAtlasError(response, http.StatusInternalServerError, "INTERNAL_ERROR", "The request could not be completed.")
			return
		}
		if result.Map == nil {
			writeAtlasError(response, http.StatusInternalServerError, "INTERNAL_ERROR", "The request could not be completed.")
			return
		}

		response.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		response.Header().Set("X-Content-Type-Options", "nosniff")
		_ = json.NewEncoder(response).Encode(result.Map)
	})
}

func writeAtlasError(response http.ResponseWriter, status int, code string, message string) {
	payload := errorEnvelope{}
	payload.Error.Code = code
	payload.Error.Message = message
	payload.Error.RequestID = fmt.Sprintf("request-%d", requestSequence.Add(1))

	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.Header().Set("X-Content-Type-Options", "nosniff")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(payload)
}
