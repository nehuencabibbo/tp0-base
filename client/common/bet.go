package common

import (
	"encoding/csv"
	"fmt"
	"strings"
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

// Recives a string that represents a bet with it's fields encoded as a csv 
// separated by ,. Returns the corresponding Bet if possible
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