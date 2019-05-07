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
	"fmt"
	"log"
	"testing"
	"time"
)

func TestPoolSize(t *testing.T) {
	pool := NewPool(10, func(in interface{}) (interface{}, error) { return "hello " + in.(string) + "!", nil })
	if exp, act := 10, len(pool.workers); exp != act {
		t.Errorf("Wrong size of pool: %v != %v", act, exp)
	}

	pool.SetSize(10)
	if exp, act := 10, pool.GetSize(); exp != act {
		t.Errorf("Wrong size of pool: %v != %v", act, exp)
	}

	pool.SetSize(9)
	if exp, act := 9, pool.GetSize(); exp != act {
		t.Errorf("Wrong size of pool: %v != %v", act, exp)
	}

	pool.SetSize(0)
	if exp, act := 0, pool.GetSize(); exp != act {
		t.Errorf("Wrong size of pool: %v != %v", act, exp)
	}

	pool.SetSize(10)
	if exp, act := 10, pool.GetSize(); exp != act {
		t.Errorf("Wrong size of pool: %v != %v", act, exp)
	}

	// Finally, make sure we still have actual active workers
	r, _ := pool.Execute("Orc")
	if exp, act := "hello Orc!", r.(string); exp != act {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}

	pool.Close()
	if exp, act := 0, pool.GetSize(); exp != act {
		t.Errorf("Wrong size of pool: %v != %v", act, exp)
	}
}

func TestPoolFunc(t *testing.T) {
	pool := NewPool(10, func(in interface{}) (interface{}, error) {
		intVal := in.(int)
		return intVal * 2, nil
	})
	defer pool.Close()

	for i := 0; i < 10; i++ {
		ret, _ := pool.Execute(10)
		if exp, act := 20, ret.(int); exp != act {
			t.Errorf("Wrong result: %v != %v", act, exp)
		}
	}
}

type Operation struct {
	a int
	b int
}

func Execute(op Operation) int {
	time.Sleep(3000 * time.Millisecond)
	return op.a + op.b
}

//func Test_Single_Pool(t *testing.T) {
//
//	pool := NewPool(1, func(in interface{}) interface{} {
//		return Execute(in.(Operation))
//	})
//
//	r := pool.Execute(Operation{1,1})
//
//	log.Println(fmt.Sprintf("Result => %v", r))
//
//	if pool.GetQueuedJobs() != 0 {
//		panic("Invalid jobs in the queue")
//	}
//	pool.Close()
//
//}
//
//
func Test_Basic_Pool(t *testing.T) {

	pool := NewPool(5, func(in interface{}) (interface{}, error) {
		return Execute(in.(Operation)), nil
	})

	ops := []interface{}{Operation{1, 1},
		Operation{1, 2},
		Operation{1, 3},
		Operation{1, 43},
		Operation{1, 13},
		Operation{1, 23},
		Operation{1, 33},
		Operation{1, 4},
		Operation{1, 5},
		Operation{1, 6},
		Operation{1, 7},
		Operation{1, 14},
		Operation{1, 15},
		Operation{1, 16},
		Operation{1, 17},
		Operation{1, 8}}

	output := pool.ExecuteM(ops)

	//wg.Wait()
	for r := range output {
		log.Println(fmt.Sprintf("Result => %v", r))
	}

	if pool.GetQueuedJobs() != 0 {
		panic("Invalid jobs in the queue")
	}
	pool.Close()

}
