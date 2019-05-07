/*
 * Copyright (c) 2019.
 *
 * This file is part of gopool.
 *
 * gopool is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * gopool is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with gopool.  If not, see <https://www.gnu.org/licenses/>.
 */

package gopool

import (
	"errors"
	"sync"
	"sync/atomic"
)

// This is the basic worker struct
// the pool referenced
// The close channel receives a signal to close the worker
// The closed channel signals that the worker was closed
type structWorker struct {
	pool   *Pool
	close  chan interface{}
	closed chan interface{}
}

// the Result of any go routine running inside the pool
type Result struct {
	Output interface{}
	Err    error
}

// The request channel is used to send the result back
type request struct {
	param         interface{}
	outputChannel chan Result
	wg            *sync.WaitGroup
}

var (
	ErrPoolClosed = errors.New("the pool was closed")
)

func (worker *structWorker) run() {
	defer close(worker.closed)
	for {
		select {
		case request, ok := <-worker.pool.inputChannel:
			if !ok {
				return
			}
			// execute the function
			result, err := worker.pool.f(request.param)
			request.outputChannel <- Result{result, err}
			atomic.AddInt64(&worker.pool.queuedJobs, -1)
			if request.wg != nil {
				request.wg.Done()
			}
		case <-worker.close:
			return
		}
	}
}

func (worker *structWorker) stop() {
	close(worker.close)
}

func (worker *structWorker) join() {
	<-worker.closed
}

// The Pool struct
// The mutex control access for some operations in the pool
// The queuedJobs tells the number of requests inside the pool in some moment
// The inputChannel channel receives a new job to execute
// The workers is the pool array
// the f func is the routine to be executed
type Pool struct {
	mutex        sync.Mutex
	queuedJobs   int64
	inputChannel chan request
	workers      []structWorker
	f            func(interface{}) (interface{}, error)
}

func (pool *Pool) SetSize(n int) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	poolLen := len(pool.workers)
	for i := poolLen; i < n; i++ {
		pool.workers = append(pool.workers, pool.newWorker())
	}
	for i := n; i < poolLen; i++ {
		pool.workers[i].stop()
	}
	for i := n; i < poolLen; i++ {
		pool.workers[i].join()
	}
	// resize the workers
	pool.workers = pool.workers[:n]
}

func (pool *Pool) GetSize() int {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	return len(pool.workers)
}

func (pool *Pool) GetQueuedJobs() int64 {
	return atomic.LoadInt64(&pool.queuedJobs)
}

func (pool *Pool) newWorker() structWorker {
	worker := structWorker{
		pool:   pool,
		close:  make(chan interface{}),
		closed: make(chan interface{}),
	}
	go worker.run()
	return worker
}

func (pool *Pool) Close() {
	pool.SetSize(0)
}

// Execute sync inside the pool
func (pool *Pool) Execute(in interface{}) (interface{}, error) {
	output := make(chan Result)
	atomic.AddInt64(&pool.queuedJobs, 1)
	pool.inputChannel <- request{param: in, outputChannel: output}
	result, ok := <-output
	if !ok {
		return nil, ErrPoolClosed
	}
	return result.Output, result.Err
}

// Execute async
func (pool *Pool) ExecuteA(in interface{}) chan Result {
	atomic.AddInt64(&pool.queuedJobs, 1)
	return pool.ExecuteM([]interface{}{in})
}

// Execute multiples requests async
func (pool *Pool) ExecuteM(in []interface{}) chan Result {
	count := len(in)
	output := make(chan Result, count)
	atomic.AddInt64(&pool.queuedJobs, int64(count))
	var wg sync.WaitGroup
	wg.Add(len(in))
	for _, p := range in {
		pool.inputChannel <- request{param: p, outputChannel: output, wg: &wg}
	}
	go func() {
		wg.Wait()
		close(output)
	}()
	return output
}

func NewPool(size int, f func(interface{}) (interface{}, error)) *Pool {
	pool := &Pool{
		inputChannel: make(chan request),
		f:            f,
	}
	pool.SetSize(size)
	return pool
}
