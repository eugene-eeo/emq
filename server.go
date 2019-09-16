package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
)

type server struct {
	mu         sync.Mutex
	waiters    *Waiters
	dispatched *Dispatched
	tasks      map[string]*Task
	queues     map[string]*Queue
	router     *http.ServeMux
	version    Version
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
	s.tasks[t.Id] = t
	s.getQueue(t.QueueName).Enqueue(t)
	s.waiters.Update(s.queues)
}

func (s *server) hello() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)
		enc.Encode(s.version)
	})
}

func (s *server) enqueue() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tc := &TaskConfig{}
		queueName := r.URL.Path[len("/enqueue/"):]
		if len(queueName) == 0 {
			http.Error(w, http.StatusText(422), 422)
			return
		}

		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(tc); err != nil {
			http.Error(w, http.StatusText(422), 422)
			return
		}

		if err := tc.Fill(); err != nil {
			http.Error(w, err.Error(), 422)
			return
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		_, ok := s.tasks[tc.Id]
		if ok {
			http.Error(w, "Task already enqueued", 401)
			return
		}

		task := NewTaskFromConfig(tc, queueName)
		s.enqueueTask(task)

		enc := json.NewEncoder(w)
		enc.Encode(map[string]string{"id": task.Id})
	})
}

func (s *server) addWaiter() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dec := json.NewDecoder(r.Body)
		enc := json.NewEncoder(w)
		wc := WaitConfigJson{}

		err := dec.Decode(&wc)
		if err != nil {
			http.Error(w, http.StatusText(422), 422)
			return
		}

		s.mu.Lock()
		waiter := NewWaiterFromConfig(&wc)
		s.waiters.AddWaiter(waiter)
		s.waiters.Update(s.queues)
		s.mu.Unlock()

		<-waiter.Ready

		s.mu.Lock()
		for _, task := range waiter.Tasks {
			if task != nil {
				s.dispatched.Track(task)
			}
		}
		log.Print("Waiter finished")
		enc.Encode(waiter.Tasks)
		s.mu.Unlock()
	})
}

func (s *server) done() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		taskId := r.URL.Path[len("/done/"):]
		if len(taskId) == 0 {
			http.Error(w, http.StatusText(422), 422)
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		s.dispatched.Put(s.tasks[taskId], StatusOk)
	})
}

func (s *server) fail() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		taskId := r.URL.Path[len("/fail/"):]
		if len(taskId) == 0 {
			http.Error(w, http.StatusText(422), 422)
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		s.dispatched.Put(s.tasks[taskId], StatusFail)
	})
}

func (s *server) listenDispatched() {
	for taskInfo := range s.dispatched.fwd {
		s.mu.Lock()
		t := taskInfo.task
		switch taskInfo.status {
		case StatusOk:
			delete(s.tasks, t.Id)
			log.Printf("Removed %s (OK)", t.Id)
		case StatusFail:
			if s.tasks[t.Id] != nil {
				t.Retries--
				if t.Retries >= 0 {
					s.enqueueTask(t)
					log.Printf("Requeue %s", t.Id)
				} else {
					delete(s.tasks, t.Id)
					log.Printf("Removed %s (Fail)", t.Id)
				}
			}
		}
		s.dispatched.Untrack(t)
		s.mu.Unlock()
	}
}

func (s *server) routes() {
	s.router.Handle("/fail/", Chain(s.fail(), logRequest, enforceMethod("POST")))
	s.router.Handle("/done/", Chain(s.done(), logRequest, enforceMethod("POST")))
	s.router.Handle("/wait/", Chain(
		s.addWaiter(),
		logRequest,
		enforceMethod("POST"),
		enforceJSONHandler,
	))
	s.router.Handle("/enqueue/", Chain(
		s.enqueue(),
		logRequest,
		enforceMethod("POST"),
		enforceJSONHandler,
	))
	s.router.Handle("/", Chain(s.hello(), logRequest))
}
