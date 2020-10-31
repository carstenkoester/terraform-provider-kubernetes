package kubernetes

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccKubernetesDataSourceAllApiResources_basic(t *testing.T) {
	rxPosNum := regexp.MustCompile("^[1-9][0-9]*$")
	nsName := regexp.MustCompile("^[a-zA-Z][-\\w]*$")
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			// FIXME: These tests are broken. They are copy-and-paste leftover. Need to define proper test critera. We can
			// check the output for the existence of some resource types that always exist, perhaps.
			{
				Config: testAccKubernetesDataSourceAllApiResourcesConfig_basic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("data.kubernetes_all_namespaces.test", "namespaces.#", rxPosNum),
					resource.TestCheckResourceAttrSet("data.kubernetes_all_namespaces.test", "namespaces.0"),
					resource.TestMatchResourceAttr("data.kubernetes_all_namespaces.test", "namespaces.0", nsName),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceAllApiResourcesConfig_basic() string {
	return `
data "kubernetes_all_api_resources" "test" {}
`
}
