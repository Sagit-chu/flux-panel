package middleware

import (
	"fmt"
	"net/http"

	"go-backend/internal/http/response"
)

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				response.WriteJSON(w, response.Err(-2, fmt.Sprint(rec)))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
