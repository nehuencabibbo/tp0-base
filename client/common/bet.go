package common

import (
	"encoding/csv"
	"fmt"
	"os"
)

const separator = "#"

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
	message := agencyNumber + separator + 
		b.name + separator + 
		b.surname + separator + 
		b.identityCard + separator + 
		b.birthDate + separator + 
		b.number
	
    var data_to_send []byte
	data_to_send = AppendStringWithItsLength(message, data_to_send)

    return data_to_send
}

// getBetsFromCsv parses a csv file containing bet fields separated by ','.
// If any line is not correctly formated, an error is returned, else
// a slice containing Bets.
func getBetsFromCsv(path string) ([]Bet, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, fmt.Errorf("error: could not open file: %v", err)
    }
    defer file.Close()

    reader := csv.NewReader(file)
	reader.Comma = ','

    records, err := reader.ReadAll()
    if err != nil {
        return nil, fmt.Errorf("error: could not read CSV file: %v", err)
    }

    var bets []Bet
    for line_number, record := range records {
        if len(record) != 5 {
			err := fmt.Errorf("error: Line %d has an incorrect amount of arguments, need 5, was given %d",
				line_number,
				len(records),
			)

            return nil, err
        }

        bets = append(bets, Bet{
            name:         record[0],
            surname:      record[1],
            identityCard: record[2],
            birthDate:    record[3],
            number:       record[4],
        })
    }

    return bets, nil
}