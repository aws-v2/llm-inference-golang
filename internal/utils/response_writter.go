   package utils

import (
	"encoding/json"
	"net/http"
)

// --- Standard Response Envelope ---

type apiResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}



func WriteJSONSucces(w http.ResponseWriter, statusCode int, message string , data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(apiResponse{
		Message: message,
		Code: statusCode,
		Data: data,
	})
}



func WriteJSONError(w http.ResponseWriter, statusCode int, err error)  {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(apiResponse{
		Message: err.Error(),
		Code: statusCode,
	})

	return
}
