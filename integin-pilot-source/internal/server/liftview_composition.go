package server

import (
	"net/http"

	"integin/internal/liftviewexport"
)

// NewLiftViewExportHandler builds the simulator export handler. It needs no
// database: solve, gates, and drawing are stateless engine calls.
func NewLiftViewExportHandler() (http.Handler, error) {
	return liftviewexport.Handler{}, nil
}
