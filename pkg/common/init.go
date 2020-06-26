package common

// Conf Loaded subscriptionId
var Conf Config

//EnvConf Loaded environment config
var EnvConf Config

// Client Authorized Azure Client
var Client AzureClient

func init() {
	/*
		Conf = LoadConfig()
		fmt.Println(Conf)
		if err := config.ParseEnvironment(); err != nil {
			log.Fatalf("failed to parse environment: %s\n", err)
		}

		sub = config.SubscriptionID()
		grouname = config.GroupName()
		fmt.Println(sub)
		Client = NewClient()
	*/
}
