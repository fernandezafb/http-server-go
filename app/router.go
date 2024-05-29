package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path"
	"slices"
	"strings"
)

var GZIP_ENCODING = "gzip"

type Route struct {
	pathCondition func(path string) bool
	handler       func(req HTTPRequest) HTTPResponse
}

type Router []Route

var router = Router{
	Route{
		pathCondition: func(path string) bool {
			return path == "/"
		},
		handler: func(req HTTPRequest) HTTPResponse {
			return HTTPResponse{
				Status: fmt.Sprintf(" %d %s", 200, statusCodes[200]),
			}
		},
	},
	Route{
		pathCondition: func(path string) bool {
			return strings.HasPrefix(path, "/echo/")
		},
		handler: func(req HTTPRequest) HTTPResponse {
			return echoHandler(req)
		},
	},
	Route{
		pathCondition: func(path string) bool {
			return path == "/user-agent"
		},
		handler: func(req HTTPRequest) HTTPResponse {
			var userAgent string
			for _, header := range req.Headers {
				if strings.HasPrefix(header, "user-agent") {
					userAgent = strings.Split(header, ": ")[1]
					break
				}
			}

			return HTTPResponse{
				Status: fmt.Sprintf(" %d %s", 200, statusCodes[200]),
				Headers: []string{
					"Content-Type: text/plain",
					fmt.Sprintf("Content-Length: %d", len(userAgent)),
				},
				Body: userAgent,
			}
		},
	},
	Route{
		pathCondition: func(path string) bool {
			return strings.HasPrefix(path, "/files/")
		},
		handler: func(req HTTPRequest) HTTPResponse {
			switch req.Method {
			case "GET":
				return processFileGetRequest(req)
			case "POST":
				return processFilePostRequest(req)
			default:
				return HTTPResponse{
					Status: fmt.Sprintf(" %d %s", 404, statusCodes[404]),
				}
			}
		},
	},
}

func echoHandler(req HTTPRequest) HTTPResponse {
	body := strings.Split(req.Path, "/")[2]
	responseHeaders := []string{}

	// Search for valid encoding in the headers
	for _, header := range req.Headers {
		headerParts := strings.Split(header, ": ")
		headerValues := strings.Split(headerParts[1], ", ")

		// Process 'accept-encoding' header with a valid gzip encoding
		if headerParts[0] == "accept-encoding" && slices.Contains(headerValues, GZIP_ENCODING) {
			responseHeaders = append(responseHeaders, fmt.Sprintf("Content-Encoding: %s", GZIP_ENCODING))
			compressedBody, err := compress(body)
			if err != nil {
				return HTTPResponse{
					Status: fmt.Sprintf(" %d %s", 500, statusCodes[500]),
				}
			}
			body = string(compressedBody)
		}
	}

	// Add content type and length headers
	responseHeaders = append(responseHeaders, "Content-Type: text/plain")
	responseHeaders = append(responseHeaders, fmt.Sprintf("Content-Length: %d", len(body)))

	return HTTPResponse{
		Status:  fmt.Sprintf(" %d %s", 200, statusCodes[200]),
		Headers: responseHeaders,
		Body:    body,
	}
}

func processFileGetRequest(req HTTPRequest) HTTPResponse {
	filePath := path.Join(req.DirectoryPath, strings.Split(req.Path, "/")[2])
	file, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return HTTPResponse{
				Status: fmt.Sprintf(" %d %s", 404, statusCodes[404]),
			}
		}

		return HTTPResponse{
			Status: fmt.Sprintf(" %d %s", 500, statusCodes[500]),
		}
	}

	return HTTPResponse{
		Status: fmt.Sprintf(" %d %s", 200, statusCodes[200]),
		Headers: []string{
			"Content-Type: application/octet-stream",
			fmt.Sprintf("Content-Length: %d", len(file)),
		},
		Body: string(file),
	}
}

func processFilePostRequest(req HTTPRequest) HTTPResponse {
	filePath := path.Join(req.DirectoryPath, strings.Split(req.Path, "/")[2])

	err := os.WriteFile(filePath, bytes.Trim(req.Body, "\x00"), 0644)
	if err != nil {
		return HTTPResponse{
			Status: fmt.Sprintf(" %d %s", 500, statusCodes[500]),
		}
	}

	return HTTPResponse{
		Status: fmt.Sprintf(" %d %s", 201, statusCodes[201]),
	}
}

func compress(body string) ([]byte, error) {
	var b bytes.Buffer

	gz := gzip.NewWriter(&b)
	defer gz.Close()

	if _, err := gz.Write([]byte(body)); err != nil {
		return nil, err
	}

	// Explicitly close the writer: https://www.joeshaw.org/dont-defer-close-on-writable-files/
	if err := gz.Close(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
