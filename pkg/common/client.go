package common

import (
	"azure-request-limitometer/internal/config"
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

var subscriptionID = "645f4d1b-d55d-4dba-944d-3be470c458d2"

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
		getVMClient(),
		getNicClient(),
		getLBClient(),
	}
	return
}

func getVMClient() compute.VirtualMachinesClient {
	vmClient := compute.NewVirtualMachinesClient(subscriptionID)
	a, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		log.Fatalf("failed to create authorizer from environment: %s\n", err)
	}
	vmClient.Authorizer = a
	vmClient.AddToUserAgent(config.UserAgent())
	return vmClient
}

func getNicClient() network.InterfacesClient {
	nicClient := network.NewInterfacesClient(subscriptionID)
	a, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		log.Fatalf("failed to create authorizer from environment: %s\n", err)
	}
	nicClient.Authorizer = a
	nicClient.AddToUserAgent(config.UserAgent())
	return nicClient
}

func getLBClient() network.LoadBalancersClient {
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
	client := getVMClient()
	fmt.Printf("VM")
	return client.Get(ctx, config.GroupName(), nodename, compute.InstanceView)
}

// GetLBFromVMName returns primary LB object based on vm name.
func (az AzureClient) GetLbFromVMName(nodename string) (network.LoadBalancer, error) {
	return az.getLoadBalancer(context.Background(), nodename, true)

}

// GetLoadBalancer gets info on a loadbalancer
func (az AzureClient) getLoadBalancer(ctx context.Context, resource string, vmResource bool) (network.LoadBalancer, error) {
	lbClient := getLBClient()
	if vmResource {
		resource = az.getLBNameFromVMName(resource)
		fmt.Println("LB", resource)
	}
	return lbClient.Get(ctx, config.GroupName(), resource, "")
}

func (az AzureClient) getLBNameFromVMName(nodename string) string {
	vm, error := az.GetVM(context.Background(), nodename)
	if error != nil {
		fmt.Printf("failed to getVM: %v", error)
	}
	primaryNicID, err := getPrimaryInterfaceID(vm)
	fmt.Println(primaryNicID)
	if err != nil {
		fmt.Printf("failed to getPrimaryInterfaceID from VM: %v", err)
	}

	nicName, err := getLastSegment(primaryNicID)
	fmt.Println(nicName)
	if err != nil {
		fmt.Printf("failed to nic name from nicID: %v", err)
	}
	lbName := "hetal-test"
	return lbName
}

// GetNicFromVMName returns primary nic object based on vm name
func (az AzureClient) GetNicFromVMName(nodename string) (network.Interface, error) {
	return az.getNic(nodename, true)
}

func (az AzureClient) getNic(resource string, vmResource bool) (network.Interface, error) {

	client := getNicClient()
	if vmResource {
		resource = az.getNicNameFromVMName(resource)
		fmt.Println("Nic", resource)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6000*time.Second)
	defer cancel()
	//return client.Get(ctx, resourceGroupName, networkInterfaceName, expand)
	return client.Get(ctx, config.GroupName(), resource, "")
}

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
	ctx, cancel := context.WithTimeout(context.Background(), 6000*time.Second)
	defer cancel()
	fmt.Println("GetAllVM")
	result, err := az.VirtualMachinesClient.List(ctx, config.GroupName())
	if err != nil {
		log.Panicf("failed to get all VMs: %v", err)
	}
	return
}

// PutVM returns the Virtual Machine object
func (az AzureClient) PutVM(nodename string) (res autorest.Response) {
	ctx, cancel := context.WithTimeout(context.Background(), 6000*time.Second)
	defer cancel()
	fmt.Println("PutVM")
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
	ctx, cancel := context.WithTimeout(context.Background(), 6000*time.Second)
	defer cancel()

	result, err := az.InterfacesClient.List(ctx, config.GroupName())
	if err != nil {
		log.Printf("failed to get all Interfaces; check HTTP_PROXY: %v", err)
	}

	return result
}

/*
// PutNic returns the Interface object
func (c AzureClient) PutNic(vmName string) autorest.Response {
	ctx, cancel := context.WithTimeout(context.Background(), 6000*time.Second)
	defer cancel()

	nicName := c.getNicNameFromVMName(vmName)

	nic := c.GetNicFromNicName(nicName)

	req, err := c.InterfacesClient.CreateOrUpdatePreparer(ctx, Conf.ResourceGroup, nicName, nic)
	if err != nil {
		err = autorest.NewErrorWithError(err, "compute.InterfacesClient", "CreateOrUpdatePreparer", nil, "Failure preparing request")
	}

	var resp *http.Response
	resp, err = autorest.SendWithSender(c.InterfacesClient, req,
		azure.DoRetryWithRegistration(c.InterfacesClient.Client))
	err = autorest.Respond(resp, azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusCreated))
	if err != nil {
		glog.Fatal(err)
	}

	return autorest.Response{Response: resp}
}
*/
