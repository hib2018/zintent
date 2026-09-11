package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/hib2018/zintent/tui/internal/output"
	"github.com/hib2018/zintent/tui/internal/protocol"
	"github.com/hib2018/zintent/tui/internal/runner"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCode(err))
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("zintent", flag.ContinueOnError)
	corePath := fs.String("core", findCore(), "path to zintent-core")
	jsonMode := fs.Bool("json", false, "write JSON response")
	actorID := fs.String("actor-id", "", "fallback local actor ID")
	if err := fs.Parse(args); err != nil {
		return err
	}
	operation := "protocol_info"
	if fs.NArg() > 0 {
		operation = fs.Arg(0)
	}
	if operation != "protocol_info" {
		return fmt.Errorf("command %q is not available in the foundation build", operation)
	}
	if *actorID == "" {
		if current, err := user.Current(); err == nil {
			*actorID = current.Username
		}
	}
	payload, _ := json.Marshal(map[string]string{"operation": operation})
	req := protocol.Request{ProtocolVersion: protocol.Version, RequestID: fmt.Sprintf("cli-%d", time.Now().UnixNano()), Operation: operation, PayloadSchema: "zintent.command/1", Payload: payload}
	response, err := (runner.Core{Executable: *corePath}).Run(context.Background(), req)
	if err != nil {
		return err
	}
	return output.Write(os.Stdout, response, *jsonMode)
}

func findCore() string {
	if value := os.Getenv("ZINTENT_CORE"); value != "" {
		return value
	}
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "zintent-core")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "../zig-out/bin/zintent-core"
}

func exitCode(err error) int {
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	return 70
}
