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
	protocol Protocol 
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig, protocol Protocol) *Client {
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

	err = c.protocol.SendFinishedTransmision(c.conn)
	if err != nil {
		return fmt.Errorf("error: failed to send finished transmision message code: %v", err)
	}

	log.Infof("action: finished_transmision | result: success")

	winners, err := c.getLotteryResults()
	if err != nil {
		return fmt.Errorf("error: couldn't get lottery results: %w", err)
	}

	log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %v",
		len(winners),
	)

	return nil
}

func (c *Client) getLotteryResults() ([]string, error){
		err := c.protocol.SendGetLotteryResults(c.config.ID, c.conn)
		if err != nil {
			return []string{}, err
		}
		
		log.Infof("action: awaiting_for_lottery_winners | status: in_progress")
		message, err := c.protocol.ReadMessageType(c.conn)
		if err != nil {
			return []string{}, err 
		}

		
		if message == LotteryWinners {
			winners, err := c.protocol.GetLotteryWinners(c.conn)
			if err != nil {
				return []string{}, err
			}
			
			log.Infof("action: awaiting_for_lottery_winners | status: success | code: %v",
				message,
			)
			return winners, nil
		} 
		
		log.Criticalf("action: recived_lottery_winners_result | result: fail | reason: unkown message code | code: %v",
			message,	
		)		

		return []string{}, nil
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

		// Pasarlo al protocolo
		formatedBet := c.protocol.FormatBet(c.config.ID, bet)
		
		needToSendBatch := betsInCurrentBatch == c.config.MaxBatchSize || 
		   	len(dataToSend) + len(formatedBet) > MaxBatchByteSize

		if needToSendBatch {
			err := c.protocol.SendBatch(betsInCurrentBatch, currentBatchNumber, dataToSend, c.conn)
			if err != nil {
				return err 
			}

			log.Infof("action: awaiting_server-response | result: in_progress")
	
			response, err := c.protocol.ReadMessageType(c.conn)
			if err != nil {
				return fmt.Errorf("error: failed awaiting for server response for batch %v: %v", 
					currentBatchNumber,
					err,
				)
			}
		
			logServerResponse(response)
			
			betsInCurrentBatch = 0
			currentBatchNumber += 1
			dataToSend = dataToSend[:0]
		}
		
		dataToSend = append(dataToSend, formatedBet...)
		betsInCurrentBatch += 1	
	}

	// Send the last batch 
	if len(dataToSend) != 0 {
		err := c.protocol.SendBatch(betsInCurrentBatch, currentBatchNumber, dataToSend, c.conn)
		if err != nil {
			return err 
		}

		response, err := c.protocol.ReadMessageType(c.conn)
		if err != nil {
			return fmt.Errorf("error: failed awaiting for server response for batch %v: %v", 
				currentBatchNumber,
				err,
			)
		}
	
		logServerResponse(response)
	}
	
	return nil
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