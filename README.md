# emq

Tiny message queue server over HTTP.
Primarily an excuse for me to learn net/http.

Features:

 - Retries
 - Per-dispatch timeouts
 - Job timeouts

## Running

```sh
$ go install github.com/eugene-eeo/emq
$ emq --addr ':8080'
```

## Endpoints

### `POST /enqueue/<queue-name>`

Enqueues a task in the specified *queue-name*.
The POST request should include a JSON object in the body with the following schema:

```js
{
    "content": { /* Anything can go here */ },
    "retry":  1, // job duration in seconds (default: 5 minutes)
    "expiry": 1  // expiry in seconds (default: 1 day)
}
```

 - `retry` determines how long after a task has been
  dispatched to a worker it should be considered a failure
  if the task hasn't been ACKed in that amount of time.

 - The `expiry` key determines how long the task should live
  in the queue before it is deleted, regardless of whether it
  has been dispatched.

### `POST /wait/`

The trailing slash is important.
Waits for queues to be ready and dequeues tasks from them.
The POST request should contain a JSON object with the following schema:

```js
{
    "timeout": 1, // timeout in milliseconds (default: 0)
    "queues": ["queue-name-1", "queue-name-2"]
}
```

 - The `queues` array can contain repeats.
 - By default `timeout`'s value will be 0, which is the fast case
   where the waiter returns almost immediately.

The reply will be an array of tasks, as follows:

```js
[
  {
    "id": "job-id",
    "content": { /* job-content */},
  },
  ...
]
```

It may contain nulls where queues are not ready (have no tasks).
By definition, nulls will only appear when the waiter has timed
out or a timeout of 0 was specified.

### `POST /ack/`

Mark tasks as completed.
Also deletes it from the queue.

```js
{
    "ids": ["..."]
}
```

### `POST /nak/`

Same as `/ack`, but mark tasks as failed.
Failed tasks may be retried many times before they expire.
However they will be put in the back of their queues.


### `GET /peek/<queue-name>?n=<num>`

Get (but not dispatch) at most `num` number of jobs from `<queue-name>`.

### `GET /queues/`

Get the names of all current queues (as a JSON array of strings).
