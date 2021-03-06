//  Copyright 2017 Google Inc. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package workflow

import (
	"fmt"
	"sync"
)

// WaitForInstancesStopped is a Daisy WaitForInstancesStopped workflow step.
type WaitForInstancesStopped []string

func (s *WaitForInstancesStopped) run(w *Workflow) error {
	var wg sync.WaitGroup
	e := make(chan error)

	for _, name := range *s {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := w.ComputeClient.WaitForInstanceStopped(w.Project, w.Zone, namer(name, w.Name, w.suffix)); err != nil {
				e <- err
			}
		}(name)
	}

	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		return err
	case <-w.Ctx.Done():
		return nil
	}
}

func (s *WaitForInstancesStopped) validate() error {
	// Instance checking.
	for _, i := range *s {
		if !instanceExists(i) {
			return fmt.Errorf("cannot wait for instance stopped. Instance not found: %s", i)
		}
	}
	return nil
}
