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
    "retries": 1,      // total # of retries before giving up
    "job_duration": 1, // job duration in seconds
    "expiry": 1        // expiry in seconds
}
```

 - The `job_duration` key determines how long after a task has been
  dispatched to a worker it should be considered a failure if the
  task hasn't been marked done in that amount of time. (Set to any
  negative value to disable this feature.)

 - The `expiry` key determines how long the task should live in the
  queue before it is considered to fail. (Regardless of whether it
  has been dispatched.)

### `POST /wait/`

Waits for queues to be ready and dequeues tasks from them.
The POST request should contain a JSON object with the following schema:

```js
{
    "timeout": 1, // timeout in seconds
    "queues": ["queue-name-1", "queue-name-2"]
}
```

 - The `queues` array can contain repeats to signify that you want
 a number of tasks from the same queue.

 - `timeout` can be set to -1 to wait until all queues are ready.

The reply will be as follows:

```js
[{"id": "job-id", "content": { /* job-content */}, "retries": 0}, ...]
```

It may contain nulls where queues are not ready (have no tasks).

### `POST /done/:id`

Mark a task as completed.
Also deletes it from the queue.

### `POST /fail/:id`

Mark a task as failed.
Failed tasks may be retried (up to the `retries` parameter in the task config).
