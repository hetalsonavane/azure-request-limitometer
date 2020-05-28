package common

// Conf Loaded subscriptionId

// Client Authorized Azure Client
var Client AzureClient

func init() {
	Client = NewClient()
}
