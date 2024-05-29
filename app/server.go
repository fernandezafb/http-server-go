package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
)

const (
	CRLF         = "\r\n"
	HTTP_VERSION = "HTTP/1.1"
)

var (
	statusCodes = map[int]string{
		200: "OK",
		201: "Created",
		404: "Not Found",
	}
)

type HTTPRequest struct {
	Method  string
	Path    string
	Version string
	Headers []string
	Body    []byte
	DirectoryPath string
}

type HTTPResponse struct {
	Status  string
	Headers []string
	Body    string
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	// Parse the command line arguments for '--directory' flag
	directoryPath := flag.String("directory", "", "The directory to serve files from. Defaults to the current directory.")
	flag.Parse()

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleRequest(c, *directoryPath)
	}
}

func handleRequest(c net.Conn, directoryPath string) {
	defer c.Close()

	req := make([]byte, 1024)

	_, err := c.Read(req)
	if err != nil {
		fmt.Println("Failed to read request: ", err.Error())
		os.Exit(1)
	}

	request := parseRequest(req, directoryPath)

	for _, route := range router {
		if !route.pathCondition(request.Path) {
			continue
		}

		response := route.handler(request)
		writeResponse(c, parseResponse(response))
		return
	}

	notFoundResponse := HTTPResponse{
		Status: fmt.Sprintf(" %d %s", 404, statusCodes[404]),
	}
	writeResponse(c, parseResponse(notFoundResponse))
}

func parseRequest(req []byte, directoryPath string) HTTPRequest {
	sections := strings.Split(string(req), CRLF)
	params := strings.Fields(sections[0])
	body := sections[len(sections)-1]

	// Header names are case-insensitive, convert them to lowercase
	headers := []string{}
	for _, header := range sections[1 : len(sections)-2] {
		headers = append(headers, strings.ToLower(header))
	}

	return HTTPRequest{
		Method:  params[0],
		Path:    params[1],
		Version: params[2],
		Headers: headers,
		Body:    []byte(body),
		DirectoryPath: directoryPath,
	}
}

func parseResponse(res HTTPResponse) []byte {
	var builder strings.Builder

	builder.WriteString(HTTP_VERSION)
	builder.WriteString(res.Status)
	builder.WriteString(CRLF)

	for _, header := range res.Headers {
		builder.WriteString(header)
		builder.WriteString(CRLF)
	}

	builder.WriteString(CRLF)
	builder.WriteString(res.Body)

	return []byte(builder.String())
}

func writeResponse(c net.Conn, res []byte) {
	_, err := c.Write(res)
	if err != nil {
		fmt.Println("Failed writing response: ", err.Error())
		os.Exit(1)
	}
}
