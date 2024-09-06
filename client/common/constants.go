package common 

// Client side message codes 
const (
	BatchStart = 0
	FinishedTransmision = 1
	GetLotteryResults = 2
) 

// Server side message codes
const (
	Success = 0
	Error = 1
	CantGiveLotteryResults = 2
	LotteryWinners = 3
)

// Message lengths
const (
	// Bytes that a document (fixed size) occupies
	DocumentBytes = 4 
	// Bytes used for indicating the length of the 
	// winners being sent
	WinnersLengthBytes = 4
)

