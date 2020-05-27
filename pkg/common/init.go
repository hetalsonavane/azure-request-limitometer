package common

// Conf Loaded Configuration from azure.json

// Client Authorized Azure Client
var Client AzureClient

func init() {

	Client = NewClient()
}
