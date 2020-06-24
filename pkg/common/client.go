package common

import (

	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"github.com/hetalsonavane/azure-request-limitometer/internal/config"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/hetalsonavane/azure-request-limitometer/internal/config"
)

// AzureClient This is an authorized client for Azure communication.
type AzureClient struct {
	compute.VirtualMachinesClient
	network.InterfacesClient
	network.LoadBalancersClient
}

// NewClient Initialized an authorized Azure client
func NewClient() (client AzureClient) {
	fmt.Println(config.ClientID())
	client = AzureClient{
		GetVmClient(),
		GetNicClient(),
		GetLbClient(),
	}
	return
}

//GetVmClient return vmClient
func GetVmClient() compute.VirtualMachinesClient {
	//fmt.Println(config.SubscriptionID())
	vmClient := compute.NewVirtualMachinesClient(config.SubscriptionID())
	a, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		log.Fatalf("failed to create authorizer from environment: %s\n", err)
	}
	vmClient.Authorizer = a
	vmClient.AddToUserAgent(config.UserAgent())
	return vmClient
}

// GetNicClient return nic client
func GetNicClient() network.InterfacesClient {
	nicClient := network.NewInterfacesClient(config.SubscriptionID())
	a, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		log.Fatalf("failed to create authorizer from environment: %s\n", err)
	}
	nicClient.Authorizer = a
	nicClient.AddToUserAgent(config.UserAgent())
	return nicClient
}

// GetLbClient return LB client
func GetLbClient() network.LoadBalancersClient {
	lbClient := network.NewLoadBalancersClient(config.SubscriptionID())
	a, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		log.Fatalf("failed to create authorizer from environment: %s\n", err)
	}
	lbClient.Authorizer = a
	lbClient.AddToUserAgent(config.UserAgent())
	return lbClient
}

// GetVM Returns a VirtualMachine object.
func (az AzureClient) GetVM(ctx context.Context, nodename string) (compute.VirtualMachine, error) {
	client := GetVmClient()
	//	fmt.Printf("VM")
	return client.Get(ctx, config.GroupName(), nodename, compute.InstanceView)
}

// GetAllLoadBalancer return info on a loadbalancer
func (az AzureClient) GetAllLoadBalancer() (network.LoadBalancerListResultPage, error) {
	lbClient := GetLbClient()
	ctx, cancel := context.WithTimeout(context.Background(), 6000*time.Second)
	defer cancel()
	return lbClient.List(ctx, config.GroupName())
}

// GetNicFromVMName returns primary nic object based on vm name
func (az AzureClient) GetNicFromVMName(nodename string) (network.Interface, error) {
	return az.getNic(nodename, true)
}

// getNic return a nic object
func (az AzureClient) getNic(resource string, vmResource bool) (network.Interface, error) {

	client := GetNicClient()
	if vmResource {
		resource = az.getNicNameFromVMName(resource)
		//	fmt.Println("Nic", resource)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6000*time.Second)
	defer cancel()
	return client.Get(ctx, config.GroupName(), resource, "")
}

// getNicNameFromVMName return a nicname from VM
func (az AzureClient) getNicNameFromVMName(nodename string) string {
	vm, error := az.GetVM(context.Background(), nodename)
	if error != nil {
		fmt.Printf("failed to getVM: %v", error)
	}
	primaryNicID, err := getPrimaryInterfaceID(vm)

	if err != nil {
		fmt.Printf("failed to getPrimaryInterfaceID from VM: %v", err)
	}

	nicName, err := getLastSegment(primaryNicID)

	if err != nil {
		fmt.Printf("failed to nic name from nicID: %v", err)
	}

	return nicName
}

// This returns the full identifier of the primary NIC for the given VM.
func getPrimaryInterfaceID(machine compute.VirtualMachine) (string, error) {
	if len(*machine.NetworkProfile.NetworkInterfaces) == 1 {
		return *(*machine.NetworkProfile.NetworkInterfaces)[0].ID, nil
	}

	for _, ref := range *machine.NetworkProfile.NetworkInterfaces {
		if *ref.Primary {
			return *ref.ID, nil
		}
	}

	return "", fmt.Errorf("failed to find a primary nic for the vm. vmname=%q", *machine.Name)
}

// returns the deepest child's identifier from a full identifier string.
func getLastSegment(ID string) (string, error) {
	parts := strings.Split(ID, "/")
	name := parts[len(parts)-1]
	if len(name) == 0 {
		return "", fmt.Errorf("resource name was missing from identifier")
	}
	return name, nil
}

// GetAllVM Returns a ListResultPage of all VMs in the ResourceGroup of the Config
func (az AzureClient) GetAllVM() (result compute.VirtualMachineListResultPage) {
	client := GetVmClient()
	ctx, cancel := context.WithTimeout(context.Background(), 6000*time.Second)
	defer cancel()
	//fmt.Println("GetAllVM")
	result, err := client.List(ctx, config.GroupName())
	if err != nil {
		log.Panicf("failed to get all VMs: %v", err)
	}
	return
}

// PutVM returns the Virtual Machine object
func (az AzureClient) PutVM(nodename string) (res autorest.Response) {
	ctx, cancel := context.WithTimeout(context.Background(), 6000*time.Second)
	defer cancel()
	//fmt.Println("PutVM")
	node, err := az.GetVM(ctx, nodename)
	if err != nil {
		log.Panic(err)
	}
	req, err := az.VirtualMachinesClient.CreateOrUpdatePreparer(ctx, config.GroupName(), nodename, node)
	if err != nil {
		log.Panic(err)
	}

	var result *http.Response
	result, err = autorest.SendWithSender(az.VirtualMachinesClient, req,
		azure.DoRetryWithRegistration(az.VirtualMachinesClient.Client))
	err = autorest.Respond(result, azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusCreated))
	if err != nil {
		log.Panic(err)
	}
	res.Response = result

	return
}

// GetAllNics Returns a ListResultPage of all Interfaces in the ResourceGroup of the Config
func (az AzureClient) GetAllNics() network.InterfaceListResultPage {
	client := GetNicClient()
	ctx, cancel := context.WithTimeout(context.Background(), 6000*time.Second)
	defer cancel()
	result, err := client.List(ctx, config.GroupName())
	if err != nil {
		log.Printf("failed to get all Interfaces; check HTTP_PROXY: %v", err)
	}

	return result
}
