package common

import (
	"encoding/json"
	"net/http"
	"net/url"
)

type Response struct {
	Errno int         `json:"errno"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
}

func HttpReturn(errno int, msg string, data interface{}) (re []byte, err error) {
	var (
		resp Response
	)
	resp.Errno = errno
	resp.Msg = msg
	resp.Data = data
	re, err = json.Marshal(resp)
	return
}

func GetPostData(r *http.Request, key string) (re string, err error) {
	if err = r.ParseForm(); err != nil {
		return
	}

	return r.PostForm.Get(key), nil
}

func GetRequestData(r *http.Request) (re url.Values, err error) {
	if err = r.ParseForm(); err != nil {
		return
	}

	return r.Form, nil
}
