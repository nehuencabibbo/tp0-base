import socket
import logging
import signal
from common import utils

"""Separator used in the protocol"""
SEPARATOR = '#'
"""Amount of bytes used for describing the length of a determined bet"""
MESSAGE_LENGTH_BYTES = 4
"""Amount of expected fields to be in a message"""
EXPECTED_BET_FIELDS = 6
"""Time after which a connection is considered finished"""
SOCKET_TIMEOUT = 1.0

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
        bets_recived = 0
        try:
            # If no messages are recived after a second passes, communication is considered
            # finished
            sock.settimeout(SOCKET_TIMEOUT)
            while True:
                bet = self.__read_bet(sock)
                utils.store_bets([bet])
                bets_recived += 1
                logging.info(f"action: apuesta_almacenada | result: success | dni: {bet.document} | numero: {bet.number}")
        except socket.timeout as e: 
            success = True
            logging.info(f"action: finished_sending_batches | result: success | via: {e}")
        except OSError as e:
            logging.error(f"action: receive_message | result: fail | error: {e}")
        except ProtocolError as e:
            logging.error(f"action: parsing_bet | result: fail | error: {e}")


        try:
            response = "0" if success else "1"
            self.__send_response(sock, response)
        except OSError as e:
            logging.error(f"action: sending_response | result: fail | error: {e}")
        finally:
            sock.close()

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
    def __read_bet(socket) -> utils.Bet:
        """
        Reads a bet from the socket according to the described protocol.
        Ensures no short read happen.
        If the socket closes during the process then None is returned.
        """
        # It reads the first four bytes to know the length of the entire bet
        # then it proceds to read the bet and return it 

        need_to_read = int.from_bytes(utils.read_all(socket, MESSAGE_LENGTH_BYTES), byteorder='big')

        message: list[str] = utils.read_all(socket, need_to_read).decode('utf-8').split(SEPARATOR)

        if len(message) != EXPECTED_BET_FIELDS:
            raise ProtocolError((
                f"error: Missing fields, need 6, but {len(message)} were given. "
                f"The following was read: {message}"
                ))
        
        return utils.Bet(
            message[0],
            message[1],
            message[2],
            message[3],
            message[4],
            message[5],
        )
    
    @staticmethod
    def __send_response(socket, response: str):
        """
        Sends server response to the client following the described protocol.
        Ensures no short writes happen
        """
        response += SEPARATOR
        message = response.encode('utf-8')
        socket.sendall(message)