package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
)

var _ = net.Listen
var _ = os.Exit

func main() {
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	var directory string
	flag.StringVar(&directory, "directory", ".", "Directory to serve files from")
	flag.Parse()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		fmt.Println("Accepted connection from", conn.RemoteAddr())
		go handleConnection(conn, directory)
	}
}

func handleConnection(conn net.Conn, directory string) {
	defer conn.Close()

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading request:", err)
		return
	}

	requestStr := string(buf[:n])
	fmt.Printf("Received request: %s\n", requestStr)

	if strings.HasPrefix(requestStr, "GET ") {
		firstLine := strings.Split(requestStr, "\r\n")[0]
		parts := strings.Split(firstLine, " ")

		if len(parts) >= 2 {
			path := parts[1]
			fmt.Printf("Received request for path: %s\n", path)

			if path == "/" {
				conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
			} else if strings.HasPrefix(path, "/echo/") {
				echoText := strings.TrimPrefix(path, "/echo/")

				acceptEncoding := extractHeader(requestStr, "Accept-Encoding")
				var contentEncoding string
				if strings.Contains(acceptEncoding, "gzip") {
					contentEncoding = "gzip"
				}

				response := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n"
				if contentEncoding != "" {
					response += fmt.Sprintf("Content-Encoding: %s\r\n", contentEncoding)
				}
				response += fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(echoText), echoText)

				conn.Write([]byte(response))
			} else if path == "/user-agent" {
				userAgent := extractHeader(requestStr, "User-Agent")
				if userAgent != "" {
					response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(userAgent), userAgent)
					conn.Write([]byte(response))
				}
			} else if strings.HasPrefix(path, "/files/") {
				fileName := strings.TrimPrefix(path, "/files/")
				filePath := fmt.Sprintf("%s/%s", directory, fileName)
				file, err := os.ReadFile(filePath)
				fmt.Sprintf("Received request for file: %s\n", fileName)
				fmt.Printf("File path: %s\n", filePath)
				fmt.Printf("File content: %s\n", string(file))
				if err != nil {
					conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
					return
				}
				response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(file), file)
				conn.Write([]byte(response))
			} else {
				conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
			}
		}
	} else if strings.HasPrefix(requestStr, "POST ") {
		firstLine := strings.Split(requestStr, "\r\n")[0]
		parts := strings.Split(firstLine, " ")

		if len(parts) >= 2 {
			fileName := strings.TrimPrefix(parts[1], "/files/")
			fmt.Printf("Received POST request for file: %s\n", fileName)
			filePath := fmt.Sprintf("%s/%s", directory, fileName)
			fileContent := strings.TrimSpace(strings.Join(strings.Split(requestStr, "\r\n\r\n")[1:], "\r\n\r\n"))
			err := os.WriteFile(filePath, []byte(fileContent), 0644)
			if err != nil {
				conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
				return
			}
			response := fmt.Sprintf("HTTP/1.1 201 Created\r\n\r\n")
			conn.Write([]byte(response))
		}
	}
}

func extractHeader(request string, headerName string) string {
	lines := strings.Split(request, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, headerName+":") {
			return strings.TrimSpace(strings.TrimPrefix(line, headerName+":"))
		}
	}
	return ""
}
