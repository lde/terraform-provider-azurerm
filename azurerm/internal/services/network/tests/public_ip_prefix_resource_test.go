package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/acceptance"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/acceptance/check"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

type PublicIPPrefixResource struct {
}

func (t PublicIPPrefixResource) Exists(ctx context.Context, clients *clients.Client, state *terraform.InstanceState) (*bool, error) {
	id, err := azure.ParseAzureResourceID(state.ID)
	if err != nil {
		return nil, err
	}
	resGroup := id.ResourceGroup
	name := id.Path["publicIPPrefixes"]

	resp, err := clients.Network.PublicIPPrefixesClient.Get(ctx, resGroup, name, "")
	if err != nil {
		return nil, fmt.Errorf("reading Public IP Prefix (%s): %+v", id, err)
	}

	return utils.Bool(resp.ID != nil), nil
}

func testCheckPublicIPPrefixDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := acceptance.AzureProvider.Meta().(*clients.Client).Network.PublicIPPrefixesClient
		ctx := acceptance.AzureProvider.Meta().(*clients.Client).StopContext

		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		publicIpPrefixName := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for public ip prefix: %s", publicIpPrefixName)
		}

		future, err := client.Delete(ctx, resourceGroup, publicIpPrefixName)
		if err != nil {
			return fmt.Errorf("Error deleting Public IP Prefix %q (Resource Group %q): %+v", publicIpPrefixName, resourceGroup, err)
		}

		if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
			return fmt.Errorf("Error waiting for deletion of Public IP Prefix %q (Resource Group %q): %+v", publicIpPrefixName, resourceGroup, err)
		}

		return nil
	}
}

func TestAccPublicIpPrefix_basic(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_public_ip_prefix", "test")
	r := PublicIPPrefixResource{}

	data.ResourceTest(t, r, []resource.TestStep{
		{
			Config: r.basic(data),
			Check: resource.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("ip_prefix").Exists(),
				check.That(data.ResourceName).Key("prefix_length").HasValue("28"),
			),
		},
		data.ImportStep(),
	})
}

func TestAccPublicIpPrefix_prefixLength31(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_public_ip_prefix", "test")
	r := PublicIPPrefixResource{}

	data.ResourceTest(t, r, []resource.TestStep{
		{
			Config: r.prefixLength31(data),
			Check: resource.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("ip_prefix").Exists(),
				check.That(data.ResourceName).Key("prefix_length").HasValue("31"),
			),
		},
		data.ImportStep(),
	})
}

func TestAccPublicIpPrefix_prefixLength24(t *testing.T) {
	// NOTE: This test will fail unless the subscription is updated
	//        to accept a minimum PrefixLength of 24
	data := acceptance.BuildTestData(t, "azurerm_public_ip_prefix", "test")
	r := PublicIPPrefixResource{}

	data.ResourceTest(t, r, []resource.TestStep{
		{
			Config: r.prefixLength24(data),
			Check: resource.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("ip_prefix").Exists(),
				check.That(data.ResourceName).Key("prefix_length").HasValue("24"),
			),
		},
		data.ImportStep(),
	})
}

func TestAccPublicIpPrefix_update(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_public_ip_prefix", "test")
	r := PublicIPPrefixResource{}

	data.ResourceTest(t, r, []resource.TestStep{
		{
			Config: r.withTags(data),
			Check: resource.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("tags.%").HasValue("2"),
				check.That(data.ResourceName).Key("tags.environment").HasValue("Production"),
				check.That(data.ResourceName).Key("tags.cost_center").HasValue("MSFT"),
			),
		},
		{
			Config: r.withTagsUpdate(data),
			Check: resource.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("tags.%").HasValue("1"),
				check.That(data.ResourceName).Key("tags.environment").HasValue("staging"),
			),
		},
	})
}

func TestAccPublicIpPrefix_disappears(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_public_ip_prefix", "test")
	r := PublicIPPrefixResource{}

	data.ResourceTest(t, r, []resource.TestStep{
		{
			Config: r.basic(data),
			Check: resource.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				testCheckPublicIPPrefixDisappears(data.ResourceName),
			),
			ExpectNonEmptyPlan: true,
		},
	})
}

func (PublicIPPrefixResource) basic(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurerm_public_ip_prefix" "test" {
  name                = "acctestpublicipprefix-%d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger)
}

func (PublicIPPrefixResource) withTags(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurerm_public_ip_prefix" "test" {
  name                = "acctestpublicipprefix-%d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name

  tags = {
    environment = "Production"
    cost_center = "MSFT"
  }
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger)
}

func (PublicIPPrefixResource) withTagsUpdate(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurerm_public_ip_prefix" "test" {
  name                = "acctestpublicipprefix-%d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name

  tags = {
    environment = "staging"
  }
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger)
}

func (PublicIPPrefixResource) prefixLength31(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurerm_public_ip_prefix" "test" {
  name                = "acctestpublicipprefix-%d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name

  prefix_length = 31
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger)
}

func (PublicIPPrefixResource) prefixLength24(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurerm_public_ip_prefix" "test" {
  name                = "acctestpublicipprefix-%d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name

  prefix_length = 24
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger)
}
