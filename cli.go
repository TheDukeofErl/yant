// This file is part of the program "yant".
// Please see the LICENSE file for copyright information.

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/noisetorch/pulseaudio"
)

type CLIOpts struct {
	doLog       bool
	sourceName  string
	unload      bool
	loadInput   bool
	threshold   int
	list        bool
}

func parseCLIOpts() CLIOpts {
	var opt CLIOpts
	flag.BoolVar(&opt.doLog, "log", false, "Print debugging output to stdout")
	flag.StringVar(&opt.sourceName, "s", "", "Use the specified source device ID")
	flag.BoolVar(&opt.loadInput, "i", false, "Load supressor for input. If no source device ID is specified the default pulse audio source is used.")
	flag.BoolVar(&opt.unload, "u", false, "Unload supressor")
	flag.IntVar(&opt.threshold, "t", -1, "Voice activation threshold")
	flag.BoolVar(&opt.list, "l", false, "List available PulseAudio devices")
	flag.Parse()

	return opt
}

func doCLI(opt CLIOpts, config *config, librnnoise string) {
	paClient, err := pulseaudio.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't create pulseaudio client: %v\n", err)
		cleanupExit(librnnoise, 1)
	}
	defer paClient.Close()

	ctx := ntcontext{}

	info, err := serverInfo(paClient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't fetch audio server info: %s\n", err)
	}
	ctx.serverInfo = info

	ctx.config = config
	ctx.librnnoise = librnnoise

	ctx.paClient = paClient

	if opt.list {
		fmt.Println("Sources:")
		sources := getSources(&ctx, paClient)
		for i := range sources {
			fmt.Printf("\tDevice Name: %s\n\tDevice ID: %s\n\n", sources[i].Name, sources[i].ID)
		}

		cleanupExit(librnnoise, 0)
	}

	if opt.threshold > 0 {
		if opt.threshold > 95 {
			fmt.Fprintf(os.Stderr, "Threshold of '%d' too high, setting to maximum of 95.\n", opt.threshold)
			ctx.config.Threshold = 95
		} else {
			ctx.config.Threshold = opt.threshold
		}
	}

	if opt.unload {
		err := unloadSupressor(&ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error unloading PulseAudio Module: %+v\n", err)
			cleanupExit(librnnoise, 1)
		}
		cleanupExit(librnnoise, 0)
	}

	if opt.loadInput {
		sources := getSources(&ctx, paClient)

		if opt.sourceName == "" {
			defaultSource, err := getDefaultSourceID(paClient)
			if err != nil {
				fmt.Fprintf(os.Stderr, "No source specified to load and failed to load default source: %+v\n", err)
				cleanupExit(librnnoise, 1)
			}
			opt.sourceName = defaultSource
		}
		for i := range sources {
			if sources[i].ID == opt.sourceName {
				sources[i].checked = true
				err := loadSupressor(&ctx, &sources[i])
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error loading PulseAudio Module: %+v\n", err)
					cleanupExit(librnnoise, 1)
				}
				cleanupExit(librnnoise, 0)
			}
		}
		fmt.Fprintf(os.Stderr, "PulseAudio source not found: %s\n", opt.sourceName)
		cleanupExit(librnnoise, 1)

	}

}

func cleanupExit(librnnoise string, exitCode int) {
	removeLib(librnnoise)
	os.Exit(exitCode)
}
