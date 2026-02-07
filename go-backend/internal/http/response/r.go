package response

import (
	"encoding/json"
	"net/http"
	"time"
)

type R struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	TS   int64       `json:"ts"`
	Data interface{} `json:"data,omitempty"`
}

func OK(data interface{}) R {
	return R{
		Code: 0,
		Msg:  "操作成功",
		TS:   time.Now().UnixMilli(),
		Data: data,
	}
}

func OKEmpty() R {
	return R{
		Code: 0,
		Msg:  "操作成功",
		TS:   time.Now().UnixMilli(),
	}
}

func Err(code int, msg string) R {
	return R{
		Code: code,
		Msg:  msg,
		TS:   time.Now().UnixMilli(),
	}
}

func ErrDefault(msg string) R {
	return Err(-1, msg)
}

func WriteJSON(w http.ResponseWriter, payload R) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(payload)
}
