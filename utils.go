package main

import "net/http"
import "encoding/json"

func decodeJSONFromHTTP(w http.ResponseWriter, r *http.Request, v interface{}) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024*1024))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		http.Error(w, err.Error(), 400)
		return err
	}
	return nil
}
