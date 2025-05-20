package provider

import (
	"context"
	"github.com/reconcile-kit/api/resource"
	"net/http"
	"testing"
)

type ApiResource struct {
	resource.Resource
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Spec      Spec   `json:"spec"`
	Status    Status `json:"status"`
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

	res := &ApiResource{
		Resource: resource.Resource{
			ResourceGroup: "compute.salt.x5.ru",
			Kind:          "port",
			Namespace:     "default",
			Name:          "vm-port1",
			ShardID:       "default",
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

	body, err := provider.Create(ctx, res)
	if err != nil {
		t.Fatal(err)
	}

	items, err := provider.ListPending(ctx, "default", resource.GroupKind{Group: "compute.salt.x5.ru", Kind: "port"})
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 1 {
		t.Fatal("List pending error")
	}

	item := items[0]
	if item.Name != "vm-port1" {
		t.Fatal("List pending error")
	}

	if body.Spec.Flavor != "m1.small" {
		t.Errorf("Spec.Flavor got %s, want %s", body.Spec.Flavor, "m1.small")
	}

	body.Labels["test-label"] = "test"
	body, err = provider.Update(ctx, body)
	if err != nil {
		t.Fatal(err)
	}

	body.Status.Flavor = "m1.small"
	body.Status.DiskSize = 200
	body.Status.FIP = "192.168.0.14"

	body, err = provider.UpdateStatus(ctx, body)
	if err != nil {
		t.Fatal(err)
	}

	if body.Status.Flavor != "m1.small" {
		t.Errorf("Spec.Flavor got %s, want %s", body.Status.Flavor, "m1.small")
	}

	body, ok, err := provider.Get(ctx, "default", resource.GroupKind{Group: body.ResourceGroup, Kind: body.Kind}, resource.ObjectKey{
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

	err = provider.Delete(ctx, "default", resource.GroupKind{Group: body.ResourceGroup, Kind: body.Kind}, resource.ObjectKey{
		Namespace: body.Namespace,
		Name:      body.Name,
	})
	if err != nil {
		t.Fatal(err)
	}

	body, ok, err = provider.Get(ctx, "default", resource.GroupKind{Group: body.ResourceGroup, Kind: body.Kind}, resource.ObjectKey{
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
