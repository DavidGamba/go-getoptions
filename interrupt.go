// This file is part of go-getoptions.
//
// Copyright (C) 2015-2025  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package getoptions

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/DavidGamba/go-getoptions/text"
)

// InterruptContext - Creates a top level context that listens to os.Interrupt, syscall.SIGHUP and syscall.SIGTERM and calls the CancelFunc if the signals are triggered.
// When the listener finishes its work, it sends a message to the done channel.
//
// Use:
//
//	func main() { ...
//	ctx, cancel, done := getoptions.InterruptContext()
//	defer func() { cancel(); <-done }()
//
// NOTE: InterruptContext is a method to reuse gopt.Writer
func InterruptContext() (ctx context.Context, cancel context.CancelFunc, done chan struct{}) {
	done = make(chan struct{}, 1)
	ctx, cancel = context.WithCancel(context.Background())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	go func() {
		defer func() {
			signal.Stop(signals)
			cancel()
			done <- struct{}{}
		}()
		select {
		case <-signals:
			fmt.Fprintf(Writer, "\n%s\n", text.MessageOnInterrupt)
		case <-ctx.Done():
		}
	}()
	return ctx, cancel, done
}
