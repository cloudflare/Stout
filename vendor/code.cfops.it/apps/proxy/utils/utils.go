package utils

import (
	"log"
	"math/rand"
	"net"
	"net/http"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_")

type BugsnagLogger struct{}

func (l BugsnagLogger) Printf(format string, v ...interface{}) {
	log.Printf(format, v)
	return
}

func FailResponseIf(w http.ResponseWriter, err error) bool {
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte("Incorrectly formatted request issued."))
		return true
	}

	return false
}

func PanicIf(err error) {
	if err != nil {
		panic(err)
	}
}

func RandString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func GetFreePort() int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func CopyRequest(r *http.Request) *http.Request {
	return &http.Request{
		Method:           r.Method,
		URL:              r.URL,
		Proto:            r.Proto,
		ProtoMajor:       r.ProtoMajor,
		ProtoMinor:       r.ProtoMinor,
		Header:           r.Header,
		Body:             r.Body,
		ContentLength:    r.ContentLength,
		TransferEncoding: r.TransferEncoding,
		Close:            r.Close,
		Host:             r.Host,
		Form:             r.Form,
		PostForm:         r.PostForm,
		MultipartForm:    r.MultipartForm,
		Trailer:          r.Trailer,
		RemoteAddr:       r.RemoteAddr,
		TLS:              r.TLS,
		Cancel:           r.Cancel,
		Response:         r.Response,
	}
}
