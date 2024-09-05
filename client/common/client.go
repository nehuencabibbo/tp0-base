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

const ServerSeparator = '#'

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
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

	bet := getBetFromEnvVars()
	err = c.sendBet(bet)
	if err != nil {
		c.conn.Close()
		return fmt.Errorf("error: couldn't send the bet %v", err)
	}

	log.Infof("action: apuesta_enviada | result: success | dni: %v | numero: %v",
		bet.identityCard,
		bet.number,
	)

	if c.recivedSigterm {
		c.conn.Close()
		log.Infof("action: closing_client_socket | result: success | reason: recived_sigterm")
		
		return nil
	}

	msg, err := c.readServerResponse(ServerSeparator)
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

func (c* Client) readServerResponse(separator byte) (string, error) {
    reader := bufio.NewReader(c.conn)
    response, err := reader.ReadString(separator)
    if err != nil {
        return "", err
    }
	
	response = strings.TrimSuffix(response, string(separator))
    return response, nil
}

func logServerResponse(msg string) {
	if msg == "success" {
		log.Infof("action: recived_server_confirmation | result: success")
	} else {
		log.Infof("action: recived_server_confirmation | result: failure")
	}
}