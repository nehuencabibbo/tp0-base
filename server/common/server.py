import socket
import logging
import signal
from common import utils
from typing import *

"""Amount of bytes used for indicating batch length"""
BATCH_LENGTH_BYTES = 1
"""Amount of bytes used for protocol related messages"""
MESSAGE_HEADER_LENGTH = 1
"""Separator used in the protocol"""
SEPARATOR = '#'
"""Amount of bytes used for describing the length of a determined bet"""
MESSAGE_LENGTH_BYTES = 4
"""Amount of expected fields to be in a message"""
EXPECTED_BET_FIELDS = 6
"""Time after which a connection is considered finished"""
SOCKET_TIMEOUT = 60.0
"""Amount of bytes used for indicating agency number length"""
AGENCY_LENGTH_BYTES = 1
"""Amount of agencys that are involved in the protocol"""
AGENCYS = 5
"""Document bytes"""
DOCUMENT_BYTES = 4 
"""Amount of bytes used to send winners length"""
WINNERS_LENGTH_BYTES = 4

"""Protocol Message codes"""

"""Client side"""
BATCH_START = 0
FINISHED_TRANSMISION = 1
GET_LOTTERY_RESULTS = 2

"""Server side"""
SUCCESS = 0
ERROR = 1
CANT_GIVE_LOTTERY_RESULTS = 2
LOTTERY_WINNERS = 3

"""Custom exception for communication protocol related issues """
class ProtocolError(Exception):
    pass 

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._recived_sigterm = False
        self._agencys = AGENCYS
        self._agencysRequestingWinners = set()
        self._winners = None

        signal.signal(signal.SIGTERM, self.__sigterm_handler)

    def __sigterm_handler(self, signum, frame):
        self._recived_sigterm = True

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """

        # TODO: Modify this program to handle signal to graceful shutdown
        # the server
        while not self._recived_sigterm:
            client_socket = self.__accept_new_connection()
            if self._recived_sigterm:
                client_socket.close()
                logging.info('action: closing_client_socket | result: in_progress | reason: recived_sigterm')

                break

            self.__handle_client_connection(client_socket)

        self._server_socket.close()
        logging.info('action: closing_server_socket | result: success | reason: recived_sigterm')

    def __handle_client_connection(self, sock):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        # Used so no nested try catch blocks are needed
        success = False 
        try:
            # If no messages are recived after a SOCKET_TIMEOUT passes, communication is considered
            # finished
            sock.settimeout(SOCKET_TIMEOUT)
            while True:
                header = self.__read_header(sock)
                logging.info(f"action: reading_header | header: {header}")
                if header == BATCH_START:
                    (bets, rejected) = self.__read_batch(sock)
                    utils.store_bets(bets)
                    if rejected == 0:
                        logging.info(f"action: apuesta_recibida | result: success | cantidad: {len(bets)}")
                        self.__send_response(sock, SUCCESS)
                    else: 
                        logging.info(f"action: apuesta_recibida | result: failure | cantidad: {rejected}")
                        self.__send_response(sock, ERROR)
                        # If any batch has defects, communication is terminated with the client
                        break
                elif header == FINISHED_TRANSMISION:
                    logging.info(f"action: transmision_terminated | result: success")
                    break
                elif header == GET_LOTTERY_RESULTS:
                    agency = self.__read_agency(sock)
                    logging.info(f"action: processing_get_lottery_results | agency: {agency}")

                    self._agencysRequestingWinners.add(agency)
                    logging.info((f"action: recived_lottery_winners_request | agency: {agency} | " 
                                 f"currently_requested_winners {len(self._agencysRequestingWinners)}"))
                    
                    if len(self._agencysRequestingWinners) == self._agencys:
                        logging.info(f"action: sorteo | result: success")
                        self.__start_lottery()
                        # Recived sigterm during the process 
                        if self._winners == None:
                            break

                        self.__send_lottery_results(agency, sock)
                        break
                    else:
                        # If the lottery is not ready, disconnect the client and make it try again
                        self.__send_cant_give_lottery_results(sock)
                        break
                else:
                    raise ProtocolError(f"error: Unkown message: {header}")
        except socket.timeout as e: 
            logging.info(f"action: reciving_message | result: fail | via: {e}")
        except OSError as e:
            logging.error(f"action: reciving_message | result: fail | via: {e}")
        except ProtocolError as e:
            logging.error(f"action: reciving_message | result: fail | via: {e}")
        finally:
            sock.close()
            logging.info('action: closing_client_socket | result: in_progress | reason: conection finished')

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c

    @staticmethod
    def __read_bet(sock) -> utils.Bet:
        """
        Reads a bet from the socket according to the described protocol.
        Ensures no short read happen.
        If the socket closes during the process then None is returned.
        """
        # It reads the first four bytes to know the length of the entire bet
        # then it proceds to read the bet and return it 
        need_to_read = int.from_bytes(utils.read_all(sock, MESSAGE_LENGTH_BYTES), byteorder='big')

        message: list[str] = utils.read_all(sock, need_to_read).decode('utf-8').split(SEPARATOR)

        if len(message) != EXPECTED_BET_FIELDS:
            raise ProtocolError((
                f"error: Missing fields, need 6, but {len(message)} were given. "
                f"The following was read: {message}"
                ))
        
        bet = utils.Bet(
            message[0],
            message[1],
            message[2],
            message[3],
            message[4],
            message[5],
        )

        return bet

    @staticmethod
    def __read_header(sock):
        """
        Reads the header according to the described protocol.
        The header indicates the type of message that is about to be sent
        and it occupies exactly one byte
        """
        header = utils.read_all(sock, MESSAGE_HEADER_LENGTH)
        return int.from_bytes(header, byteorder='big')
    
    def __read_batch(self, sock) -> Tuple[list[utils.Bet], int]:
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
    def __send_response(sock, response: int):
        """
        Sends server response to the client following the described protocol.
        Ensures no short writes happen
        """
        response_bytes = response.to_bytes(1, 'big')
        sock.sendall(response_bytes)

    @staticmethod
    def __read_agency(sock):
        agency = int.from_bytes(utils.read_all(sock, AGENCY_LENGTH_BYTES), byteorder='big')

        return agency
    
    def __start_lottery(self) -> Dict[str, str]: 
        bets = utils.load_bets()

        winners = {} #
        for bet in bets:
            if self._recived_sigterm:
                return
            
            if utils.has_won(bet):
                document = bet.document
                agency = bet.agency

                if not agency in winners:
                    winners[agency] = list()
                    winners[agency].append(document)
                else:
                    winners[agency].append(document)

        self._winners = winners

    def __send_lottery_results(self, agency: int, sock):
        winners = self._winners.get(agency, list())

        header = LOTTERY_WINNERS.to_bytes(MESSAGE_HEADER_LENGTH, 'big', signed=False)
    
        message = b''.join([int(document).to_bytes(DOCUMENT_BYTES, 'big', signed=False) for document in winners])
        # message = b''.join([document.encode('utf-8') for document in winners])

        message_length = len(message).to_bytes(WINNERS_LENGTH_BYTES, 'big', signed=False)

        sock.sendall(header + message_length + message)

    @staticmethod
    def __send_cant_give_lottery_results(sock):
        header = CANT_GIVE_LOTTERY_RESULTS.to_bytes(MESSAGE_HEADER_LENGTH, 'big', signed=False)

        sock.sendall(header)



        