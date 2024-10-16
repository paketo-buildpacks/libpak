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
	"io"
	"os/exec"
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

	// Stdin is the Reader to use for stdin.
	Stdin io.Reader

	// Stdout is the Writer to use for stdout.
	Stdout io.Writer

	// Stderr is the Writer to use for stderr.
	Stderr io.Writer
}

//go:generate mockery --name Executor --case=underscore

// Executor is the interface for types that can execute an Execution.
type Executor interface {

	// Execute executes the command described in the Execution.
	Execute(execution Execution) error
}

// CommandExecutor is an implementation of Executor that uses exec.Command and runs the command without a TTY.
type CommandExecutor struct{}

func (CommandExecutor) Execute(execution Execution) error {
	// #nosec G204 -- this is a generic executor so this cannot apply
	cmd := exec.Command(execution.Command, execution.Args...)

	if execution.Dir != "" {
		cmd.Dir = execution.Dir
	}

	if len(execution.Env) > 0 {
		cmd.Env = execution.Env
	}

	cmd.Stdin = execution.Stdin
	cmd.Stdout = execution.Stdout
	cmd.Stderr = execution.Stderr

	return cmd.Run()
}
