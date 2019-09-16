package main

import (
	"encoding/json"
	"github.com/eugene-eeo/emq/tctx2"
	"github.com/satori/go.uuid"
	"log"
	"net/http"
	"sync"
	"time"
)

type server struct {
	mu         sync.Mutex
	dispatched chan TaskInfo
	waiters    *Waiters
	tasksById  map[uuid.UUID]*Task
	queues     map[string]*Queue
	router     *http.ServeMux
	version    Version
	context    *tctx2.Context
}

func (s *server) getQueue(name string) *Queue {
	q, ok := s.queues[name]
	if !ok {
		q = NewQueue(name)
		s.queues[name] = q
	}
	return q
}

func (s *server) enqueueTask(t *Task) {
	s.tasksById[t.Id] = t
	s.getQueue(t.QueueName).Enqueue(t)
	s.waiters.Update(s.queues)
}

func (s *server) hello(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)
	enc.Encode(s.version)
}

func (s *server) enqueue(w http.ResponseWriter, r *http.Request) {
	tc := &TaskConfig{}
	queueName := r.URL.Path[len("/enqueue/"):]
	if len(queueName) == 0 {
		http.Error(w, http.StatusText(404), 404)
		return
	}

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(tc); err != nil {
		http.Error(w, err.Error(), 422)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id, err := uuid.NewV4()
	for {
		if err != nil {
			http.Error(w, err.Error(), 422)
			return
		}
		if s.tasksById[id] == nil {
			break
		}
		id, err = uuid.NewV4()
	}

	task := NewTaskFromConfig(tc, queueName)
	task.Id = id
	s.enqueueTask(task)
	if task.Expiry > 0 {
		s.context.Add(TaskInfo{task.Id, StatusExpired}, task.Expiry)
	}

	enc := json.NewEncoder(w)
	enc.Encode(task.Id)
}

func (s *server) waitForWaiter(w *Waiter) {
	if w.Timeout < 0 {
		<-w.Ready
		return
	}
	timer := time.NewTimer(w.Timeout)
	select {
	case <-timer.C:
		timer.Stop()
		s.mu.Lock()
		defer s.mu.Unlock()
		// If we haven't been consumed yet
		if !w.Done {
			w.Consume(s.queues)
			s.waiters.Remove(w)
		}
	case <-w.Ready:
		timer.Stop()
	}
}

func (s *server) addWaiter(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	enc := json.NewEncoder(w)
	wc := WaitConfigJson{}

	err := dec.Decode(&wc)
	if err != nil {
		http.Error(w, http.StatusText(422), 422)
		return
	}

	waiter := NewWaiterFromConfig(&wc)

	// Fast case
	if waiter.Timeout == 0 {
		s.mu.Lock()
		waiter.Consume(s.queues)
		s.mu.Unlock()
	} else {
		s.mu.Lock()
		s.waiters.AddWaiter(waiter)
		s.waiters.Update(s.queues)
		s.mu.Unlock()
		// Wait happens here!
		s.waitForWaiter(waiter)
	}

	s.mu.Lock()
	for _, task := range waiter.Tasks {
		if task != nil && task.JobDuration > 0 {
			// Add timers if necessary
			s.context.Add(TaskInfo{task.Id, StatusTimeout}, task.JobDuration)
		}
	}
	log.Print("Waiter finished")
	enc.Encode(waiter.Tasks)
	s.mu.Unlock()
}

func (s *server) makeTaskUpdater(prefix string, status TaskStatus) http.HandlerFunc {
	size := len(prefix)
	return func(w http.ResponseWriter, r *http.Request) {
		taskId := r.URL.Path[size:]
		uid, err := uuid.FromString(taskId)
		if err != nil {
			http.Error(w, err.Error(), 422)
			return
		}
		s.dispatched <- TaskInfo{uid, status}
	}
}

func (s *server) handleTaskInfo(ti TaskInfo) {
	t := s.tasksById[ti.id]
	if t == nil {
		return
	}
	if ti.status == StatusTimeout || ti.status == StatusFail {
		t.Retries--
		if t.Retries >= 0 {
			s.enqueueTask(t)
			log.Printf("Requeue %s", t.Id)
			// Don't delete
			return
		}
	}
	delete(s.tasksById, t.Id)
	log.Printf("Removed %s", t.Id)
}

func (s *server) listenDispatched() {
	for {
		select {
		case obj := <-s.context.C:
			switch obj.(type) {
			case TaskInfo:
				s.mu.Lock()
				s.handleTaskInfo(obj.(TaskInfo))
				s.mu.Unlock()
			}
		case taskInfo := <-s.dispatched:
			s.mu.Lock()
			s.handleTaskInfo(taskInfo)
			s.mu.Unlock()
		}
	}
}

func (s *server) routes() {
	s.router.Handle("/fail/", Chain(s.makeTaskUpdater("/fail/", StatusFail), logRequest, enforceMethod("POST")))
	s.router.Handle("/done/", Chain(s.makeTaskUpdater("/fail/", StatusDone), logRequest, enforceMethod("POST")))
	s.router.Handle("/wait/", Chain(
		s.addWaiter,
		logRequest,
		enforceMethod("POST"),
		enforceJSONHandler,
	))
	s.router.Handle("/enqueue/", Chain(
		s.enqueue,
		logRequest,
		enforceMethod("POST"),
		enforceJSONHandler,
	))
	s.router.Handle("/", Chain(s.hello, logRequest))
}
