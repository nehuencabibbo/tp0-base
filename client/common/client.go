package common

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/op/go-logging"
)

// General constants
const (
	BaseFileName = "./data/agency-"
	MaxBatchByteSize = 8000
)

// Client side porotocol constants"""
const (
	BatchStart = 0
	FinishedTransmision = 1
	GetLotteryResults = 2
) 

// Server side porotocol constants
const (
	ServerSeparator = '#'
	Success = 0
	Error = 1
	CantGiveLotteryResults = 2
	LotteryWinners = 3
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
	MaxBatchSize  int
}

// Client Entity that encapsulates how
type Client struct {
	config ClientConfig
	conn net.Conn
	recivedSigterm bool
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
		recivedSigterm: false,
	}
	return client
}

func (c *Client) createConnection() error {
	socket, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)

		return fmt.Errorf("error creating socket: %w", err)
	}

	c.conn = socket

	return nil
}

func (c *Client) StartClientLoop() error {
	// https://gobyexample.com/signals
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM)

	go func() {
		<- ch
		c.recivedSigterm = true
		log.Infof("action: recieved_sigterm")
	}()

	defer c.shutdown()

	err := c.createConnection()
	if err != nil {
		return fmt.Errorf("error starting the client: %w", err)
	}

	file_name := fmt.Sprintf("%s%s.csv", BaseFileName, c.config.ID)
	bets, err := getBetsFromCsv(file_name)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	err = c.sendBatchOfBets(bets)
	if err != nil {
		return fmt.Errorf("error: couldn't send the bet %v", err)
	}

	err = c.sendFinishedTransmision()
	if err != nil {
		return fmt.Errorf("error: failed to send finished transmision message code: %v", err)
	}

	log.Infof("action: finished_transmision | result: success")

	err = c.getLotteryResults()
	if err != nil {
		return fmt.Errorf("error: couldn't get lottery results: %w", err)
	}
	
	return nil
}

func (c *Client) getLotteryResults() ([]int, error){
	// 1 - Mandar el mensaje de get lottery results 
	// 2 - Esperar a que el servidor me responda
	// 3 - Si el servidor me da los winners, parsearlos, sino volver a 1
	for {
		err := c.sendGetLotteryResults()
		if err != nil {
			return []int{}, err
		}

		message, err := c.readServerResponse()
		if err != nil {
			return []int{}, err 
		}

		if message == CantGiveLotteryResults {
			time.Sleep(1 * time.Second)
			continue
		} else if message == LotteryWinners {
			winners, err := c.getLotteryWinners()
			if err != nil {
				return winners, err
			}

			return winners, nil
		}
	}
}

func (c *Client) getLotteryWinners() ([]int32, error){
	// Leer 4 bytes para saber cuanto tengo que leer 
	// Convertirlo a entero
	// Leer eso
	// cada cuatro bytes ir convirtiendo el numero

	needToRead, err := ReadAll(c.conn, 4)
	if err != nil {
		return []int{}, err 
	}

	needToRead = binary.BigEndian.Uint32(needToRead)
	
	winners := ReadAll(c.conn, needToRead)
}

// sendFinishedTransmision sends the FinishedTransmision message. Following the 
// described protocol
func (c *Client) sendGetLotteryResults() error {
	var data []byte
	data = append(data, byte(GetLotteryResults))

    err := SendAll(data, c.conn)
    if err != nil {
        return fmt.Errorf("error sending get lottery results header: %w", err)
    }

    return nil
}

// shotdown Closes client resources before shuting down
func (c* Client) shutdown() {
	if c.conn != nil {
		c.conn.Close()
		log.Infof("action: closing_socket | result: success")
	}
}

// sendBatchOfBets recives a slice of bets and sends them to the server.
// After each batch, it awaits for the server response, if the response is
// negative or if there's any error realting sockets, sending is completly
// stopped and the corresponding error is returned.
// Each batch has at most maxBatchSize bets (declared in config).
// If the batch is bigger than 8kb with maxBatchSize 
// then as much bets as it's possible are sent so that each batch weights
// at maximum 8kb
func (c *Client) sendBatchOfBets(batches []Bet) error {
	var dataToSend []byte
	betsInCurrentBatch := 0
	currentBatchNumber := 1
	for _, bet := range batches { 
		// Before appending each bet to the current batch, sigterm signal
		// is checked for
		if (c.recivedSigterm) { 
			log.Infof("action: recived_sigterm | during: sending_batches")
			break 
		}

		formatedBet := bet.FormatToSend(c.config.ID)
		
		needToSendBatch := betsInCurrentBatch == c.config.MaxBatchSize || 
		   	len(dataToSend) + len(formatedBet) > MaxBatchByteSize

		if needToSendBatch {
			err := c.sendBatch(betsInCurrentBatch, currentBatchNumber, dataToSend)
			if err != nil {
				return err 
			}
			
			betsInCurrentBatch = 0
			currentBatchNumber += 1
			dataToSend = dataToSend[:0]
		}
		
		dataToSend = append(dataToSend, formatedBet...)
		betsInCurrentBatch += 1	
	}

	// Send the last batch 
	if len(dataToSend) != 0 {
		err := c.sendBatch(betsInCurrentBatch, currentBatchNumber, dataToSend)
		if err != nil {
			return err 
		}
	}
	
	return nil
}

// sendBatch Sends a batch according to the described protocol and logs every step during
// the process
func (c *Client) sendBatch(betsInBatch int, batchNumber int, dataToSend []byte) error {
	log.Infof("action: sending_batch_start | result: in_progress ")
	err := c.sendBatchStart(uint8(betsInBatch))
	if err != nil { 
		return fmt.Errorf("error: failed to send batch start header: %v", 
			err,
		)
	}
	log.Infof("action: sending_batch_start | result: success ")

	log.Infof("action: sending_batch_data | result: in progress | batch_number: %v",
		batchNumber,
	)

	
	err = SendAll(dataToSend, c.conn)
	if err != nil { 
		return fmt.Errorf("error: failed to send batch %v: %v", 
			batchNumber,
			err,
		)
	}

	log.Infof("action: sending_batch_data | result: success | batch_number: %v",
		batchNumber,
	)
	
	log.Infof("action: awaiting_server-response | result: in_progress")
	
	response, err := c.readServerResponse()
	if err != nil {
		return fmt.Errorf("error: failed to send batch %v: %v", 
			batchNumber,
			err,
		)
	}

	logServerResponse(response)

	return nil
}

// readServerResponse Reads the server response to sending a batch of bets
// following the described protocol
func (c* Client) readServerResponse() (int, error) {
	response, err := ReadAll(c.conn, 1)
    if err != nil {
        return -1, err
    }

	intResponse := int(response[0])
	
    return intResponse, nil
}

// logServerResponse logs the server response. If status code is unknown it's logged 
// as unkown
func logServerResponse(code int) {
	if code == Success {
		log.Infof("action: recived_server_confirmation | result: success | status code: %d",
			code,
		)
	} else if code == Error {
		log.Infof("action: recived_server_confirmation | result: failure | status code: %d",
			code,
		)
	} else {
		log.Infof("action: recived_server_confirmation | result: unkown_status_code | status code: %d",
			code,
		)
	}	
}

// sendBatchStart sends the BatchStart message concatenated with the amount
// of bets in the batch to read as a u8. Following the described protocol
func (c *Client) sendBatchStart(betsInCurrentBatch uint8) error {
	var data []byte
	data = append(data, byte(BatchStart))
	data = append(data, byte(betsInCurrentBatch))

    err := SendAll(data, c.conn)
    if err != nil {
        return fmt.Errorf("error sending batch start message: %w", err)
    }

    return nil
}

// sendFinishedTransmision sends the FinishedTransmision message. Following the 
// described protocol
func (c *Client) sendFinishedTransmision() error {
	var data []byte
	data = append(data, byte(FinishedTransmision))

    err := SendAll(data, c.conn)
    if err != nil {
        return fmt.Errorf("error sending batch start message: %w", err)
    }

    return nil
}