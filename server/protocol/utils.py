"""
Ensures no short reads happen, reads till need_to_read bytes where read
from the socket
"""
def read_all(socket, need_to_read) -> bytes:
    buffer = b''
    while need_to_read > 0:
        read = socket.recv(need_to_read)
        # Client socket closed
        if len(read) == 0:
            raise OSError("error: tried to read from a closed socket")
        need_to_read -= len(read)
        buffer += read

    return buffer