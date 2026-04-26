package main

// launcher is the container entrypoint. One image contains all demo programs;
// set APP=registry, APP=tenant-lookup, or APP=svc1...svc15 to choose which one
// this Pod should run.

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func main() {
	app := strings.TrimSpace(os.Getenv("APP"))
	if app == "" && len(os.Args) > 1 {
		app = strings.TrimSpace(os.Args[1])
	}
	if app == "" {
		app = "registry"
	}
	if !validApp(app) {
		log.Fatalf("invalid APP %q", app)
	}

	path := filepath.Join("/app/bin", app)
	args := []string{path}
	if len(os.Args) > 2 {
		args = append(args, os.Args[2:]...)
	}

	if err := syscall.Exec(path, args, os.Environ()); err != nil {
		cmd := exec.Command(path, args[1:]...)
		cmd.Env = os.Environ()
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if runErr := cmd.Run(); runErr != nil {
			log.Fatalf("run %s: %v", app, runErr)
		}
	}
}

func validApp(app string) bool {
	if app == "registry" || app == "tenant-lookup" {
		return true
	}
	if !strings.HasPrefix(app, "svc") {
		return false
	}
	n, err := strconv.Atoi(strings.TrimPrefix(app, "svc"))
	if err != nil {
		return false
	}
	return n >= 1 && n <= 15
}
