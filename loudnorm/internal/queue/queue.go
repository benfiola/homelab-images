package queue

import "sync"

type Queue struct {
	Items chan string
	mu    sync.Mutex
	paths map[string]bool
}

func New(capacity int) *Queue {
	return &Queue{
		Items: make(chan string, capacity),
		paths: make(map[string]bool),
	}
}

func (q *Queue) Add(path string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.paths[path] {
		return false
	}

	q.paths[path] = true
	q.Items <- path
	return true
}

func (q *Queue) Get() string {
	path := <-q.Items
	q.mu.Lock()
	delete(q.paths, path)
	q.mu.Unlock()
	return path
}

func (q *Queue) Len() int {
	return len(q.Items)
}

func (q *Queue) Close() {
	close(q.Items)
}
