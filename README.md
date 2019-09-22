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
  task hasn't been marked done in that amount of time.
  Set to any value <= 0 to disable this feature.
  By default, there is no limit on the job duration.

 - The `expiry` key determines how long the task should live in the
  queue before it is considered to fail, regardless of whether it
  has been dispatched.
  Setting this to any value <= 0 disables this feature.
  By default, there is no limit on the job expiry.

### `POST /wait/`

The trailing slash is important.
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
 By default `timeout`'s value will be 0, which is the fast case
 where the waiter returns almost immediately.

The reply will be an array of tasks, as follows:

```js
[
  {
    "id": "job-id",
    "retries": 0,
    "content": { /* job-content */},
  },
  ...
]
```

It may contain nulls where queues are not ready (have no tasks).
By definition, nulls will only appear when
the waiter has timed out or a timeout of 0 was specified.

### `POST /done/:id`

Mark a task as completed.
Also deletes it from the queue.

### `POST /fail/:id`

Mark a task as failed.
Failed tasks may be retried (up to the `retries` parameter in the task config).
