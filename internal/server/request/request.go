package request

import (
	"encoding/json"
	"net/http"
)

func ReadJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
