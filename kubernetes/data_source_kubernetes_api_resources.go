package kubernetes

import (
	"crypto/sha256"
	"fmt"
	"log"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
)

func dataSourceKubernetesAllApiResources() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceKubernetesAllApiResourcesRead,
		Schema: map[string]*schema.Schema{
			"api_resources": {
				// FIXME: This probably really should be a Set (schema.TypeSet), not a List. In that case, we'd
				// need to tell Terraform how to dedup set elements.
				Type:        schema.TypeList,
				Description: "List of all available API resource types in a cluster.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"api_group": {
							Type:        schema.TypeString,
							Description: "API Group",
							Computed:    true,
						},
						"api_group_version": {
							Type:        schema.TypeString,
							Description: "API Group version",
							Computed:    true,
						},
						"name": {
							Type:        schema.TypeString,
							Description: "API resource name",
							Computed:    true,
						},
						"fully_qualified_name": {
							Type:        schema.TypeString,
							Description: "Fully qualified name (Concatenation of resource name and API group name)",
							Computed:    true,
						},
						"shortnames": {
							Type:        schema.TypeSet,
							Description: "Set of short names for the resource",
							Computed:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Set: schema.HashString,
						},
						"namespaced": {
							Type:        schema.TypeBool,
							Description: "Indicated whether or not the resource is namespaced",
							Computed:    true,
						},
						"kind": {
							Type:        schema.TypeString,
							Description: "API resource kind",
							Computed:    true,
						},
						"verbs": {
							Type:        schema.TypeSet,
							Description: "Set of verbs supported by this resource",
							Computed:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Set: schema.HashString,
						},
					},
				},
			},
		},
	}
}

// Map sliced used as internal storage for API resources
type apiResource map[string]interface{}

func dataSourceKubernetesAllApiResourcesRead(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	log.Printf("[INFO] Listing API resources")
	discoveryclient := conn.Discovery()
	lists, err := discoveryclient.ServerPreferredResources()

	resources := []apiResource{}

	// Read resources and create internal data structure (map)
	for _, list := range lists {
		if len(list.APIResources) == 0 {
			continue
		}
		gv, err := k8sschema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}

		for _, resource := range list.APIResources {
			if len(resource.Verbs) == 0 {
				continue
			}
			resources = append(resources, apiResource{
				"api_group":         gv.Group,
				"api_group_version": gv.String(),
				"name":              resource.Name,
				"shortnames":        resource.ShortNames,
				"namespaced":        resource.Namespaced,
				"kind":              resource.Kind,
				"verbs":             resource.Verbs,
			})
		}
	}

	// Compute fully-qualified name for each resource
	for _, r := range resources {
		fqn := string(r["name"].(string))
		if len(r["api_group"].(string)) > 0 {
			fqn += "." + r["api_group"].(string)
		}
		r["fully_qualified_name"] = fqn
		log.Printf("[INFO] Found API resource: %s", fqn)
	}

	// Sort the internal data structure. This will help us to create a unique ID
	sort.Slice(resources, func(i, j int) bool {
		return resources[i]["fully_qualified_name"].(string) < resources[j]["fully_qualified_name"].(string)
	})

	// Now update the Terraform set
	err = d.Set("api_resources", resources)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Received number of API resources: %d", len(resources))

	// Create a Terraform set ID by hashing over all API resource FQNs
	idsum := sha256.New()
	for _, v := range resources {
		_, err := idsum.Write([]byte(v["fully_qualified_name"].(string)))
		if err != nil {
			return err
		}
	}
	id := fmt.Sprintf("%x", idsum.Sum(nil))
	log.Printf("[DEBUG] Resource ID %s", id)
	d.SetId(id)

	return nil
}
