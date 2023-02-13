package tritonhttp

import (
	"bufio"
	"fmt"
	"strings"
)

type Request struct {
	Method string // e.g. "GET"
	URL    string // e.g. "/path/to/a/file"
	Proto  string // e.g. "HTTP/1.1"

	// Headers stores the key-value HTTP headers
	Headers map[string]string

	Host  string // determine from the "Host" header
	Close bool   // determine from the "Connection" header
}

func readLine(br *bufio.Reader) (string, error) {
	// delimiter = "\r\n"

	token, _, err := br.ReadLine()
	return string(token), err
}

func readRequest(br *bufio.Reader) (req *Request, fRequest bool, err error) {
	// Create a Request object
	req = &Request{}

	// Read the first line of the request received
	line, err := readLine(br)
	if err != nil {
		return nil, true, err
	}

	// Parse the first line
	startLine, err := parseRequestLine(line)
	if err != nil {
		return nil, false, err
	}

	// Save the Request Method
	req.Method = startLine[0]
	if req.Method != "GET" { // Check for the method's validity
		return nil, false, fmt.Errorf("400: Invalid method")
	}

	req.URL = startLine[1] // Save the Request URL
	if string(req.URL[0]) != "/" {
		return nil, false, fmt.Errorf("400: Invalid URL")
	}

	req.Proto = startLine[2]     // Save the Request Version (HTTP/1.1)
	if req.Proto != "HTTP/1.1" { // Check for the version's validity
		return nil, false, fmt.Errorf("400: Invalid version")
	}

	// Start reading the headers
	req.Headers = make(map[string]string)
	for {
		line, err := readLine(br)
		if err != nil {
			return nil, false, err
		}

		// header end
		if line == "" {
			break
		}

		// Check for valid headers
		if !strings.Contains(line, ":") {
			return nil, false, fmt.Errorf("400: Invalid header")
		}
		// extract header information
		header := strings.Split(line, ":")
		headerKey := CanonicalHeaderKey(header[0])    // Extract Header Key
		headerVal := strings.TrimLeft(header[1], " ") // Extract Header Corresponding Value (by removing leading spaces)
		req.Headers[headerKey] = headerVal            // Save in the Header Map

	}

	// Check for HOST in Request Headers
	hostVal, hasHost := req.Headers["Host"]
	if hasHost {
		req.Host = hostVal
	} else {
		return nil, false, fmt.Errorf("400: Host Needed")
	}

	// Check for Connection in Request Header
	connectionVal, hasConnection := req.Headers["Connection"]
	if hasConnection {
		if connectionVal == "close" {
			req.Close = true
		} else {
			req.Close = false
		}
	}

	return req, false, nil
}

func parseRequestLine(line string) ([]string, error) {
	fields := strings.SplitN(line, " ", 3)

	if len(fields) != 3 {
		return []string{}, fmt.Errorf("could not parse the request line, got fields %v", fields)
	}
	return fields, nil
}
