import logging
import os
import signal
import socket

from protocol.protocol_error import ProtocolError
from protocol.constants import *
from common import utils
from typing import *

SOCKET_TIMEOUT = 30.0
"""Time after not reciving any message which the socket is automatically closed"""

# Abstracts the handling of client connections, simply doing:
# process = Process(target=self.__handle_client_connection, args=(client_socket, ))
# is not good enough, as many unnecesary resources from self are copyed. 
# More specifically, server_socket is copyed, which can lead to issues
class ClientHandler():
    # Just the needed shared resources are passed 
    def __init__(self, client_sock, protocol, agencys_that_finished_sending_bets, winners, locks, agencys, can_start_lottery, winners_were_set):
        self._recived_sigterm = False 
        self._sock = client_sock
        self._protocol = protocol
        self._agencys = agencys 

        # Shared memory -> TENER UN DICCIONARIO DE LOCKS 
        self._locks = locks # Dict -> <shared_resource_name> (str): Lock -> !For store_bets and load_bets file_lock is the name
        self._agencys_that_finished_sending_bets = agencys_that_finished_sending_bets # Value (int)
        self._winners = winners # Dict -> agency (str): winners (list(str))
        self._winners_were_set = winners_were_set # Value bool
        self._can_start_lottery = can_start_lottery #This is an event initially set to false
        
        signal.signal(signal.SIGTERM, self.__sigterm_handler)


    def __sigterm_handler(self, signum, frame):
        # PID Printed just to check actual child process is running
        logging.info(f'action: recived_sigterm | in: child_process | pid: {os.getpid()}') 
        self._recived_sigterm = True


    def start(self):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """

        try:
            # If no messages are recived after a SOCKET_TIMEOUT passes, communication is considered
            # finished
            self._sock.settimeout(SOCKET_TIMEOUT)
            while not self._recived_sigterm:
                message_type = self._protocol.read_message_type(self._sock)
                logging.debug(f"action: reading_message_type | message_type: {message_type}")
                message = self._protocol.read_message(message_type, self._sock)
                if message_type == BATCH_START:
                    (bets, rejected) = message
                    with self._locks["file_lock"]:
                        utils.store_bets(bets)

                    if rejected == 0:
                        logging.info(f"action: apuesta_recibida | result: success | cantidad: {len(bets)}")
                        self._protocol.send_response(SUCCESS, self._sock)

                    else: 
                        logging.info(f"action: apuesta_recibida | result: failure | cantidad: {rejected}")
                        self._protocol.send_response(ERROR, self._sock)

                        # If any batch has defects, communication is terminated
                        break

                # When transmission is finished, agency is counter towards needed agencys to
                # start the lottery
                elif message_type == FINISHED_TRANSMISION:
                    logging.debug((f"action: finished_sending_batches | agency: {agency} | " 
                                f"currently_requested_winners {self._agencys_that_requested_winners}"))
                    
                    with self._locks["agencys_that_finished_sending_bets"]:
                        self._agencys_that_finished_sending_bets.value += 1
                        
                        # Can give out lottery results 
                        if len(self._agencys_that_finished_sending_bets) == self._agencys:
                            self._can_start_lottery.set()

                elif message_type == GET_LOTTERY_RESULTS:
                    # 1 - Espero a que todas las agencias hayan mandado FINISHED TRANSMISION
                    # 2 - Cuando todas las agencias mandan finished transmision tomo el lock de winners
                    #     -> Si el lock de winners estaba en None, empiezo la loteria (lo setteo)
                    #     -> Si el lock de winners no estaba en None, tomo el resultado 
                    agency = message
                    logging.debug(f"action: processing_get_lottery_results | agency: {agency}")

                    # Wait till the lottery can start
                    self._can_start_lottery.wait()

                    with self._locks["winners"]:
                        # Solo se cargan una vez los ganadores 
                        if not self._winners_were_set.value:
                            # No hay ganadores 
                            with self._locks["file_lock"]:
                                logging.info(f"action: sorteo | result: success")

                                self.__start_lottery()
                                
                                # Sigterm was recived during the process
                                if not self._winners_were_set.value:
                                    break
                        
                        self._protocol.send_response(LOTTERY_WINNERS, self._sock, self._winners.copy(), agency)
                        logging.debug(f"action: sent_lottery_winners | agency: {agency}")
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
            self._sock.close()
            logging.info('action: closing_client_socket | result: in_progress | reason: conection finished')

    def __start_lottery(self) -> Dict[str, str]: 
        bets = utils.load_bets()

        winners = {} # agency(str): winnerDocument (str)
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
        self._winners_were_set.value = True
