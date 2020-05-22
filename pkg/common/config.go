package common

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/Azure/go-autorest/autorest/azure"
)

const apiVersion = "2018-10-01"
const azureInstanceMetadataEndpoint = "http://169.254.169.254/metadata/instance"

// Queries the Azure Instance Metadata Service for the instance's compute metadata
func retrieveComputeInstanceMetadata() (metadata ComputeInstanceMetadata, err error) {
	c := &http.Client{}

	req, _ := http.NewRequest("GET", azureInstanceMetadataEndpoint+"/compute", nil)
	req.Header.Add("Metadata", "True")
	q := req.URL.Query()
	q.Add("format", "json")
	q.Add("api-version", apiVersion)
	req.URL.RawQuery = q.Encode()

	resp, err := c.Do(req)
	if err != nil {
		err = fmt.Errorf("sending Azure Instance Metadata Service request failed: %v", err)
	}
	defer resp.Body.Close()

	rawJSON, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		err = fmt.Errorf("reading response body failed: %v", err)
		return
	}
	if err := json.Unmarshal(rawJSON, &metadata); err != nil {
		err = fmt.Errorf("unmarshaling JSON response failed: %v", err)
	}

	return
}

func retrieveenvdata() JsonData {

	config := JsonData{
		Name:              os.Getenv("Name"),
		SubscriptionID:    os.Getenv("SubscriptionID"),
		Location:          os.Getenv("Location"),
		ResourceGroupName: os.Getenv("ResourceGroupName"),
		Environment:       os.Getenv("Environment"),
	}
	return config
}

// LoadConfig Returns a Config struct created from Environment Variables
func LoadConfig() (config Config) {
	m := retrieveenvdata()

	env, err := azure.EnvironmentFromName(m.Environment)
	if err != nil {
		err = fmt.Errorf("Could not get environment object from metadata name: %v", err)
	}
	config = Config{
		VMName:              m.Name,
		SubscriptionID:      m.SubscriptionID,
		Location:            m.Location,
		ResourceGroup:       m.ResourceGroupName,
		AzureEnvironment:    m.Environment,
		EnvironmentEndpoint: env.ResourceManagerEndpoint,
	}

	return
}
