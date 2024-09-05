import os
import socket
import logging
import signal
from common import utils
from typing import *
from protocol.protocol import Protocol
from protocol.protocol_error import ProtocolError

from multiprocessing import Process, Manager, Event

from common.client_handler import ClientHandler

"""Server side"""
from protocol.constants import SUCCESS, ERROR, CANT_GIVE_LOTTERY_RESULTS, LOTTERY_WINNERS

"""Client side"""
from protocol.constants import BATCH_START, FINISHED_TRANSMISION, GET_LOTTERY_RESULTS

AGENCYS = 5
"""Amount of supported agencys"""

SOCKET_TIMEOUT = 30.0
"""Time after not reciving any message which the socket is automatically closed"""

class Server:
    def __init__(self, port, listen_backlog, protocol: Protocol):
        # NOT process safe, even though each process recives a copy of self, this is not process 
        # safe, musn't be used but is not needed
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._recived_sigterm = False
        self._agencys = AGENCYS
        self._protocol = protocol #Class

        # Concurrency related 
        self._processes = []

        # Shared memory 
        self._manager = Manager()
        self._locks = {
            'file_lock': self._manager.Lock(),
            'winners': self._manager.Lock(),
            'agencys_that_finished_sending_bets': self._manager.Lock()
        }
        self._winners = self._manager.dict()
        self._winners_were_set = self._manager.Value('b', False)
        self._agencys_that_finished_sending_bets = self._manager.Value('i', 0)
        self._can_start_lottery = Event()

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

        while not self._recived_sigterm:
            client_socket = self.__accept_new_connection()
            if self._recived_sigterm:
                client_socket.close()
                logging.info('action: closing_client_socket | result: in_progress | reason: recived_sigterm')

                break

            client_handler = ClientHandler(
                client_socket, 
                self._protocol(), 
                self._agencys_that_finished_sending_bets,
                self._winners,
                self._locks,
                self._agencys,
                self._can_start_lottery,
                self._winners_were_set
            )

            process = Process(target=client_handler.start, args=())
            self._processes.append(process)
            process.start()

            # Prevents resources being wasted, if a process terminated and the server is still running, it should be 
            # joined asap
            # self.__clean_up_finished_processes()

        # self.__shutdown()
        

    def __shutdown(self):
        # Propagate sigterm to child processes, if running send sigterm else join
        for process in self._processes:
            if process.is_alive():
                pid = process.pid
                process.termiate()
                logging.info(f'action: propagated_sigterm_to_child_process | pid: {pid}')
            else: 
                pid = process.pid 
                process.join()
                self._processes.remove(process)
                logging.info(f'action: terminating_child_process | result: success | pid: {pid} | reason: recived_sigterm')

        self._server_socket.close()
        logging.info('action: closing_server_socket | result: success | reason: recived_sigterm')
        self._manager.shutdown()
        logging.debug('action: shutting_down_resource_manager | result: success | reason: recived_sigterm')
        
        # Join the processes that sigterm was previously propagated to
        for process in self._processes:
            pid = process.pid
            process.join()
            logging.info(f'action: terminating_child_process | result: success | pid: {pid} | reason: recived_sigterm')

    
    def __clean_up_finished_processes(self):
        for process in self._processes:
            if not process.is_alive():
                pid = process.pid
                process.join()
                logging.info(f'action: terminated_finished_child_process | result: success | pid: {pid}')
                self._processes.remove(process)


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