package common

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/cerence/azure-request-limitometer/internal/config"

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

// LoadConfig Returns a Config struct created from Environment Variables
func LoadConfig() (config Config) {
	m, err := retrieveComputeInstanceMetadata()
	if err != nil {
		err = fmt.Errorf("unable to load the config: %v", err)
	}

	env, err := azure.EnvironmentFromName(m.Environment)
	if err != nil {
		err = fmt.Errorf("Could not get environment object from metadata name: %v", err)
	}

	os.Setenv("AZURE_GROUP_NAME", m.ResourceGroupName)
	os.Setenv("AZURE_LOCATION_DEFAULT", m.Location)
	os.Setenv(" AZURE_SUBSCRIPTION_ID", m.SubscriptionID)
	os.Setenv("AZURE_USE_DEVICEFLOW", "true")
	os.Setenv("AZURE_SAMPLES_KEEP_RESOURCES", "true")
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

//EnvLoadConfig return object which load env config
func EnvLoadConfig() (envconfig Config) {

	envconfig = Config{

		SubscriptionID: config.SubscriptionID(),
		Location:       config.Location(),
		ResourceGroup:  config.GroupName(),
	}

	return
}
