"""
Message codes
"""

"""Client side"""
BATCH_START = 0
FINISHED_TRANSMISION = 1
GET_LOTTERY_RESULTS = 2

"""Server side"""
SUCCESS = 0
ERROR = 1
CANT_GIVE_LOTTERY_RESULTS = 2
LOTTERY_WINNERS = 3


"""
Protocol length constants
"""

MESSAGE_HEADER_LENGTH = 1
"""Amount of bytes used to indicate message type"""

DOCUMENT_BYTES = 4 
"""Amount of bytes a document has"""

WINNERS_LENGTH_BYTES = 4
"""Amount of bytes used to send winners documents length"""

BET_LENGTH_BYTES = 4
"""Amount of bytes used for describing the length of a determine bet in a batch"""

BATCH_LENGTH_BYTES = 1
"""Amount of bytes used for indicating batch length"""

EXPECTED_BET_FIELDS = 6
"""Amount of expected fields to be in a message"""

AGENCY_LENGTH_BYTES = 1
"""Amount of bytes used for indicating agency number length"""

SEPARATOR = '#'
"""Separator used in the protocol"""
