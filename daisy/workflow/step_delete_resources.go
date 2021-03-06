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

// DeleteResources deletes GCE resources.
type DeleteResources struct {
	Instances, Disks, Images []string
}

func (d *DeleteResources) validate() error {
	// Disk checking.
	for _, disk := range d.Disks {
		if !diskExists(disk) {
			return fmt.Errorf("cannot delete disk. Disk not found: %s", disk)
		}
		if err := diskNamesToDelete.add(disk); err != nil {
			return fmt.Errorf("error scheduling disk for deletion: %s", err)
		}
	}

	// Instance checking.
	for _, i := range d.Instances {
		if !instanceExists(i) {
			return fmt.Errorf("cannot delete instance. Instance not found: %s", i)
		}
		if err := instanceNamesToDelete.add(i); err != nil {
			return fmt.Errorf("error scheduling instance for deletion: %s", err)
		}
	}

	return nil
}

func (d *DeleteResources) run(w *Workflow) error {
	var wg sync.WaitGroup
	e := make(chan error)

	for _, i := range d.Instances {
		wg.Add(1)
		go func(i string) {
			defer wg.Done()
			if err := w.deleteInstance(namer(i, w.Name, w.suffix)); err != nil {
				e <- err
			}
		}(i)
	}

	for _, i := range d.Images {
		wg.Add(1)
		go func(i string) {
			defer wg.Done()
			if err := w.deleteImage(i); err != nil {
				e <- err
			}
		}(i)
	}

	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		if err != nil {
			return err
		}
	case <-w.Ctx.Done():
		return nil
	}

	// Delete disks only after instances have been deleted.
	e = make(chan error)
	for _, d := range d.Disks {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			if err := w.deleteDisk(namer(d, w.Name, w.suffix)); err != nil {
				e <- err
			}
		}(d)
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
