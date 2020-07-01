package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cerence/azure-request-limitometer/internal/config"
	"github.com/cerence/azure-request-limitometer/pkg/common"
	"github.com/cerence/azure-request-limitometer/pkg/outputs"

	"github.com/golang/glog"
	flag "github.com/spf13/pflag"
)

var azureClient = common.Client

const (
	cliName        = "limitometer"
	cliDescription = "Collects the number of remaining requests in Azure Resource Manager"
	cliVersion     = "2.0.0"
)

var (
	nodename     = flag.String("node", "", "Valid node in the resource group to create compute queries. Environment Variable: NODE_NAME")
	target       = flag.String("output", "pushgateway", "Target output for the limitometer, supported values are: [influxdb|pushgateway]")
	mode         = flag.String("mode", "oneshot", "Operational mode for limitometer, supported values are: [oneshot|service]")
	pollInterval = flag.Int("poll-interval", 60, "Only for 'service' mode: Poll interval for refreshing metrics in seconds")
	configSource = flag.String("config", "metadata", "To decide from where to load config, supported values are: [metadata|environment]")
)

func printUsage() {
	if flag.Args()[0] == "help" {
		fmt.Printf("%s\n\n", cliName)
		fmt.Println(cliDescription)
		flag.PrintDefaults()
		os.Exit(2)
	}
}

func printHelp() {
	if flag.Args()[0] == "version" {
		fmt.Printf("%s version %s\n", cliName, cliVersion)
		os.Exit(0)
	}
}

func getValuesAndWriteToOutput(nodename string) {
	log.Printf("Querying Azure API for remaining requests")
	requestsRemaining := getRequestsRemaining(nodename)

	log.Printf("Writing to database: %s", *target)
	if strings.ToLower(*target) == "influxdb" {
		outputs.WriteOutputInflux(requestsRemaining, "requestRemaining")
	} else if strings.ToLower(*target) == "pushgateway" {
		outputs.WriteOutputPushGateway(requestsRemaining)
	} else {
		glog.Exit("Did not provide a output through -output flag. Exiting.")
	}
}

func main() {
	flag.Parse()

	if len(flag.Args()) > 0 {
		printHelp()
		printUsage()
	}

	env, exists := os.LookupEnv("NODE_NAME")
	if exists {
		*nodename = env
	}
	confval, exists := os.LookupEnv("LIMITOMETER_CONFIG")
	if exists {
		*configSource = confval
	}

	if strings.ToLower(*configSource) == "metadata" {
		common.Client = common.NewClient("metadata")
	} else if strings.ToLower(*configSource) == "environment" {
		if err := config.ParseEnvironment(); err != nil {
			log.Fatalf("failed to parse environment: %s\n", err)
		}
		common.Client = common.NewClient("environment")

	} else {
		glog.Exit("Did not provide a output through -output flag. Exiting.")
	}

	log.Printf("Starting limitometer with %s as target VM", *nodename)
	if strings.ToLower(*mode) == "oneshot" {
		log.Printf("Running in oneshot mode, will get remaining requests once and exit afterwards")
		getValuesAndWriteToOutput(*nodename)
		os.Exit(0)
	} else if strings.ToLower(*mode) == "service" {
		log.Printf("Running in service mode, will poll Azure API every %d seconds", *pollInterval)

		// set up signal channel to manage SIGINT and SIGTERM
		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			for {
				getValuesAndWriteToOutput(*nodename)
				time.Sleep(time.Duration(*pollInterval) * time.Second)
			}
		}()

		<-done
		log.Printf("Received signal to stop. Shutting down.")
		os.Exit(0)
	} else {
		glog.Exit("Did not provide a valid operations mode through -mode flag. Exiting.")
	}

}
