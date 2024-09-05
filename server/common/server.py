import socket
import logging
import signal
from common import utils
from typing import *
from protocol.protocol import Protocol
from protocol.protocol_error import ProtocolError

"""Server side"""
from protocol.constants import SUCCESS, ERROR

"""Client side"""
from protocol.constants import BATCH_START, FINISHED_TRANSMISION

AGENCYS = 5
"""Amount of supported agencys"""

SOCKET_TIMEOUT = 30.0
"""Time after not reciving any message which the socket is automatically closed"""

class Server:
    def __init__(self, port, listen_backlog, protocol: Protocol):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._recived_sigterm = False
        self._protocol = protocol
        self._has_already_started_lottery = False

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
        try:
            # If no messages are recived after a SOCKET_TIMEOUT passes, communication is considered
            # finished
            sock.settimeout(SOCKET_TIMEOUT)
            while True:
                message_type = self._protocol.read_message_type(sock)
                logging.debug(f"action: reading_message_type | message_type: {message_type}")
                message = self._protocol.read_message(message_type, sock)
                if message_type == BATCH_START:
                    (bets, rejected) = message
                    utils.store_bets(bets)

                    if rejected == 0:
                        logging.info(f"action: apuesta_recibida | result: success | cantidad: {len(bets)}")
                        self._protocol.send_response(SUCCESS, sock)

                    else: 
                        logging.info(f"action: apuesta_recibida | result: failure | cantidad: {rejected}")
                        self._protocol.send_response(ERROR, sock)

                        # If any batch has defects, communication is terminated
                        break

                elif message_type == FINISHED_TRANSMISION:
                    logging.debug(f"action: transmision_terminated | result: success")

                    break

                else:
                    logging.critical(f"Unhandled message type: {message_type}")
                    break
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
