package main

import (
	"encoding/json"
	"github.com/eugene-eeo/emq/tctx2"
	"log"
	"net/http"
	"sync"
	"time"
)

type server struct {
	uid        uint64
	mu         sync.Mutex
	dispatched chan TaskInfo
	waiters    *Waiters
	tasksById  map[string]*Task
	tasks      map[TaskUid]*Task
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
	s.tasks[t.Uid] = t
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
			http.Error(w, http.StatusText(404), 404)
			return
		}

		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(tc); err != nil {
			http.Error(w, err.Error(), 422)
			return
		}

		if err := tc.Fill(); err != nil {
			http.Error(w, err.Error(), 422)
			return
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		if s.tasksById[tc.Id] != nil {
			http.Error(w, "Task already enqueued", 401)
			return
		}

		s.uid++
		task := NewTaskFromConfig(tc, queueName)
		task.Uid = TaskUid(s.uid)
		s.enqueueTask(task)
		if task.Expiry > 0 {
			s.context.Add(TaskInfo{task.Uid, StatusExpired}, task.Expiry)
		}

		enc := json.NewEncoder(w)
		enc.Encode(map[string]string{"id": task.Id})
	})
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

		s.waitForWaiter(waiter)

		s.mu.Lock()
		for _, task := range waiter.Tasks {
			if task != nil && task.JobDuration > 0 {
				// Add timers if necessary
				s.context.Add(TaskInfo{task.Uid, StatusTimeout}, task.JobDuration)
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
		t := s.tasksById[taskId]
		if t != nil {
			go func() { s.dispatched <- TaskInfo{t.Uid, StatusOk} }()
		}
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
		t := s.tasksById[taskId]
		if t != nil {
			go func() { s.dispatched <- TaskInfo{t.Uid, StatusFail} }()
		}
	})
}

func (s *server) handleTaskInfo(ti TaskInfo) {
	uid := ti.uid
	t := s.tasks[uid]
	switch ti.status {
	case StatusOk:
		if t != nil {
			delete(s.tasksById, t.Id)
			delete(s.tasks, uid)
			log.Printf("Removed %s (OK)", t.Id)
		}
	case StatusExpired:
		if t != nil {
			delete(s.tasksById, t.Id)
			delete(s.tasks, uid)
			log.Printf("Removed %s (Expired)", t.Id)
		}
	case StatusTimeout:
		fallthrough
	case StatusFail:
		// Already timed out somewhere
		if t == nil {
			return
		}
		t.Retries--
		if t.Retries >= 0 {
			s.enqueueTask(t)
			log.Printf("Requeue %s", t.Id)
		} else {
			delete(s.tasksById, t.Id)
			delete(s.tasks, uid)
			log.Printf("Removed %s (Fail)", t.Id)
		}
	}
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
