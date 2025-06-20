package provider

import (
	"context"
	"fmt"
	"github.com/reconcile-kit/api/resource"
	"math/rand"
	"net/http"
	"testing"
	"time"
)

type ApiResource struct {
	resource.Resource
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Spec      Spec   `json:"spec"`
	Status    Status `json:"status"`
}

func (c *ApiResource) GetGK() resource.GroupKind {
	return resource.GroupKind{Group: "compute.salt.x5.ru", Kind: "port"}
}

func (c *ApiResource) DeepCopy() *ApiResource {
	return resource.DeepCopyStruct(c).(*ApiResource)
}

type Spec struct {
	Flavor   string `json:"flavor"`
	DiskSize int    `json:"disk_size"`
	FIP      string `json:"fip"`
}

type Status struct {
	Flavor   string `json:"flavor"`
	DiskSize int    `json:"disk_size"`
	FIP      string `json:"fip"`
}

func TestNewStateManagerProvider(t *testing.T) {

	httpCliet := &http.Client{}

	provider, err := NewStateManagerProvider[*ApiResource]("http://localhost:8080", httpCliet)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	vmPortName := "test-vm-port1" + fmt.Sprint(rand.Int())
	shardID := "test-" + fmt.Sprint(rand.Int())

	fmt.Println(vmPortName, shardID)

	body := &ApiResource{
		Resource: resource.Resource{
			ResourceGroup: "compute.salt.x5.ru",
			Kind:          "port",
			Namespace:     "default",
			Name:          vmPortName,
			ShardID:       shardID,
			Annotations:   map[string]string{"revision": "new"},
			Labels:        map[string]string{"project_id": "1234567"},
		},
		Spec: Spec{
			Flavor:   "m1.small",
			DiskSize: 200,
			FIP:      "192.168.0.14",
		},
		Status: Status{},
	}

	err = provider.Create(ctx, body)
	if err != nil {
		t.Fatal(err)
	}

	if body.CreatedAt == "" {
		t.Fatal("Created At should not be empty")
	}

	items, err := provider.ListPending(ctx, shardID, resource.GroupKind{Group: "compute.salt.x5.ru", Kind: "port"})
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 1 {
		t.Fatal("List pending error", len(items))
	}

	item := items[0]
	if item.Name != vmPortName {
		t.Fatal("List pending error")
	}

	if body.Spec.Flavor != "m1.small" {
		t.Errorf("Spec.Flavor got %s, want %s", body.Spec.Flavor, "m1.small")
	}

	body.Labels["test-label"] = "test"
	err = provider.Update(ctx, body)
	if err != nil {
		t.Fatal(err)
	}

	body.Status.Flavor = "m1.small"
	body.Status.DiskSize = 200
	body.Status.FIP = "192.168.0.14"

	updatedAt := body.UpdatedAt
	time.Sleep(1 * time.Second)

	err = provider.UpdateStatus(ctx, body)
	if err != nil {
		t.Fatal(err)
	}

	if updatedAt == body.UpdatedAt {
		t.Fatal("Updated At should not be updated")
	}

	if body.Status.Flavor != "m1.small" {
		t.Errorf("Spec.Flavor got %s, want %s", body.Status.Flavor, "m1.small")
	}

	body, ok, err := provider.Get(ctx, resource.GroupKind{Group: body.ResourceGroup, Kind: body.Kind}, resource.ObjectKey{
		Namespace: body.Namespace,
		Name:      body.Name,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("not exist resource")
	}

	v, ok := body.Labels["test-label"]
	if !ok || v != "test" {
		t.Errorf("Labels got %s, want %s", v, "test")
	}

	err = provider.Delete(ctx, resource.GroupKind{Group: body.ResourceGroup, Kind: body.Kind}, resource.ObjectKey{
		Namespace: body.Namespace,
		Name:      body.Name,
	})
	if err != nil {
		t.Fatal(err)
	}

	body, ok, err = provider.Get(ctx, resource.GroupKind{Group: body.ResourceGroup, Kind: body.Kind}, resource.ObjectKey{
		Namespace: body.Namespace,
		Name:      body.Name,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("exist resource")
	}
}
