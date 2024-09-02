package common

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/op/go-logging"
)

const BASE_FILE_NAME = "./data/agency-"
const MAX_BATCH_BYTE_SIZE = 8000

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
	maxBatchSize  int
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

	err := c.createConnection()
	if err != nil {
		return fmt.Errorf("error starting the client: %w", err)
	}

	err = c.sendBatchOfBets()
	if err != nil {
		c.conn.Close()
		return fmt.Errorf("error: couldn't send the bet %v", err)
	}

	if c.recivedSigterm {
		c.conn.Close()
		log.Infof("action: closing_client_socket | result: success | reason: recived_sigterm")
		
		return nil
	}

	msg, err := c.readServerResponse('#')
	if err != nil {
		return err
	}

	logServerResponse(msg)
	
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

func (c *Client) sendBatchOfBets() error {
	file_name := fmt.Sprintf("%s%s.csv", BASE_FILE_NAME, c.config.ID)
	file, err := os.Open(file_name)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ','

	line_number := 1
	bets_in_current_batch := 0
	var batch []byte
	for {
		// Each time a line is processed, check if SIGTERM was sent
		// just breaking the loop closes the file immediately
		if c.recivedSigterm { break }
		line, err := reader.Read()
		if err != nil {
			if err.Error() == "EOF" { 
				// If there are bets remaining, send them
				if len(batch) != 0 {
					err := SendAll(batch, c.conn)
					if err != nil {
						return fmt.Errorf("error while sending batch: %v", err)
					}
				}

				break 
			}

			return fmt.Errorf("error while reading line: %v", err)
		}

		// If there's and invalid line, stop sending
		if len(line) != 5 {
			return fmt.Errorf("error: invalid line in csv %s in line %d", 
				file_name, 
				line_number,
			)
		}

		bet := Bet {
			name: line[0], 
			surname: line[1], 
			identityCard: line[2], 
			birthDate: line[3], 
			number: line[4], 
		}

		log.Infof("action: apuesta_encolada | result: success | dni: %v | numero: %v",
			bet.identityCard,
			bet.number,
		)

		formatedBet := bet.FormatToSend(c.config.ID)

		if len(formatedBet) + len(batch) > MAX_BATCH_BYTE_SIZE || 
			bets_in_current_batch == 10 { //TODO: Parsear del config
			err := SendAll(batch, c.conn)
			if err != nil {
				return fmt.Errorf("error while sending batch: %v", err)
			}
			log.Infof("action: batch_enviado | result: success | weight: %v | cantidad: %v",
				len(batch),
				bets_in_current_batch,
			)

			batch = batch[:0]
			bets_in_current_batch = 0
		}

		batch = append(batch, formatedBet...)

		bets_in_current_batch += 1
		line_number += 1
	}

	return nil
}

func (c* Client) readServerResponse(separator byte) (string, error) {
    reader := bufio.NewReader(c.conn)
    response, err := reader.ReadString(separator)
    if err != nil {
        return "", err
    }
	
	response = strings.TrimSuffix(response, "#")
    return response, nil
}

func logServerResponse(msg string) {
	if msg == "0" {
		log.Infof("action: recived_server_confirmation | result: success | status code: %v",
		msg,
		)
	} else {
		log.Infof("action: recived_server_confirmation | result: failure | status code: %v",
		msg,
		)
	}
}