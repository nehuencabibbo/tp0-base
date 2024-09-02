package common

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/op/go-logging"
)

const (
	ServerSeparator = '#'
	BaseFileName = "./data/agency-"
	MaxBatchByteSize = 8000
	BatchStart = 0
	FinishedTransmision = 1
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

func getBetFromEnvVars() *Bet {
	name := os.Getenv("NOMBRE")
	surname := os.Getenv("APELLIDO")
	identityCard := os.Getenv("DOCUMENTO")
	birthDate := os.Getenv("NACIMIENTO")
	number := os.Getenv("NUMERO")

	bet := &Bet{
		name, 
		surname, 
		identityCard, 
		birthDate, 
		number,
	}

	return bet
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
	
	return nil
}

func (c *Client) sendBet(bet *Bet) error {
	formatedBet := bet.FormatToSend(c.config.ID)
	err := SendAll(formatedBet, c.conn)
	if err != nil {
		return err
	}

	return nil
}

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
	i := 0
	betsInCurrentBatch := 0
	currentBatchNumber := 1
	for { 
		// Before appending each bet to the current batch, sigterm signal
		// is checked for
		if (c.recivedSigterm) { break }

		formatedBet := batches[i].FormatToSend(c.config.ID)

		if betsInCurrentBatch == c.config.MaxBatchSize || 
		   len(dataToSend) + len(formatedBet) > MaxBatchByteSize {
				log.Infof("action: sending_batch_start | result: in_progress ")
			   	err := c.sendBatchStart(uint8(betsInCurrentBatch))
			   	if err != nil { 
				   	return fmt.Errorf("error: failed to send batch start header: %v", 
				   		err,
					)
				}
			log.Infof("action: sending_batch_start | result: success ")

			log.Infof("action: sending_batch_data | result: in progress")

			// log.Debugf("Data being sent: %v", dataToSend)

			err = SendAll(dataToSend, c.conn)
			if err != nil { 
				return fmt.Errorf("error: failed to send batch %v: %v", 
					currentBatchNumber,
					err,
				)
			}

			log.Infof("action: sending_batch_data | result: success")

			betsInCurrentBatch = 0
			
			log.Infof("action: awaiting_server-response | result: in_progress")

			response, err := c.readServerResponse(ServerSeparator)
			if err != nil {
				return fmt.Errorf("error: failed to send batch %v: %v", 
					currentBatchNumber,
					err,
				)
			}

			fmt.Print("aca no llega ")

			currentBatchNumber += 1
			logServerResponse(response)
		}
		
		dataToSend = append(dataToSend, formatedBet...)
		
		betsInCurrentBatch += 1
		i += 1

		time.Sleep(1 * time.Second) 
	}

	return nil
}

func (c* Client) readServerResponse(separator byte) (string, error) {
    reader := bufio.NewReader(c.conn)
    response, err := reader.ReadString(separator)
    if err != nil {
        return "", err
    }
	
	response = strings.TrimSuffix(response, string(ServerSeparator))
    return response, nil
}

func logServerResponse(msg string) {
	if msg == "success" {
		log.Infof("action: recived_server_confirmation | result: success | status code: %v",
		msg,
		)
	} else {
		log.Infof("action: recived_server_confirmation | result: failure | status code: %v",
		msg,
		)
	}
}

// sendBatchStart sends the BatchStart message concatenated with the amount
// of bets in the batch to read as a u8 .
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

// sendBatchStart sends the BatchStart message concatenated with the amount
// of bets in the batch to read as a u8 .
func (c *Client) sendFinishedTransmision() error {
	var data []byte
	data = append(data, byte(FinishedTransmision))

    err := SendAll(data, c.conn)
    if err != nil {
        return fmt.Errorf("error sending batch start message: %w", err)
    }

    return nil
}