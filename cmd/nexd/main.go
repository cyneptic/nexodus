package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"

	"github.com/nexodus-io/nexodus/internal/nexodus"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

const (
	nexodusLogEnv = "NEXD_LOGLEVEL"
)

type nexdMode int

const (
	nexdModeAgent nexdMode = iota
	nexdModeProxy
)

// This variable is set using ldflags at build time. See Makefile for details.
var Version = "dev"

func nexdRun(cCtx *cli.Context, logger *zap.Logger, mode nexdMode) error {
	controller := cCtx.Args().First()
	if controller == "" {
		logger.Info("<controller-url> required")
		return nil
	}

	_, err := nexodus.CtlStatus(cCtx)
	if err == nil {
		return fmt.Errorf("existing nexd service already running")
	}

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)

	userspaceMode := false
	relayNode := false
	discoveryNode := false
	var childPrefix []string
	switch mode {
	case nexdModeAgent:
		childPrefix = cCtx.StringSlice("child-prefix")
		relayNode = cCtx.Bool("relay-node")
		discoveryNode = cCtx.Bool("discovery-node")
	case nexdModeProxy:
		// These are top-level flags that are not compatible with `nexd proxy`.
		// It would be nicer to convert these to subcommands, but that's for
		// another day. TODO
		proxyBadFlags := [3]string{"child-prefix", "relay-node", "discovery-node"}
		for _, flag := range proxyBadFlags {
			if cCtx.IsSet(flag) {
				return fmt.Errorf("flag %s is not compatible with `nexd proxy`", flag)
			}
		}
		userspaceMode = true
		logger.Info("Starting in L4 proxy mode")
	}

	nex, err := nexodus.NewNexodus(
		logger.Sugar(),
		controller,
		cCtx.String("username"),
		cCtx.String("password"),
		cCtx.Int("listen-port"),
		cCtx.String("public-key"),
		cCtx.String("private-key"),
		cCtx.String("request-ip"),
		cCtx.String("local-endpoint-ip"),
		childPrefix,
		cCtx.Bool("stun"),
		relayNode,
		discoveryNode,
		cCtx.Bool("relay-only"),
		cCtx.Bool("insecure-skip-tls-verify"),
		Version,
		userspaceMode,
		cCtx.String("state-dir"),
	)
	if err != nil {
		logger.Fatal(err.Error())
	}

	wg := &sync.WaitGroup{}
	for _, egressRule := range cCtx.StringSlice("egress") {
		err := nex.UserspaceProxyAdd(ctx, wg, egressRule, nexodus.ProxyTypeEgress)
		if err != nil {
			logger.Sugar().Errorf("Failed to add egress proxy rule (%s): %v", egressRule, err)
		}
	}
	for _, ingressRule := range cCtx.StringSlice("ingress") {
		err := nex.UserspaceProxyAdd(ctx, wg, ingressRule, nexodus.ProxyTypeIngress)
		if err != nil {
			logger.Sugar().Errorf("Failed to add ingress proxy rule (%s): %v", ingressRule, err)
		}
	}
	if err := nex.Start(ctx, wg); err != nil {
		logger.Fatal(err.Error())
	}
	wg.Wait()

	return nil
}

var additionalPlatformFlags []cli.Flag = nil

func main() {
	// set the log level
	debug := os.Getenv(nexodusLogEnv)
	var logger *zap.Logger
	var err error
	if debug != "" {
		logger, err = zap.NewDevelopment()
		logger.Info("Debug logging enabled")
	} else {
		logCfg := zap.NewProductionConfig()
		logCfg.DisableStacktrace = true
		logger, err = logCfg.Build()
	}
	if err != nil {
		logger.Fatal(err.Error())
	}

	// Overwrite usage to capitalize "Show"
	cli.HelpFlag.(*cli.BoolFlag).Usage = "Show help"
	// flags are stored in the global flags variable
	app := &cli.App{
		Name:      "nexd",
		Usage:     "Node agent to configure encrypted mesh networking with nexodus.",
		ArgsUsage: "controller-url",
		Commands: []*cli.Command{
			{
				Name:  "version",
				Usage: "Get the version of nexd",
				Action: func(cCtx *cli.Context) error {
					fmt.Printf("version: %s\n", Version)
					return nil
				},
			},
			{
				Name:  "proxy",
				Usage: "Run nexd as an L4 proxy instead of creating a network interface",
				Action: func(cCtx *cli.Context) error {
					return nexdRun(cCtx, logger, nexdModeProxy)
				},
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:     "ingress",
						Usage:    "Forward connections from the Nexodus network made to [port] on this proxy instance to port [destination_port] at [destination_ip] via a locally accessible network using a `value` in the form: protocol:port:destination_ip:destination_port. All fields are required.",
						Required: false,
					},
					&cli.StringSliceFlag{
						Name:     "egress",
						Usage:    "Forward connections from a locally accessible network made to [port] on this proxy instance to port [destination_port] at [destination_ip] via the Nexodus network using a `value` in the form: protocol:port:destination_ip:destination_port. All fields are required.",
						Required: false,
					},
				},
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "public-key",
				Value:    "",
				Usage:    "Base64 encoded public `key` for the local host - agent generates keys by default",
				EnvVars:  []string{"NEXD_PUB_KEY"},
				Required: false,
			},
			&cli.StringFlag{
				Name:     "private-key",
				Value:    "",
				Usage:    "Base64 encoded private `key` for the local host (dev purposes only - soon to be removed)",
				EnvVars:  []string{"NEXD_PRIVATE_KEY"},
				Required: false,
			},
			&cli.IntFlag{
				Name:     "listen-port",
				Value:    0,
				Usage:    "Wireguard `port` to listen on for incoming peers",
				EnvVars:  []string{"NEXD_LISTEN_PORT"},
				Required: false,
			},
			&cli.StringFlag{
				Name:     "request-ip",
				Value:    "",
				Usage:    "Request a specific `IPv4` address from IPAM if available (optional)",
				EnvVars:  []string{"NEXD_REQUESTED_IP"},
				Required: false,
			},
			&cli.StringFlag{
				Name:     "local-endpoint-ip",
				Value:    "",
				Usage:    "Specify the endpoint `IP` address of this node instead of being discovered (optional)",
				EnvVars:  []string{"NEXD_LOCAL_ENDPOINT_IP"},
				Required: false,
			},
			&cli.StringSliceFlag{
				Name:     "child-prefix",
				Usage:    "Request an `IPv4 CIDR` range of addresses that will be advertised from this node (optional)",
				EnvVars:  []string{"NEXD_REQUESTED_CHILD_PREFIX"},
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "stun",
				Usage:    "Discover the public address for this host using STUN",
				Value:    false,
				EnvVars:  []string{"NEXD_STUN"},
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "relay-node",
				Usage:    "Set if this node is to be the relay node for a hub and spoke scenarios",
				Value:    false,
				EnvVars:  []string{"NEXD_RELAY_NODE"},
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "discovery-node",
				Usage:    "Set if this node is to be the discovery node for NAT traversal in an organization",
				Value:    false,
				EnvVars:  []string{"NEXD_DISCOVERY_NODE"},
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "relay-only",
				Usage:    "Set if this node is unable to NAT hole punch in a hub zone (Nexodus will set this automatically if symmetric NAT is detected)",
				Value:    false,
				EnvVars:  []string{"NEXD_RELAY_ONLY"},
				Required: false,
			},
			&cli.StringFlag{
				Name:     "username",
				Value:    "",
				Usage:    "Username `string` for accessing the nexodus service",
				EnvVars:  []string{"NEXD_USERNAME"},
				Required: false,
			},
			&cli.StringFlag{
				Name:     "password",
				Value:    "",
				Usage:    "Password `string` for accessing the nexodus service",
				EnvVars:  []string{"NEXD_PASSWORD"},
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "insecure-skip-tls-verify",
				Value:    false,
				Usage:    "If true, server certificates will not be checked for validity. This will make your HTTPS connections insecure",
				EnvVars:  []string{"NEXD_INSECURE_SKIP_TLS_VERIFY"},
				Required: false,
			},
			&cli.StringFlag{
				Name:    "state-dir",
				Usage:   "Directory to store state in",
				Value:   "",
				EnvVars: []string{"NEXD_STATE_DIR"},
			},
		},
		Action: func(cCtx *cli.Context) error {
			return nexdRun(cCtx, logger, nexdModeAgent)
		},
	}

	app.Flags = append(app.Flags, additionalPlatformFlags...)
	sort.Slice(app.Flags, func(i, j int) bool {
		return app.Flags[i].Names()[0] < app.Flags[j].Names()[0]
	})

	if err := app.Run(os.Args); err != nil {
		logger.Fatal(err.Error())
	}
}
