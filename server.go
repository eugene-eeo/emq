package main

import "strconv"
import "time"
import "sync"
import "net/http"
import "encoding/json"
import "github.com/eugene-eeo/emq/uid"

type Server struct {
	sync.RWMutex
	mq     *MQ
	ws     *Waiters
	mux    *http.ServeMux
	gcfreq time.Duration
}

func (s *Server) GetID() uid.UID {
	for {
		id := uid.Generate()
		if s.mq.Tasks[id] == nil {
			return id
		}
	}
}

func (s *Server) UpdateWaitSpecs() {
	s.Lock()
	defer s.Unlock()

	now := time.Now()
	s.mq.GC(now)
	for w := s.ws.Head(); w != nil; w = w.Next() {
		tasks, ready := w.Ready(s.mq, now)
		if ready {
			s.Consume(tasks, now)
			w.ready <- WaitDone{now, tasks}
		}
	}
}

func (s *Server) Consume(tasks []*Task, now time.Time) {
	for _, t := range tasks {
		if t != nil {
			s.mq.Dispatch(t, now)
		}
	}
}

func (s *Server) Enqueue(w http.ResponseWriter, r *http.Request) {
	// decode request
	qn := r.URL.Path[len("/enqueue/"):]
	tc := TaskConfig{}
	if err := decodeJSONFromHTTP(w, r, &tc); err != nil {
		return
	}

	s.Lock()
	defer s.Unlock()

	task := tc.ToTask(s.GetID(), time.Now())
	s.mq.Add(qn, &task)
	go s.UpdateWaitSpecs()
}

func (s *Server) GCLoop() {
	for {
		now := <-time.After(s.gcfreq)
		s.Lock()
		s.mq.GC(now)
		s.Unlock()
	}
}

func (s *Server) Wait(w http.ResponseWriter, r *http.Request) {
	wsc := WaitSpecConfig{}
	if err := decodeJSONFromHTTP(w, r, &wsc); err != nil {
		return
	}

	var tasks []*Task
	ws := wsc.ToWaitSpec()
	have_tasks := false

	if ws.Timeout > 0 {
		s.Lock()
		s.ws.Append(&ws)
		s.Unlock()

		go s.UpdateWaitSpecs()

		select {
		case wd := <-ws.ready:
			tasks = wd.tasks
			have_tasks = true
		case <-time.After(ws.Timeout):
		}
		s.Lock()
		s.ws.Remove(&ws)
		s.Unlock()
	}

	if !have_tasks {
		// Just take what we can (because of timeout)
		s.Lock()
		now := time.Now()
		tasks = ws.Take(s.mq, now)
		s.Consume(tasks, now)
		s.Unlock()
	}
	w.WriteHeader(200)
	enc := json.NewEncoder(w)
	enc.Encode(tasks)
}

type UIDs struct {
	IDs []uid.UID `json:"ids"`
}

func (s *Server) UpdateDispatchedTasksHTTP(then func(t *Task)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uids := UIDs{}
		if err := decodeJSONFromHTTP(w, r, &uids); err != nil {
			return
		}

		s.Lock()
		defer s.Unlock()
		now := time.Now()
		for _, uid := range uids.IDs {
			task := s.mq.Find(uid, now)
			if task != nil && task.InDispatch(now) {
				then(task)
			}
		}
		go s.UpdateWaitSpecs()
	}
}

func (s *Server) Queues(w http.ResponseWriter, r *http.Request) {
	s.RLock()
	queues := make([]string, 0, len(s.mq.Queues))
	for qn := range s.mq.Queues {
		queues = append(queues, qn)
	}
	s.RUnlock()
	w.WriteHeader(200)
	enc := json.NewEncoder(w)
	enc.Encode(queues)
}

func (s *Server) Peek(w http.ResponseWriter, r *http.Request) {
	qn := r.URL.Path[len("/peek/"):]
	n, err := strconv.Atoi(r.URL.Query().Get("n"))
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	s.RLock()
	now := time.Now()
	tasks := []*Task{}
	q := s.mq.Queues[qn]
	if q != nil {
		t := q.Head()
		for i := 0; i < n; i++ {
			u, next := q.NextUndispatched(t, now)
			if u == nil {
				break
			}
			tasks = append(tasks, u)
			t = next
		}
	}
	s.RUnlock()
	w.WriteHeader(200)
	enc := json.NewEncoder(w)
	enc.Encode(tasks)
}

func NewServer(gcfreq time.Duration) *Server {
	s := &Server{
		mq:     NewMQ(),
		ws:     NewWaiters(),
		mux:    http.NewServeMux(),
		gcfreq: gcfreq,
	}
	PostJSON := func(f http.HandlerFunc) http.Handler {
		return Chain(f,
			EnforceMethodMiddleware("POST"),
			EnforceJSONMiddleware(),
		)
	}
	s.mux.Handle("/enqueue/", PostJSON(s.Enqueue))
	s.mux.Handle("/wait/", PostJSON(s.Wait))
	s.mux.Handle("/ack/", PostJSON(s.UpdateDispatchedTasksHTTP(s.mq.DeleteTask)))
	s.mux.Handle("/nak/", PostJSON(s.UpdateDispatchedTasksHTTP(s.mq.Failed)))
	s.mux.Handle("/peek/", Chain(s.Peek, EnforceMethodMiddleware("GET")))
	s.mux.Handle("/queues/", Chain(s.Queues, EnforceMethodMiddleware("GET")))
	return s
}
