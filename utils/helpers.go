package utils

import (
	"fmt"
	"net"
	"os"
)

// fileExists checks if a file exists
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// FindAvailableAPIPort finds an available port for the API server
func FindAvailableAPIPort() int {
	port := 8080
	for {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			listener.Close()
			return port
		}
		port++
	}
}
