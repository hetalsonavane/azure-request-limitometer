package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest"
)

// Example Request Headers:
// 'x-ms-ratelimit-remaining-resource': 'Microsoft.Compute/HighCostGet3Min;133,Microsoft.Compute/HighCostGet30Min;657'
// 'x-ms-ratelimit-remaining-resource': 'Microsoft.Compute/LowCostGet3Min;3989,Microsoft.Compute/LowCostGet30Min;31790'
// 'x-ms-ratelimit-remaining-resource': 'Microsoft.Compute/PutVM3Min;740,Microsoft.Compute/PutVM30Min;3695'
// `X-Ms-Ratelimit-Remaining-Subscription-Reads: [11535]`

var expectedHeaderField = "X-Ms-Ratelimit-Remaining-Resource"
var expectedHeaderFormat = regexp.MustCompile(`(Microsoft.\w+\/\w+);(\d+)`)
var expectedSubIDReadsHeaderField = "X-Ms-Ratelimit-Remaining-Subscription-Reads"
var subIDReadsHeader = "SubIDReads"

func getRequestsRemaining(nodename string) (requestsRemaining map[string]int) {
	requestsRemaining = make(map[string]int)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	Getvm, err := azureClient.GetVM(ctx, nodename)
	if err != nil {
		log.Printf("failed to get vm: %s\n", err)
	}

	Getnic, err := azureClient.GetNicFromVMName(nodename)
	if err != nil {
		log.Printf("failed to get nic: %s\n", err)
	}

	GetAllVM := azureClient.GetAllVM()
	//putvm := azureClient.PutVM(nodename)
	GetAllNic := azureClient.GetAllNics()
	Getlb, err := azureClient.GetAllLoadBalancer()
	if err != nil {
		log.Printf("failed to get nic: %s\n", err)
	}
	responses := []autorest.Response{

		Getvm.Response,
		Getnic.Response,
		Getlb.Response().Response,
		GetAllVM.Response().Response,
		GetAllNic.Response().Response,
		//putvm,
	}
	fmt.Println(responses)

	for _, response := range responses {
		if response.StatusCode != 200 {
			log.Fatalf("Response did not return a StatusCode of 200. StatusCode: %d", response.StatusCode)
		}
		for k, v := range extractRequestsRemaining(response.Header) {
			requestsRemaining[k] = v
		}
		for k, v := range extractSubIDRequestsRemaining(response.Header) {
			requestsRemaining[k] = v
		}
	}

	return
}

func extractRequestsRemaining(h http.Header) (requestsRemaining map[string]int) {
	requestsRemaining = map[string]int{}

	headerSubfields := strings.Split(h.Get(expectedHeaderField), ",")

	for _, field := range headerSubfields {

		matches := expectedHeaderFormat.FindStringSubmatch(field)
		if !(len(matches) == 3) {
			continue
		}

		requestType := matches[1]
		requestsLeft, err := strconv.Atoi(matches[2])
		if err != nil {
			log.Fatal(err)
		}
		requestsRemaining[requestType] = requestsLeft
	}

	return requestsRemaining
}

func extractSubIDRequestsRemaining(h http.Header) (requestsRemaining map[string]int) {
	requestsRemaining = map[string]int{}
	subIDReadsHeaderField := h.Get(expectedSubIDReadsHeaderField)
	if subIDReadsHeaderField != "" {
		requestLeft, err := strconv.Atoi(subIDReadsHeaderField)
		if err != nil {
			log.Fatal(err)
		}
		requestsRemaining[subIDReadsHeader] = requestLeft
	}
	return requestsRemaining
}
