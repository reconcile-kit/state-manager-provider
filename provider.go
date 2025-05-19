package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/reconcile-kit/api/resource"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type StateManagerProvider[T resource.Object[T]] struct {
	http      *http.Client
	baseURL   *url.URL
	userAgent string
}

func NewStateManagerProvider[T resource.Object[T]](base string, client *http.Client) (*StateManagerProvider[T], error) {
	u, err := url.Parse(strings.TrimSuffix(base, "/"))
	if err != nil {
		return nil, err
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &StateManagerProvider[T]{http: client, baseURL: u, userAgent: "state-mgr-sdk/1.0"}, nil
}

func (p *StateManagerProvider[T]) Get(
	ctx context.Context,
	shardID string,
	gk resource.GroupKind,
	key resource.ObjectKey,
) (_ T, found bool, err error) {

	rel := fmt.Sprintf("/api/v1/groups/%s/namespaces/%s/kinds/%s/resources/%s",
		url.PathEscape(gk.Group),
		url.PathEscape(key.Namespace),
		url.PathEscape(gk.Kind),
		url.PathEscape(key.Name),
	)
	var out T
	if err = p.do(ctx, http.MethodGet, rel, nil, &out); err != nil {
		if errors.Is(err, resource.NotFoundError) {
			return out, false, nil
		}
		return out, false, err
	}
	return out, true, nil
}

func (p *StateManagerProvider[T]) List(ctx context.Context, o resource.ListOpts) ([]T, error) {
	rel := "/api/v1/resources"
	q := url.Values{}
	if o.Group != "" {
		q.Set("resource_group", o.Group)
	}
	if o.Kind != "" {
		q.Set("kind", o.Kind)
	}
	if o.Namespace != "" {
		q.Set("namespace", o.Namespace)
	}
	if o.Name != "" {
		q.Set("name", o.Name)
	}
	if o.ShardID != "" {
		q.Set("shard_id", o.ShardID)
	}
	rel = addQuery(rel, q)

	var list []T
	return list, p.do(ctx, http.MethodGet, rel, nil, &list)
}

func (p *StateManagerProvider[T]) ListPending(
	ctx context.Context,
	shardID string,
	gk resource.GroupKind,
) ([]T, error) {

	q := url.Values{
		"pending":        []string{"true"},
		"resource_group": []string{gk.Group},
		"kind":           []string{gk.Kind},
		"shard_id":       []string{shardID},
	}
	rel := addQuery("/api/v1/resources", q)

	var list []T
	return list, p.do(ctx, http.MethodGet, rel, nil, &list)
}

func (p *StateManagerProvider[T]) Create(ctx context.Context, obj T) (T, error) {
	rel := resourcePathOfCreate(obj)
	var got T
	return got, p.do(ctx, http.MethodPost, rel, obj, &got)
}

func (p *StateManagerProvider[T]) Update(ctx context.Context, obj T) (T, error) {
	rel := resourcePathOf(obj)
	var got T
	return got, p.do(ctx, http.MethodPut, rel, obj, &got)
}

func (p *StateManagerProvider[T]) UpdateStatus(ctx context.Context, obj T) (T, error) {
	rel := resourcePathOf(obj) + "/status"
	var got T
	return got, p.do(ctx, http.MethodPut, rel, obj, &got)
}

func (p *StateManagerProvider[T]) Delete(
	ctx context.Context,
	shardID string,
	gk resource.GroupKind,
	key resource.ObjectKey,
) error {

	rel := fmt.Sprintf("/api/v1/groups/%s/namespaces/%s/kinds/%s/resources/%s",
		url.PathEscape(gk.Group),
		url.PathEscape(key.Namespace),
		url.PathEscape(gk.Kind),
		url.PathEscape(key.Name),
	)
	return p.do(ctx, http.MethodDelete, rel, nil, nil)
}
