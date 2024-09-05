package common

import (
	"encoding/binary"
)

const Separator = "#"

// Bet represts a bet made by a specific client
type Bet struct {
	name string
	surname string
	identityCard string
	birthDate string
	number string
}

// FormatToSend Formats the corresponding Bet to it's representation
// in the protocol used
func (b *Bet) FormatToSend(agencyNumber string) []byte {
	message := agencyNumber + Separator + 
		b.name + Separator + 
		b.surname + Separator + 
		b.identityCard + Separator + 
		b.birthDate + Separator + 
		b.number
	
    var data_to_send []byte
	data_to_send = appendStringWithItsLength(message, data_to_send)

    return data_to_send
}

// appendStringWithItsLength appends the length of a string as a u16 and the string 
// itself to a byte array
func appendStringWithItsLength(s string, data []byte) []byte {
	length := uint32(len(s))
	lengthBytes := make([]byte, 4)

	binary.BigEndian.PutUint32(lengthBytes, length)
	
	// https://stackoverflow.com/questions/39993688/are-slices-passed-by-value
	data = append(data, lengthBytes...)
	data = append(data, []byte(s)...)

	return data
}