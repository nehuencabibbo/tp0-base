package common

import (
	"encoding/csv"
	"fmt"
	"os"
)

const separator = "#"

// Bet represts a bet made by a specific client
type Bet struct {
	Name string
	Surname string
	IdentityCard string
	BirthDate string
	Number string
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
            Name:         record[0],
            Surname:      record[1],
            IdentityCard: record[2],
            BirthDate:    record[3],
            Number:       record[4],
        })
    }

    return bets, nil
}