// Copyright 2015 - 2017 Ka-Hing Cheung
// Copyright 2015 - 2017 Google Inc. All Rights Reserved.
// Modifications Copyright 2018 The MITRE Corporation
// Authors: Ka-Hing Cheung, Matthew Bianchi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/mattrbianchi/twig"
	"github.com/mitre/fusera/cmd"
)

func init() {
	// Customize the format of log messages
	twig.SetFlags(twig.LstdFlags | twig.Lshortfile)
}

func main() {
	cmd.Execute()
}

// TODO: I believe this code is no longer called, clean up when validated.
var waitedForSignal os.Signal

func waitForSignal(wg *sync.WaitGroup) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGUSR1, syscall.SIGUSR2)

	wg.Add(1)
	go func() {
		waitedForSignal = <-signalChan
		wg.Done()
	}()
}
