package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/reconcile-kit/api/resource"
)

func TestStateManagerProviderListAddsQueryFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		listOpts          resource.ListOpts
		wantLabelSelector []string
	}{
		{
			name: "without label selector",
			listOpts: resource.ListOpts{
				Namespace: "ns1",
				Name:      "resource-1",
				ShardID:   "shard-1",
			},
		},
		{
			name: "with in label selectors",
			listOpts: resource.ListOpts{
				Namespace: "ns1",
				Name:      "resource-1",
				ShardID:   "shard-1",
				LabelSelectors: []resource.LabelSelector{
					{
						Key:      "secg/application-app1",
						Operator: resource.LabelSelectorOperatorIn,
						Values:   []string{"true", "shadow"},
					},
					{
						Key:      "team",
						Operator: resource.LabelSelectorOperatorIn,
						Values:   []string{"network"},
					},
				},
			},
			wantLabelSelector: []string{
				"secg/application-app1 in (true,shadow),team in (network)",
			},
		},
		{
			name: "with empty selector values",
			listOpts: resource.ListOpts{
				Namespace: "ns1",
				Name:      "resource-1",
				ShardID:   "shard-1",
				LabelSelectors: []resource.LabelSelector{
					{
						Key:      "team",
						Operator: resource.LabelSelectorOperatorIn,
						Values:   []string{},
					},
				},
			},
			wantLabelSelector: []string{
				`team in ("")`,
			},
		},
		{
			name: "with empty string selector value",
			listOpts: resource.ListOpts{
				Namespace: "ns1",
				Name:      "resource-1",
				ShardID:   "shard-1",
				LabelSelectors: []resource.LabelSelector{
					{
						Key:      "group/owner",
						Operator: resource.LabelSelectorOperatorIn,
						Values:   []string{""},
					},
				},
			},
			wantLabelSelector: []string{
				`group/owner in ("")`,
			},
		},
		{
			name: "with nil selector values",
			listOpts: resource.ListOpts{
				Namespace: "ns1",
				Name:      "resource-1",
				ShardID:   "shard-1",
				LabelSelectors: []resource.LabelSelector{
					{
						Key:      "owner",
						Operator: resource.LabelSelectorOperatorIn,
						Values:   nil,
					},
				},
			},
			wantLabelSelector: []string{
				`owner in ("")`,
			},
		},
		{
			name: "with empty string among other selector values",
			listOpts: resource.ListOpts{
				Namespace: "ns1",
				Name:      "resource-1",
				ShardID:   "shard-1",
				LabelSelectors: []resource.LabelSelector{
					{
						Key:      "owner",
						Operator: resource.LabelSelectorOperatorIn,
						Values:   []string{"", "data"},
					},
				},
			},
			wantLabelSelector: []string{
				`owner in ("",data)`,
			},
		},
		{
			name: "with selector values that require quoting",
			listOpts: resource.ListOpts{
				Namespace: "ns1",
				Name:      "resource-1",
				ShardID:   "shard-1",
				LabelSelectors: []resource.LabelSelector{
					{
						Key:      "test",
						Operator: resource.LabelSelectorOperatorIn,
						Values:   []string{"(", "with space", "with,comma", `a"b`, `a\b`, "plain"},
					},
				},
			},
			wantLabelSelector: []string{
				`test in ("(","with space","with,comma","a\"b","a\\b",plain)`,
			},
		},
		{
			name: "with operator keywords as selector values",
			listOpts: resource.ListOpts{
				Namespace: "ns1",
				Name:      "resource-1",
				ShardID:   "shard-1",
				LabelSelectors: []resource.LabelSelector{
					{
						Key:      "test",
						Operator: resource.LabelSelectorOperatorIn,
						Values:   []string{"in", "notin"},
					},
				},
			},
			wantLabelSelector: []string{
				`test in (in,notin)`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			type observedRequest struct {
				path  string
				query map[string][]string
			}

			requests := make(chan observedRequest, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests <- observedRequest{
					path:  r.URL.Path,
					query: r.URL.Query(),
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("[]"))
			}))
			defer server.Close()

			provider, err := NewStateManagerProvider[*ApiResource](server.URL, server.Client())
			if err != nil {
				t.Fatalf("NewStateManagerProvider() error = %v", err)
			}

			_, err = provider.List(
				context.Background(),
				resource.GroupKind{
					Group: "compute.data",
					Kind:  "port",
				},
				tt.listOpts,
			)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}

			request := <-requests
			if request.path != "/api/v1/resources" {
				t.Fatalf("path = %q, want %q", request.path, "/api/v1/resources")
			}
			if got := request.query["resource_group"]; !slices.Equal(got, []string{"compute.data"}) {
				t.Fatalf("resource_group = %#v, want %#v", got, []string{"compute.data"})
			}
			if got := request.query["kind"]; !slices.Equal(got, []string{"port"}) {
				t.Fatalf("kind = %#v, want %#v", got, []string{"port"})
			}
			if got := request.query["namespace"]; !slices.Equal(got, []string{tt.listOpts.Namespace}) {
				t.Fatalf("namespace = %#v, want %#v", got, []string{tt.listOpts.Namespace})
			}
			if got := request.query["name"]; !slices.Equal(got, []string{tt.listOpts.Name}) {
				t.Fatalf("name = %#v, want %#v", got, []string{tt.listOpts.Name})
			}
			if got := request.query["shard_id"]; !slices.Equal(got, []string{tt.listOpts.ShardID}) {
				t.Fatalf("shard_id = %#v, want %#v", got, []string{tt.listOpts.ShardID})
			}
			if got := request.query["label_selector"]; !slices.Equal(got, tt.wantLabelSelector) {
				t.Fatalf("label_selector = %#v, want %#v", got, tt.wantLabelSelector)
			}
		})
	}
}

func TestStateManagerProviderListE2E(t *testing.T) {
	httpClient := &http.Client{}

	provider, err := NewStateManagerProvider[*ApiResource]("http://localhost:8080", httpClient)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	runID := time.Now().UnixNano()
	namespace := "default"
	shardID := fmt.Sprintf("list-e2e-%d", runID)

	resource1Name := fmt.Sprintf("test-list-resource-1-%d", runID)
	resource2Name := fmt.Sprintf("test-list-resource-2-%d", runID)

	gk := resource.GroupKind{Group: "compute.data", Kind: "port"}

	resource1 := &ApiResource{
		Resource: resource.Resource{
			ResourceGroup: gk.Group,
			Kind:          gk.Kind,
			Namespace:     namespace,
			Name:          resource1Name,
			ShardID:       shardID,
			Annotations:   map[string]string{"revision": "new"},
			Labels:        map[string]string{"team": "network", "project_id": "1234567"},
		},
		Spec: Spec{
			Flavor:   "m1.small",
			DiskSize: 200,
			FIP:      "192.168.0.14",
		},
		Status: Status{},
	}

	resource2 := &ApiResource{
		Resource: resource.Resource{
			ResourceGroup: gk.Group,
			Kind:          gk.Kind,
			Namespace:     namespace,
			Name:          resource2Name,
			ShardID:       shardID,
			Annotations:   map[string]string{"revision": "new"},
			Labels:        map[string]string{"team": "storage", "project_id": "1234567"},
		},
		Spec: Spec{
			Flavor:   "m1.large",
			DiskSize: 400,
			FIP:      "192.168.0.15",
		},
		Status: Status{},
	}

	err = provider.Create(ctx, resource1)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := provider.Delete(ctx, gk, resource.ObjectKey{Namespace: namespace, Name: resource1Name}); err != nil {
			t.Logf("cleanup delete %q error: %v", resource1Name, err)
		}
	})

	err = provider.Create(ctx, resource2)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := provider.Delete(ctx, gk, resource.ObjectKey{Namespace: namespace, Name: resource2Name}); err != nil {
			t.Logf("cleanup delete %q error: %v", resource2Name, err)
		}
	})

	items, err := provider.List(ctx, gk, resource.ListOpts{
		Namespace: namespace,
		ShardID:   shardID,
	})
	if err != nil {
		t.Fatal(err)
	}

	gotNames := map[string]struct{}{}
	for _, item := range items {
		gotNames[item.Name] = struct{}{}
	}

	if len(items) != 2 {
		t.Fatalf("List() without label selector: expected 2 items, got %d (%#v)", len(items), gotNames)
	}
	if _, ok := gotNames[resource1Name]; !ok {
		t.Fatalf("List() without label selector: expected %q to be present, got %#v", resource1Name, gotNames)
	}
	if _, ok := gotNames[resource2Name]; !ok {
		t.Fatalf("List() without label selector: expected %q to be present, got %#v", resource2Name, gotNames)
	}

	filteredItems, err := provider.List(ctx, gk, resource.ListOpts{
		Namespace: namespace,
		ShardID:   shardID,
		LabelSelectors: []resource.LabelSelector{
			{
				Key:      "team",
				Operator: resource.LabelSelectorOperatorIn,
				Values:   []string{"network"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	filteredNames := map[string]struct{}{}
	for _, item := range filteredItems {
		filteredNames[item.Name] = struct{}{}
	}

	if len(filteredItems) != 1 {
		t.Fatalf("List() with label selector: expected 1 item, got %d (%#v)", len(filteredItems), filteredNames)
	}
	if _, ok := filteredNames[resource1Name]; !ok {
		t.Fatalf("List() with label selector: expected %q to be present, got %#v", resource1Name, filteredNames)
	}
	if _, ok := filteredNames[resource2Name]; ok {
		t.Fatalf("List() with label selector: expected %q to be absent, got %#v", resource2Name, filteredNames)
	}
}
