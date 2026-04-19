package output

import (
	"encoding/json"
	"fmt"
	"io"
)

type Response struct {
	OK      bool           `json:"ok"`
	Profile string         `json:"profile"`
	Data    map[string]any `json:"data"`
	Error   *ResponseError `json:"error"`
}

type ResponseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func WriteJSON(w io.Writer, resp Response) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(resp)
}

func WriteHuman(w io.Writer, resp Response) error {
	if resp.OK {
		_, err := fmt.Fprintf(w, "ok profile=%s\n", resp.Profile)
		return err
	}

	if resp.Error == nil {
		_, err := fmt.Fprintln(w, "error")
		return err
	}

	_, err := fmt.Fprintf(w, "%s: %s\n", resp.Error.Code, resp.Error.Message)
	return err
}
