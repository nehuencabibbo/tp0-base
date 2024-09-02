package common

import (
	"fmt"
	"net"
)


func SendAll(message []byte, socket net.Conn) error {
	totalWritten := 0
	// This is done in order to avoid short writes:
	// https://cs61.seas.harvard.edu/site/2018/FileDescriptors/
    for totalWritten < len(message) {
        written, err := socket.Write(message[totalWritten:])
        if err != nil {
            return fmt.Errorf("error writing to socket: %w", err)
        }
        totalWritten += written
    }

	return nil
}