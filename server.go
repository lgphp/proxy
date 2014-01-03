package proxy

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/howeyc/fsnotify"
	"io"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"time"
)

type (
	incomingRequest struct {
		net.Conn
		reader  *bufio.Reader
		buffer  io.ReadWriter
		request *http.Request
		opened  time.Time
		err     error
	}
	nilCloser struct {
		*bytes.Buffer
	}
)

func (this nilCloser) Close() error {
	return nil
}

func Listen(configFilename string) {
	p := newProxy()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalln("error", err)
	}
	defer watcher.Close()
	err = watcher.Watch(filepath.Dir(configFilename))
	if err != nil {
		log.Fatalln("error", err)
	}
	go func() {
		for evt := range watcher.Event {
			if evt.IsModify() && filepath.Base(evt.Name) == filepath.Base(configFilename) {
				p.reload(configFilename)
			}
		}
	}()
	go p.reload(configFilename)
	p.start()
}

func (this incomingRequest) writeError(err string, code int) {
	hdr := make(http.Header)
	hdr.Set("Connection", "close")
	hdr.Set("Content-Type", "text/plain")

	body := nilCloser{bytes.NewBufferString(err)}

	res := &http.Response{
		Status:        fmt.Sprint(code, " ", err),
		StatusCode:    code,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        hdr,
		Body:          body,
		ContentLength: int64(body.Len()),
		Close:         true,
		Request:       this.request,
	}
	res.Write(this.Conn)
	this.Conn.Close()
}