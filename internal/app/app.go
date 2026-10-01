// Package app wires the parts together: it reads flags and the config file,
// builds the store, responder and server, and then runs either the screen or
// the headless mode.
package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tejjasdev/portbell/internal/brand"
	"github.com/tejjasdev/portbell/internal/config"
	"github.com/tejjasdev/portbell/internal/events"
	"github.com/tejjasdev/portbell/internal/headless"
	"github.com/tejjasdev/portbell/internal/responder"
	"github.com/tejjasdev/portbell/internal/server"
	"github.com/tejjasdev/portbell/internal/tui"
	"github.com/tejjasdev/portbell/internal/version"
)

// Exit codes.
const (
	ExitOK      = 0
	ExitRuntime = 1 // the tool could not run, for example the port is busy
	ExitUsage   = 2 // bad flags or a bad config file
)

type flags struct {
	port     int
	bind     string
	config   string
	response string
	status   int
	delay    time.Duration
	cors     bool
	maxBody  int64
	headless bool
	version  bool
}

func usage() string {
	n := brand.Name
	return fmt.Sprintf(`%[1]s: see every HTTP request that reaches your machine, and choose the reply.

Usage:
  %[1]s [flags]

Flags:
  -p, --port N          port to listen on (default 8080)
  -b, --bind ADDR       address to bind (default 127.0.0.1; 0.0.0.0 opens it to your network)
  -c, --config FILE     config file (default ./%[2]s, then your user config folder)
  -r, --response FILE   reply to send, written like an HTTP response
  -s, --status CODE     reply status code (default 200)
  -d, --delay TIME      wait before replying, for example 500ms or 2s
      --cors            answer CORS preflight requests and add CORS headers
      --max-body BYTES  largest request body to keep (default 10485760)
      --headless        no screen: print one JSON line per request on stdout
  -v, --version         print the version
  -h, --help            show this help

Examples:
  %[1]s                          start on http://127.0.0.1:8080
  %[1]s -p 9000 -s 503 -d 2s     answer every request with a slow 503
  %[1]s --headless | jq .        print requests as JSON lines

Exit codes: 0 success, 1 could not run (for example the port is busy), 2 bad flags or config.
Docs: %[3]s
`, n, brand.ConfigFile, brand.URL)
}

func parse(args []string, stdout, stderr io.Writer) (f flags, set map[string]bool, code int, stop bool) {
	fs := flag.NewFlagSet(brand.Name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.IntVar(&f.port, "p", 0, "")
	fs.IntVar(&f.port, "port", 0, "")
	fs.StringVar(&f.bind, "b", "", "")
	fs.StringVar(&f.bind, "bind", "", "")
	fs.StringVar(&f.config, "c", "", "")
	fs.StringVar(&f.config, "config", "", "")
	fs.StringVar(&f.response, "r", "", "")
	fs.StringVar(&f.response, "response", "", "")
	fs.IntVar(&f.status, "s", 0, "")
	fs.IntVar(&f.status, "status", 0, "")
	fs.DurationVar(&f.delay, "d", 0, "")
	fs.DurationVar(&f.delay, "delay", 0, "")
	fs.BoolVar(&f.cors, "cors", false, "")
	fs.Int64Var(&f.maxBody, "max-body", 0, "")
	fs.BoolVar(&f.headless, "headless", false, "")
	fs.BoolVar(&f.version, "v", false, "")
	fs.BoolVar(&f.version, "version", false, "")
	help := fs.Bool("h", false, "")
	fs.BoolVar(help, "help", false, "")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, usage())
			return f, nil, ExitOK, true
		}
		fmt.Fprintf(stderr, "%s: %v\nRun '%s --help' for usage.\n", brand.Name, err, brand.Name)
		return f, nil, ExitUsage, true
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "%s: unexpected argument %q\nRun '%s --help' for usage.\n", brand.Name, fs.Arg(0), brand.Name)
		return f, nil, ExitUsage, true
	}
	if *help {
		fmt.Fprint(stdout, usage())
		return f, nil, ExitOK, true
	}
	set = map[string]bool{}
	fs.Visit(func(fl *flag.Flag) { set[fl.Name] = true })
	return f, set, ExitOK, false
}

// Run is the program. It returns the process exit code.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	f, set, code, stop := parse(args, stdout, stderr)
	if stop {
		return code
	}
	if f.version {
		fmt.Fprintf(stdout, "%s %s\n", brand.Name, version.String())
		return ExitOK
	}

	cfg, spec, err := load(f, set)
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", brand.Name, err)
		return ExitUsage
	}

	store := events.NewStore(cfg.Server.History)
	resp := responder.New(spec, cfg.EnvPolicy())
	srv := server.New(server.Options{Addr: cfg.Addr(), MaxBody: cfg.Server.MaxBodyBytes, CORS: cfg.Server.CORS}, store, resp)

	addr, err := srv.Listen()
	if err != nil {
		if server.IsAddrInUse(err) {
			fmt.Fprintf(stderr, "%s: port %d is already in use.\nTry another one, for example: %s -p %d\n",
				brand.Name, cfg.Server.Port, brand.Name, cfg.Server.Port+1)
		} else {
			fmt.Fprintf(stderr, "%s: cannot listen on %s: %v\n", brand.Name, cfg.Addr(), err)
		}
		return ExitRuntime
	}
	url := server.Describe(addr)
	warning := ""
	if !cfg.IsLoopback() {
		warning = fmt.Sprintf("bound to %s: anyone on your network can send requests and read your replies", cfg.Server.Bind)
	}

	if !f.headless {
		if file, ok := stdout.(*os.File); ok && !isTerminal(file) {
			fmt.Fprintf(stderr, "%s: standard output is not a terminal. Use --headless to print JSON lines.\n", brand.Name)
			return ExitUsage
		}
	}

	sub, cancelSub := store.Subscribe()
	defer cancelSub()

	ctx, stopSignals := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	serveDone := make(chan error, 1)
	go func() {
		err := srv.Serve(ctx)
		serveDone <- err
		cancel() // if the server stops, stop the screen too
	}()

	var runErr error
	if f.headless {
		fmt.Fprintf(stderr, "%s %s listening on %s (headless: one JSON line per request on stdout)\n", brand.Name, version.Version, url)
		if warning != "" {
			fmt.Fprintf(stderr, "warning: %s\n", warning)
		}
		runErr = headless.Run(ctx, sub, stdout)
	} else {
		runErr = tui.Run(ctx, tui.Options{
			Store: store, Resp: resp, Sub: sub, URL: url, Warning: warning,
			Redact: cfg.Export.Redact, History: cfg.Server.History,
		})
	}
	cancel()

	serveErr := <-serveDone
	for _, err := range []error{runErr, serveErr} {
		if err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", brand.Name, err)
			return ExitRuntime
		}
	}
	return ExitOK
}

// load builds the final settings: defaults, then the config file, then flags.
func load(f flags, set map[string]bool) (config.Config, responder.Spec, error) {
	cfg := config.Defaults()
	path, found, err := config.Discover(f.config)
	if err != nil {
		return cfg, responder.Spec{}, err
	}
	if found {
		if cfg, err = config.Load(path); err != nil {
			return cfg, responder.Spec{}, err
		}
	}

	if set["p"] || set["port"] {
		cfg.Server.Port = f.port
	}
	if set["b"] || set["bind"] {
		cfg.Server.Bind = f.bind
	}
	if set["cors"] {
		cfg.Server.CORS = f.cors
	}
	if set["max-body"] {
		cfg.Server.MaxBodyBytes = f.maxBody
	}
	if err := cfg.Validate(); err != nil {
		return cfg, responder.Spec{}, err
	}

	spec, err := cfg.Spec()
	if err != nil {
		return cfg, spec, err
	}
	if set["r"] || set["response"] {
		file, err := os.Open(f.response)
		if err != nil {
			return cfg, spec, fmt.Errorf("reply file: %w", err)
		}
		defer file.Close()
		fromFile, err := responder.ParseHTTP(file)
		if err != nil {
			return cfg, spec, fmt.Errorf("reply file %s: %w", f.response, err)
		}
		fromFile.Delay = spec.Delay // files carry no delay
		spec = fromFile
	}
	if set["s"] || set["status"] {
		spec.Status = f.status
	}
	if set["d"] || set["delay"] {
		spec.Delay = f.delay
	}
	if err := spec.Validate(); err != nil {
		return cfg, spec, fmt.Errorf("reply: %w", err)
	}
	return cfg, spec, nil
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
