"""
Message codes
"""

"""Client side"""
BATCH_START = 0
FINISHED_TRANSMISION = 1

"""Server side"""
SUCCESS = 0
ERROR = 1


"""
Protocol length constants
"""

MESSAGE_HEADER_LENGTH = 1
"""Amount of bytes used to indicate message type"""

BET_LENGTH_BYTES = 4
"""Amount of bytes used for describing the length of a determine bet in a batch"""

BATCH_LENGTH_BYTES = 1
"""Amount of bytes used for indicating batch length"""

EXPECTED_BET_FIELDS = 6
"""Amount of expected fields to be in a message"""

SEPARATOR = '#'
"""Separator used in the protocol"""
