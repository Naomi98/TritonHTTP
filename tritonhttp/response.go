package tritonhttp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Response struct {
	Proto      string // e.g. "HTTP/1.1"
	StatusCode int    // e.g. 200
	StatusText string // e.g. "OK"

	// Headers stores all headers to write to the response.
	Headers map[string]string

	// Request is the valid request that leads to this response.
	// It could be nil for responses not resulting from a valid request.
	// Hint: you might need this to handle the "Connection: Close" requirement
	Request *Request

	// FilePath is the local path to the file to serve.
	// It could be "", which means there is no file to serve.
	FilePath string
}

const (
	responseProto = "HTTP/1.1"

	statusOK         = 200
	statusBadRequest = 400
	statusNotFound   = 404
)

var statusText = map[int]string{
	statusOK:         "OK",
	statusBadRequest: "Bad Request",
	statusNotFound:   "Not Found",
}

func (s *Server) handleGoodRequest(req *Request) (res *Response) {
	res = &Response{}
	if string(req.URL[len(req.URL)-1]) == "/" {
		req.URL = req.URL + "index.html"
	}
	docroot := s.VirtualHosts[req.Host]
	path := filepath.Join(docroot, req.URL)

	pathRel, err := filepath.Rel(docroot, path)
	if err != nil || strings.HasPrefix(pathRel, "..") {
		res.handleNotFound(req)
		return res
	}
	fi, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) || fi.IsDir() {
		res.handleNotFound(req)
		return res
	}

	res.FilePath = path
	res.handleOK(req)
	return res
}

// HandleOK prepares res to be a 200 OK response
func (res *Response) handleOK(req *Request) {
	res.init(req)
	res.StatusCode = statusOK
	res.StatusText = statusText[res.StatusCode]
	fi, _ := os.Stat(res.FilePath)
	ext := filepath.Ext(res.FilePath)
	res.Headers["Last-Modified"] = FormatTime(fi.ModTime())
	res.Headers["Content-Type"] = MIMETypeByExtension(ext)
	res.Headers["Content-Length"] = fmt.Sprint(fi.Size())
}

// HandleBadRequest prepares res to be a 400 Bad Request response
func (res *Response) handleBadRequest(req *Request) {
	res.init(req)
	res.StatusCode = statusBadRequest
	res.StatusText = statusText[res.StatusCode]
	res.Headers["Connection"] = "close"
	res.FilePath = ""
}

// HandleNotFound prepares res to be a 404 Not Found response
func (res *Response) handleNotFound(req *Request) {
	res.init(req)
	res.StatusCode = statusNotFound
	res.StatusText = statusText[res.StatusCode]
	res.FilePath = ""
}

func (res *Response) init(req *Request) {
	res.Proto = responseProto

	res.Headers = make(map[string]string)
	res.Headers["Date"] = FormatTime(time.Now())
	if req != nil && req.Close {
		res.Headers["Connection"] = "close"
	}

	res.Request = req
}

func (res *Response) Write(w io.Writer) error {
	err := res.writeStatusLine(w)
	if err != nil {
		return err
	}
	err = res.writeHeaders(w)
	if err != nil {
		return err
	}
	err = res.writeBody(w)
	if err != nil {
		return err
	}
	return nil
}

func (res *Response) writeStatusLine(w io.Writer) error {
	bw := bufio.NewWriter(w)

	statusLine := fmt.Sprintf("%v %v %v\r\n", res.Proto, res.StatusCode, statusText[res.StatusCode])
	if _, err := bw.WriteString(statusLine); err != nil {
		return err
	}

	if err := bw.Flush(); err != nil {
		return err
	}
	return nil
}

func (res *Response) writeHeaders(w io.Writer) error {
	bw := bufio.NewWriter(w)

	keys := make([]string, 0, len(res.Headers))

	for k := range res.Headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		header := fmt.Sprintf("%v: %v\r\n", k, res.Headers[k])
		if _, err := bw.WriteString(header); err != nil {
			return err
		}
	}
	if _, err := bw.WriteString("\r\n"); err != nil {
		return err
	}

	if err := bw.Flush(); err != nil {
		return err
	}
	return nil
}

func (res *Response) writeBody(w io.Writer) error {
	if res.FilePath == "" {
		return nil
	}

	bw := bufio.NewWriter(w)

	f, err := os.Open(res.FilePath)
	if err != nil {
		fmt.Println("File reading error", err)
		return nil
	}

	br := bufio.NewReader(f)

	if _, err := io.Copy(bw, br); err != nil {
		return err
	}

	if err := bw.Flush(); err != nil {
		return err
	}
	return nil
}
