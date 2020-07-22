package main

import "time"
import "sync"
import "net/http"
import "encoding/json"
import "github.com/eugene-eeo/emq/uid"

type Server struct {
	sync.Mutex
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
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024*1024))
	dec.DisallowUnknownFields()
	err := dec.Decode(&tc)
	if err != nil {
		http.Error(w, http.StatusText(400), 400)
		return
	}

	s.Lock()
	defer s.Unlock()

	task := tc.ToTask(s.GetID(), time.Now())
	s.mq.Add(qn, &task)
	go s.UpdateWaitSpecs()
}

func (s *Server) GC(now time.Time) {
	s.Lock()
	defer s.Unlock()
	s.mq.GC(now)
}

func (s *Server) Loop() {
	for {
		now := <-time.After(time.Duration(5) * time.Minute)
		s.GC(now)
	}
}

func (s *Server) Wait(w http.ResponseWriter, r *http.Request) {
	wsc := WaitSpecConfig{}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024*1024))
	dec.DisallowUnknownFields()
	err := dec.Decode(&wsc)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	var tasks []*Task
	ws := wsc.ToWaitSpec()

	if ws.Timeout == 0 {
		s.Lock()
		now := time.Now()
		tasks = ws.Take(s.mq, now)
		s.Consume(tasks, now)
		s.Unlock()
	} else {
		s.Lock()
		s.ws.Append(&ws)
		s.Unlock()

		go s.UpdateWaitSpecs()

		select {
		case wd := <-ws.ready:
			tasks = wd.tasks
		case now := <-time.After(ws.Timeout):
			// timeout -- just claim what we can
			s.Lock()
			tasks = ws.Take(s.mq, now)
			s.Consume(tasks, now)
			s.Unlock()
		}

		s.Lock()
		s.ws.Remove(&ws)
		s.Unlock()
	}

	w.WriteHeader(200)
	enc := json.NewEncoder(w)
	enc.Encode(tasks)
}

func (s *Server) FindDispatchedTaskHTTP(url string, next func(t *Task)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len(url):]
		uid, err := uid.FromString(id)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		s.Lock()
		defer s.Unlock()
		now := time.Now()
		task := s.mq.Find(uid, now)
		if task == nil || !task.InDispatch(now) {
			http.Error(w, "Invalid task ID or task expired/dispatch timed out.", 404)
			return
		}
		next(task)
	}
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
	s.mux.Handle("/ack/", PostJSON(s.FindDispatchedTaskHTTP("/ack/", func(t *Task) {
		s.mq.DeleteTask(t)
		go s.UpdateWaitSpecs()
	})))
	s.mux.Handle("/nak/", PostJSON(s.FindDispatchedTaskHTTP("/nak/", func(t *Task) {
		s.mq.Failed(t)
		go s.UpdateWaitSpecs()
	})))
	return s
}
