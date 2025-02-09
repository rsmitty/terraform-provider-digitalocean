package droplet_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/digitalocean/terraform-provider-digitalocean/digitalocean/acceptance"
	"github.com/digitalocean/terraform-provider-digitalocean/digitalocean/config"
	"github.com/digitalocean/terraform-provider-digitalocean/digitalocean/util"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDigitalOceanDroplet_importBasic(t *testing.T) {
	resourceName := "digitalocean_droplet.foobar"
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acceptance.TestAccPreCheck(t) },
		ProviderFactories: acceptance.TestAccProviderFactories,
		CheckDestroy:      acceptance.TestAccCheckDigitalOceanDropletDestroy,
		Steps: []resource.TestStep{
			{
				Config: acceptance.TestAccCheckDigitalOceanDropletConfig_basic(rInt),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"ssh_keys", "user_data", "resize_disk", "graceful_shutdown"}, //we ignore these attributes as we do not set to state
			},
			// Test importing non-existent resource provides expected error.
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: false,
				ImportStateId:     "123",
				ExpectError:       regexp.MustCompile(`The resource you were accessing could not be found.`),
			},
		},
	})
}

func TestAccDigitalOceanDroplet_ImportWithNoImageSlug(t *testing.T) {
	rInt := acctest.RandInt()
	var droplet godo.Droplet
	var snapshotId []int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acceptance.TestAccPreCheck(t) },
		ProviderFactories: acceptance.TestAccProviderFactories,
		CheckDestroy:      acceptance.TestAccCheckDigitalOceanDropletDestroy,
		Steps: []resource.TestStep{
			{
				Config: acceptance.TestAccCheckDigitalOceanDropletConfig_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					acceptance.TestAccCheckDigitalOceanDropletExists("digitalocean_droplet.foobar", &droplet),
					takeDropletSnapshot(rInt, &droplet, &snapshotId),
				),
			},
			{
				Config: testAccCheckDigitalOceanDropletConfig_fromSnapshot(rInt),
			},
			{
				ResourceName:      "digitalocean_droplet.from-snapshot",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"ssh_keys", "user_data", "resize_disk", "graceful_shutdown"}, //we ignore the ssh_keys, resize_disk and user_data as we do not set to state
			},
			{
				Config: " ",
				Check: resource.ComposeTestCheckFunc(
					acceptance.DeleteDropletSnapshots(&snapshotId),
				),
			},
		},
	})
}

func takeDropletSnapshot(rInt int, droplet *godo.Droplet, snapshotId *[]int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := acceptance.TestAccProvider.Meta().(*config.CombinedConfig).GodoClient()

		action, _, err := client.DropletActions.Snapshot(context.Background(), (*droplet).ID, fmt.Sprintf("snap-%d", rInt))
		if err != nil {
			return err
		}
		util.WaitForAction(client, action)

		retrieveDroplet, _, err := client.Droplets.Get(context.Background(), (*droplet).ID)
		if err != nil {
			return err
		}
		*snapshotId = retrieveDroplet.SnapshotIDs
		return nil
	}
}

func testAccCheckDigitalOceanDropletConfig_fromSnapshot(rInt int) string {
	return fmt.Sprintf(`
data "digitalocean_image" "snapshot" {
  name = "snap-%d"
}

resource "digitalocean_droplet" "from-snapshot" {
  name      = "foo-%d"
  size      = "s-1vcpu-1gb"
  image     = "${data.digitalocean_image.snapshot.id}"
  region    = "nyc3"
  user_data = "foobar"
}`, rInt, rInt)
}
