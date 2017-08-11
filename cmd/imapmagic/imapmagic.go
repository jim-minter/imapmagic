package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	goimap "github.com/emersion/go-imap"
	"github.com/jim-minter/imapmagic/pkg/config"
	"github.com/jim-minter/imapmagic/pkg/imap"
	"github.com/jim-minter/imapmagic/pkg/rules"
)

var debug = flag.Bool("debug", false, "enable debugging")
var dryRun = flag.Bool("dry-run", false, "dry run")
var configFile = flag.String("config", filepath.Join(os.Getenv("HOME"), ".imapmagic"), "path to config file")

func run(ctx context.Context) error {
	config, err := config.Read(*configFile)
	if err != nil {
		return err
	}

	c, err := imap.Connect(config, *debug)
	if err != nil {
		return err
	}
	defer c.Logout()

	err = c.Select("INBOX", false)
	if err != nil {
		return err
	}

	for {
		seqset := &goimap.SeqSet{}

		for _, rule := range []rules.Rule{rules.CIRobot, rules.MergeRobot, rules.RobotCommands, rules.OpenshiftBot} {
			set := rule(config, c)
			seqset.AddSet(&set)
		}

		if !seqset.Empty() {
			if *dryRun {
				fmt.Fprintf(os.Stdout, "would move %#v\n", seqset.String())
			} else {
				err := c.Move(seqset, config.MoveTo)
				if err != nil {
					return err
				}
			}
		}

		err := c.Idle(ctx)
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		default:
		}
	}
}

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())

	ch := make(chan os.Signal, 1)
	go func() {
		<-ch
		cancel()
	}()
	signal.Notify(ch, os.Interrupt)

	err := run(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}
}
