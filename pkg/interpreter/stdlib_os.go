package interpreter

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func loadOS(env *Environment) {
	// -----------------------------------------------------------------------
	// Environment Variables
	// -----------------------------------------------------------------------

	// env(name) -- get an environment variable, returns nil if not set
	env.Set("env", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("env: expected 1 argument, got %d", len(args))
		}
		name, ok := args[0].(*String)
		if !ok {
			return newError("env: argument must be a string, got %s", args[0].Type())
		}
		val, exists := os.LookupEnv(name.Value)
		if !exists {
			return &Nil{}
		}
		return &String{Value: val}
	}})

	// set_env(name, value) -- set an environment variable
	env.Set("set_env", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 2 {
			return newError("set_env: expected 2 arguments, got %d", len(args))
		}
		name, ok := args[0].(*String)
		if !ok {
			return newError("set_env: first argument must be a string, got %s", args[0].Type())
		}
		value, ok := args[1].(*String)
		if !ok {
			return newError("set_env: second argument must be a string, got %s", args[1].Type())
		}
		if err := os.Setenv(name.Value, value.Value); err != nil {
			return newError("set_env: %s", err.Error())
		}
		return &Nil{}
	}})

	// env_all() -- returns a hash of all environment variables
	env.Set("env_all", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 0 {
			return newError("env_all: expected 0 arguments, got %d", len(args))
		}
		pairs := make(map[string]Object)
		order := []string{}
		for _, entry := range os.Environ() {
			parts := strings.SplitN(entry, "=", 2)
			if len(parts) == 2 {
				pairs[parts[0]] = &String{Value: parts[1]}
				order = append(order, parts[0])
			}
		}
		return &Hash{Pairs: pairs, Order: order}
	}})

	// -----------------------------------------------------------------------
	// Process Execution
	// -----------------------------------------------------------------------

	// exec(command) -- execute a shell command, returns hash with stdout, stderr, status
	env.Set("exec", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("exec: expected 1 argument, got %d", len(args))
		}
		cmdStr, ok := args[0].(*String)
		if !ok {
			return newError("exec: argument must be a string, got %s", args[0].Type())
		}

		cmd := exec.Command("sh", "-c", cmdStr.Value)
		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				return newError("exec: %s", err.Error())
			}
		}

		pairs := map[string]Object{
			"stdout": &String{Value: stdout.String()},
			"stderr": &String{Value: stderr.String()},
			"status": &Integer{Value: int64(exitCode)},
		}
		return &Hash{
			Pairs: pairs,
			Order: []string{"stdout", "stderr", "status"},
		}
	}})

	// shell(command) -- execute a shell command, returns stdout, throws on non-zero exit
	env.Set("shell", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("shell: expected 1 argument, got %d", len(args))
		}
		cmdStr, ok := args[0].(*String)
		if !ok {
			return newError("shell: argument must be a string, got %s", args[0].Type())
		}

		cmd := exec.Command("sh", "-c", cmdStr.Value)
		output, err := cmd.Output()
		if err != nil {
			return newError("shell: command failed: %s", err.Error())
		}
		return &String{Value: string(output)}
	}})

	// -----------------------------------------------------------------------
	// Advanced File I/O
	// -----------------------------------------------------------------------

	// append_file(path, content) -- append content to a file
	env.Set("append_file", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 2 {
			return newError("append_file: expected 2 arguments, got %d", len(args))
		}
		path, ok := args[0].(*String)
		if !ok {
			return newError("append_file: first argument must be a string, got %s", args[0].Type())
		}
		content, ok := args[1].(*String)
		if !ok {
			return newError("append_file: second argument must be a string, got %s", args[1].Type())
		}
		f, err := os.OpenFile(path.Value, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return newError("append_file: %s", err.Error())
		}
		defer f.Close()
		if _, err := f.WriteString(content.Value); err != nil {
			return newError("append_file: %s", err.Error())
		}
		return &Nil{}
	}})

	// delete_file(path) -- delete a file
	env.Set("delete_file", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("delete_file: expected 1 argument, got %d", len(args))
		}
		path, ok := args[0].(*String)
		if !ok {
			return newError("delete_file: first argument must be a string, got %s", args[0].Type())
		}
		if err := os.Remove(path.Value); err != nil {
			return newError("delete_file: %s", err.Error())
		}
		return &Nil{}
	}})

	// rename_file(old, new) -- rename/move a file
	env.Set("rename_file", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 2 {
			return newError("rename_file: expected 2 arguments, got %d", len(args))
		}
		oldPath, ok := args[0].(*String)
		if !ok {
			return newError("rename_file: first argument must be a string, got %s", args[0].Type())
		}
		newPath, ok := args[1].(*String)
		if !ok {
			return newError("rename_file: second argument must be a string, got %s", args[1].Type())
		}
		if err := os.Rename(oldPath.Value, newPath.Value); err != nil {
			return newError("rename_file: %s", err.Error())
		}
		return &Nil{}
	}})

	// list_dir(path) -- list files in a directory
	env.Set("list_dir", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("list_dir: expected 1 argument, got %d", len(args))
		}
		path, ok := args[0].(*String)
		if !ok {
			return newError("list_dir: argument must be a string, got %s", args[0].Type())
		}
		entries, err := os.ReadDir(path.Value)
		if err != nil {
			return newError("list_dir: %s", err.Error())
		}
		elements := make([]Object, 0, len(entries))
		for _, entry := range entries {
			elements = append(elements, &String{Value: entry.Name()})
		}
		return &Array{Elements: elements}
	}})

	// dir_exists(path) -- check if a directory exists
	env.Set("dir_exists", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("dir_exists: expected 1 argument, got %d", len(args))
		}
		path, ok := args[0].(*String)
		if !ok {
			return newError("dir_exists: argument must be a string, got %s", args[0].Type())
		}
		info, err := os.Stat(path.Value)
		if err != nil {
			return &Boolean{Value: false}
		}
		return &Boolean{Value: info.IsDir()}
	}})

	// mkdir(path) -- create a directory
	env.Set("mkdir", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("mkdir: expected 1 argument, got %d", len(args))
		}
		path, ok := args[0].(*String)
		if !ok {
			return newError("mkdir: argument must be a string, got %s", args[0].Type())
		}
		if err := os.Mkdir(path.Value, 0755); err != nil {
			return newError("mkdir: %s", err.Error())
		}
		return &Nil{}
	}})

	// mkdir_p(path) -- create a directory and all parents
	env.Set("mkdir_p", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("mkdir_p: expected 1 argument, got %d", len(args))
		}
		path, ok := args[0].(*String)
		if !ok {
			return newError("mkdir_p: argument must be a string, got %s", args[0].Type())
		}
		if err := os.MkdirAll(path.Value, 0755); err != nil {
			return newError("mkdir_p: %s", err.Error())
		}
		return &Nil{}
	}})

	// getcwd() -- get current working directory
	env.Set("getcwd", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 0 {
			return newError("getcwd: expected 0 arguments, got %d", len(args))
		}
		dir, err := os.Getwd()
		if err != nil {
			return newError("getcwd: %s", err.Error())
		}
		return &String{Value: dir}
	}})

	// -----------------------------------------------------------------------
	// Path Operations
	// -----------------------------------------------------------------------

	// path_base(path) -- returns the last element of a path
	env.Set("path_base", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("path_base: expected 1 argument, got %d", len(args))
		}
		p, ok := args[0].(*String)
		if !ok {
			return newError("path_base: argument must be a string, got %s", args[0].Type())
		}
		return &String{Value: filepath.Base(p.Value)}
	}})

	// path_dir(path) -- returns the directory portion of a path
	env.Set("path_dir", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("path_dir: expected 1 argument, got %d", len(args))
		}
		p, ok := args[0].(*String)
		if !ok {
			return newError("path_dir: argument must be a string, got %s", args[0].Type())
		}
		return &String{Value: filepath.Dir(p.Value)}
	}})

	// path_ext(path) -- returns the file extension
	env.Set("path_ext", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("path_ext: expected 1 argument, got %d", len(args))
		}
		p, ok := args[0].(*String)
		if !ok {
			return newError("path_ext: argument must be a string, got %s", args[0].Type())
		}
		return &String{Value: filepath.Ext(p.Value)}
	}})

	// path_join(parts...) -- join path elements
	env.Set("path_join", &Builtin{Fn: func(args ...Object) Object {
		if len(args) == 0 {
			return newError("path_join: expected at least 1 argument")
		}
		parts := make([]string, 0, len(args))
		for _, arg := range args {
			s, ok := arg.(*String)
			if !ok {
				return newError("path_join: all arguments must be strings, got %s", arg.Type())
			}
			parts = append(parts, s.Value)
		}
		return &String{Value: filepath.Join(parts...)}
	}})

	// -----------------------------------------------------------------------
	// CLI Arguments (set by the interpreter at startup)
	// -----------------------------------------------------------------------

	// ARGV is set externally via SetArgs; default to empty array
	if _, ok := env.Get("ARGV"); !ok {
		env.Set("ARGV", &Array{Elements: []Object{}})
	}

	// -----------------------------------------------------------------------
	// exit(code) -- exit the process
	// -----------------------------------------------------------------------

	env.Set("exit", &Builtin{Fn: func(args ...Object) Object {
		code := 0
		if len(args) == 1 {
			if c, ok := args[0].(*Integer); ok {
				code = int(c.Value)
			}
		}
		os.Exit(code)
		return &Nil{}
	}})

	// Suppress unused import warnings
	_ = fmt.Sprintf
}
