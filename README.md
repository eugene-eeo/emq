# emq

Tiny message queue server over HTTP.
Primarily an excuse for me to learn net/http.

Features:

 - Retries
 - Per-dispatch timeouts
 - Job timeouts

To enqueue some data:

    $ curl --data '{"content": {"custom payload": "goes here"}}' localhost:8080/enqueue/queue-name
    {"id":"e08ef6fa-9612-4a10-abed-3aab4fe337b4"}

To dequeue:

    $ curl --data '{"queues": ["queue-name"], "timeout": 10}' localhost:8080/wait/
    [{"id": "e08ef6fa-...", "content": {...}}]


To mark task completion and failure, resp:

    $ curl localhost:8080/done/:id
    $ curl localhost:8080/fail/:id
