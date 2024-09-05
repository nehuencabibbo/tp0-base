package common

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
)

type Protocol struct {}

// ReadMessageType Reads the server response type, 
// following the described protocol
func (p* Protocol) ReadMessageType(sock net.Conn) (int, error) {
	messageType, err := ReadAll(sock, 1)
    if err != nil {
        return -1, err
    }

	intMessageType := int(messageType[0])
	
    return intMessageType, nil
}


func (p *Protocol) SendMessage(messageCode byte, body []byte, sock net.Conn) error {
	var data []byte
	data = append(data, messageCode)
	data = append(data, body...)

    err := SendAll(data, sock)
    if err != nil {
        return fmt.Errorf("error sending message: %w", err)
    }

    return nil
}


func (p *Protocol) GetLotteryWinners(sock net.Conn) ([]string, error) {
	needToRead, err := ReadAll(sock, WinnersLengthBytes)
	if err != nil {
		return []string{}, err 
	}

	needToReadUInt32 := binary.BigEndian.Uint32(needToRead)
	var winners[]string
	for needToReadUInt32 > 0 {
		winnerDocument, err := ReadAll(sock, DocumentBytes)
		if err != nil {
			return []string{}, err 
		}

		winners = append(winners, string(winnerDocument))
		
		needToReadUInt32 -= DocumentBytes

	}

	return winners, nil
}


// sendBatch Sends a batch according to the described protocol and logs every step during
// the process. 
// BatchData needs to be a byte array of the bets formated according to the protocol, no
// extra data will be added, Sending a batch of N bets is the same as sending N bets 
func (p *Protocol) SendBatch(betsInBatch int, batchNumber int, batchData []byte, sock net.Conn) error {
	err := p.sendBatchStart(uint8(betsInBatch), sock)
	if err != nil {
		return err 
	}

	log.Debugf("action: sending_batch_data | result: in progress | batch_number: %v",
		batchNumber,
	)
	
	err = SendAll(batchData, sock)
	if err != nil { 
		return fmt.Errorf("error: failed to send batch %v: %v", 
			batchNumber,
			err,
		)
	}

	log.Debugf("action: sending_batch_data | result: success | batch_number: %v",
		batchNumber,
	)

	return nil
}


func (p *Protocol) SendFinishedTransmision(sock net.Conn) error {
	return p.SendMessage(byte(FinishedTransmision), []byte{},sock)
}

func (p *Protocol) sendBatchStart(betsInBatch uint8, sock net.Conn) error {
	return p.SendMessage(byte(BatchStart), []byte{byte(betsInBatch)}, sock)
}

func (p *Protocol) SendGetLotteryResults(agencyNumber string, sock net.Conn) error {
	agencyNumberInt, err := strconv.Atoi(agencyNumber)
	if err != nil {
		return fmt.Errorf("couldn't convert agency number: %w", err)
	}

	return p.SendMessage(byte(GetLotteryResults), []byte{byte(agencyNumberInt)}, sock)
}


func (p* Protocol) FormatBet(agencyNumber string, bet Bet) []byte {
	message := agencyNumber + separator + 
		bet.Name + separator + 
		bet.Surname + separator + 
		bet.IdentityCard + separator + 
		bet.BirthDate + separator + 
		bet.Number

	var data_to_send []byte
	data_to_send = AppendStringWithItsLength(message, data_to_send)

	return data_to_send
}