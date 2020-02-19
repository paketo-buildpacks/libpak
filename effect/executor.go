/*
 * Copyright 2018-2020 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package effect

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
)

// Execution is information about a command to run.
type Execution struct {

	// Command is the command to run.
	Command string

	// Args is the arguments to the command.
	Args []string

	// Dir is the working directory the command is run in.  Defaults to the current working directory.
	Dir string

	// Environment is the environment variables that the command is run with.  Defaults to current environment.
	Env []string

	// Stdout is the Writer to use for stdout.
	Stdout io.Writer

	// Stderr is the Writer to use for stderr.
	Stderr io.Writer
}

//go:generate mockery -name Executor -case=underscore

// Executor is the interface for types that can execute an Execution.
type Executor interface {

	// Execute executes the command described in the Execution.
	Execute(execution Execution) error
}

// CommandExecutor is an implementation of Executor that uses exec.Command and runs the command without a TTY.
type CommandExecutor struct{}

func (CommandExecutor) Execute(execution Execution) error {
	cmd := exec.Command(execution.Command, execution.Args...)

	if execution.Dir != "" {
		cmd.Dir = execution.Dir
	}

	if len(execution.Env) > 0 {
		cmd.Env = execution.Env
	}

	cmd.Stdout = execution.Stdout
	cmd.Stderr = execution.Stderr

	return cmd.Run()
}

// TTYExecutor is an implementation of Executor that uses exec.Command and runs the command with a TTY.
type TTYExecutor struct{}

func (t TTYExecutor) Execute(execution Execution) error {
	cmd := exec.Command(execution.Command, execution.Args...)

	if execution.Dir != "" {
		cmd.Dir = execution.Dir
	}

	if len(execution.Env) > 0 {
		cmd.Env = execution.Env
	}

	f, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("unable to start PTY: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(execution.Stdout, f); err != nil {
		if !t.isEIO(err) {
			return fmt.Errorf("unable to write output: %w", err)
		}
	}

	return cmd.Wait()
}

func (TTYExecutor) isEIO(err error) bool {
	pe, ok := err.(*os.PathError)
	if !ok {
		return false
	}

	return pe.Err == syscall.EIO
}

// NewExecutor creates a new Executor.  If the buildpack is currently running in a TTY, returns a TTY-aware Executor.
func NewExecutor() Executor {
	// TODO: Remove once TTY support is in place
	return TTYExecutor{}
	// if isatty.IsTerminal(os.Stdout.Fd()) {
	// 	return TTYExecutor{}
	// } else {
	// 	return CommandExecutor{}
	// }
}
