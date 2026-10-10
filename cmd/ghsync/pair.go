package main

import (
	"context"
	"flag"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"koboapps/internal/ghsync"
	"koboapps/internal/i18n"
)

const (
	defaultPairPidFile    = "/tmp/ghsync-pair.pid"
	defaultPairInfoFile   = "/tmp/ghsync-pair.json"
	defaultPairResultFile = "/tmp/ghsync-pair-result"
)

func pairCommand(args []string, out io.Writer) int {
	flags := flag.NewFlagSet("pair", flag.ContinueOnError)
	flags.SetOutput(out)
	var common commonFlags
	common.register(flags)
	apiBase := flags.String("api-base", defaultAPIBase, "GitHub API base URL (tests)")
	address := flags.String("ip", "", "LAN address to bind (default: detected Wi-Fi address; never 0.0.0.0)")
	port := flags.Int("port", ghsync.DefaultPairPort, "TCP port")
	lifetime := flags.Duration("lifetime", ghsync.DefaultPairLifetime, "how long pairing stays open")
	pidFile := flags.String("pid-file", defaultPairPidFile, "where the running session records its process id")
	infoFile := flags.String("info-file", defaultPairInfoFile, "where the open session writes its address and code for the app screen")
	resultFile := flags.String("result-file", defaultPairResultFile, "where the session records how it ended, for the app screen")
	shouldStop := flags.Bool("stop", false, "stop a running pairing session")
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	messages := common.messages()
	if *shouldStop {
		return stopPairing(out, messages, *pidFile)
	}
	return runPairing(out, messages, common, ghsync.PairOptions{
		Paths:      ghsync.Paths{AppDir: common.appDir},
		APIBase:    *apiBase,
		IP:         *address,
		Port:       *port,
		Lifetime:   *lifetime,
		Backoff:    time.Second,
		InfoFile:   *infoFile,
		ResultFile: *resultFile,
	}, *pidFile)
}

func runPairing(out io.Writer, messages *i18n.Bundle, common commonFlags, options ghsync.PairOptions, pidFile string) int {
	logWriter, err := ghsync.NewLogFile(options.Paths.Log())
	if err != nil {
		say(out, messages, "msg.error", "message", messages.Describe(i18n.NewError("msg.err_log_unopenable").WithCause(err)))
		return 1
	}
	options.Log = ghsync.NewLogger(logWriter, "")
	options.Out = out
	options.Messages = messages
	options.Locale = pageLocale(common)
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		say(out, messages, "msg.error", "message", err.Error())
		return 1
	}
	defer os.Remove(pidFile)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	outcome, err := ghsync.Pair(ctx, options)
	if err != nil {
		say(out, messages, "msg.error", "message", messages.Describe(err))
		return 1
	}
	say(out, messages, pairResultKey(outcome), "repo", configuredRepo(options.Paths))
	if outcome == ghsync.PairPaired {
		return 0
	}
	return 1
}

// pageLocale is the language tag for the page; the message catalogs already follow the same choice.
func pageLocale(common commonFlags) string {
	if common.locale != "" {
		return common.locale
	}
	return strings.ReplaceAll(i18n.CurrentLocale(common.confPath), "_", "-")
}

func pairResultKey(outcome ghsync.PairOutcome) string {
	switch outcome {
	case ghsync.PairPaired:
		return "msg.pair_done"
	case ghsync.PairLocked:
		return "msg.pair_result_locked"
	case ghsync.PairExpired:
		return "msg.pair_result_expired"
	default:
		return "msg.pair_result_stopped"
	}
}

func configuredRepo(paths ghsync.Paths) string {
	config, err := ghsync.LoadConfig(paths.Config())
	if err != nil {
		return ""
	}
	return config.Repo
}

func stopPairing(out io.Writer, messages *i18n.Bundle, pidFile string) int {
	raw, err := os.ReadFile(pidFile)
	pid, parseErr := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || parseErr != nil {
		say(out, messages, "msg.pair_not_running")
		return 0
	}
	process, err := os.FindProcess(pid)
	if err == nil {
		err = process.Signal(syscall.SIGTERM)
	}
	if err != nil {
		os.Remove(pidFile)
		say(out, messages, "msg.pair_not_running")
		return 0
	}
	say(out, messages, "msg.pair_stopping")
	return 0
}
