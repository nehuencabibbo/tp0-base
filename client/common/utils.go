package common

import (
	"encoding/binary"
	"fmt"
	"net"
)

// SendAll sends message thorugh socket avoiding short writes
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

// appendStringWithItsLength appends the length of a string as a u16 and the string 
// itself to a byte array
func AppendStringWithItsLength(s string, data []byte) []byte {
	length := uint32(len(s))
	lengthBytes := make([]byte, 4)

	binary.BigEndian.PutUint32(lengthBytes, length)
	
	// https://stackoverflow.com/questions/39993688/are-slices-passed-by-value
	data = append(data, lengthBytes...)
	data = append(data, []byte(s)...)

	return data
}