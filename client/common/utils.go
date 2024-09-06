package common

import (
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"net"
	"strings"
)

// ReadAll reads the entire message from the socket avoiding short reads
func ReadAll(socket net.Conn, length int) ([]byte, error) {
    message := make([]byte, length)
    totalRead := 0

    for totalRead < length {
        read, err := socket.Read(message[totalRead:])
        if err != nil {
            return nil, fmt.Errorf("error reading from socket: %w", err)
        }
        totalRead += read
    }

    return message, nil
}

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


func GetBet(line string) (*Bet, error) {
    reader := csv.NewReader(strings.NewReader(line))
    record, err := reader.Read()
    if err != nil {
        return nil, fmt.Errorf("error reading CSV line: %v", err)
    }

    if len(record) != ExpectedBetFields {
        return nil, fmt.Errorf("error reading CSV line, it has %v fields, needs %v: %v",
                len(record),
                ExpectedBetFields,
                err,
            )
    }

    bet := &Bet{
        Name:         record[0],
        Surname:      record[1],
        IdentityCard: record[2],
        BirthDate:    record[3],
        Number:       record[4],
    }

    return bet, nil
}