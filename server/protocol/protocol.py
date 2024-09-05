from typing import *
from protocol.constants import * 
from protocol import utils
from protocol.protocol_error import ProtocolError
from common.utils import Bet


class Protocol():
    def __init__(self):
        pass

    def send_response(self, message_type, sock):
        response_bytes = message_type.to_bytes(MESSAGE_HEADER_LENGTH, 'big', signed=False)

        sock.sendall(response_bytes)

    def read_message(self, message_type, sock) -> Any:
        if message_type == BATCH_START:
            (bets, rejected) = self.__read_batch(sock)

            return (bets, rejected)
        
        elif message_type == FINISHED_TRANSMISION:

            return None
        
        else:
            raise ProtocolError("Unkown message")
    
    
    def __read_batch(self, sock) -> Tuple[list[Bet], int]:
        """
        Reads an entire batch of bets according to the described protocol.
        If there's a problem with some of the bets, the batch is read in it's
        entirety either way, and the amount of defective batches are returned
        along side the correctly parsed bets
        """
        bets_to_read = int.from_bytes(utils.read_all(sock, BATCH_LENGTH_BYTES), byteorder='big')

        bets = []
        rejected = 0
        while bets_to_read > 0:
            try:
                bet = self.__read_bet(sock)
                bets.append(bet)
            except ProtocolError:
                rejected += 1

            bets_to_read -= 1
                
        return (bets, rejected)
    
    @staticmethod
    def __read_bet(sock) -> Bet:
        """
        Reads a bet from the socket according to the described protocol.
        Ensures no short read happen.
        If the socket closes during the process then None is returned.
        """
        # It reads the first four bytes to know the length of the entire bet
        # then it proceds to read the bet and return it 
        need_to_read = int.from_bytes(utils.read_all(sock, BET_LENGTH_BYTES), byteorder='big')

        message: list[str] = utils.read_all(sock, need_to_read).decode('utf-8').split(SEPARATOR)

        if len(message) != EXPECTED_BET_FIELDS:
            raise ProtocolError((
                f"error: Missing fields, need 6, but {len(message)} were given. "
                f"The following was read: {message}"
                ))
        
        bet = Bet(
            message[0],
            message[1],
            message[2],
            message[3],
            message[4],
            message[5],
        )

        return bet


    @staticmethod
    def read_message_type(sock):
        """
        Reads the header according to the described protocol.
        The header indicates the type of message that is about to be sent
        and it occupies exactly one byte
        """
        header = utils.read_all(sock, MESSAGE_HEADER_LENGTH)
        return int.from_bytes(header, byteorder='big')
