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
	requestURL := &url.URL{
		Scheme: p.baseURL.Scheme,
		Host:   p.baseURL.Host,
		Path:   rel,
	}
	if err = p.do(ctx, http.MethodGet, requestURL, nil, &out); err != nil {
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

	var list []T
	requestURL := &url.URL{
		Scheme:   p.baseURL.Scheme,
		Host:     p.baseURL.Host,
		Path:     rel,
		RawQuery: q.Encode(),
	}
	return list, p.do(ctx, http.MethodGet, requestURL, nil, &list)
}

func (p *StateManagerProvider[T]) ListPending(
	ctx context.Context,
	shardID string,
	gk resource.GroupKind,
) ([]T, error) {
	var allResources []T
	offset := 0
	limit := 100
	for {
		q := url.Values{
			"pending":        []string{"true"},
			"resource_group": []string{gk.Group},
			"kind":           []string{gk.Kind},
			"shard_id":       []string{shardID},
			"limit":          []string{fmt.Sprintf("%d", limit)},
			"offset":         []string{fmt.Sprintf("%d", offset)},
		}

		requestURL := &url.URL{
			Scheme:   p.baseURL.Scheme,
			Host:     p.baseURL.Host,
			Path:     "/api/v1/resources",
			RawQuery: q.Encode(),
		}

		var batch []T
		err := p.do(ctx, http.MethodGet, requestURL, nil, &batch)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch batch at offset %d: %w", offset, err)
		}

		allResources = append(allResources, batch...)
		if len(batch) < limit {
			break
		}
		offset += limit
	}

	return allResources, nil
}

func (p *StateManagerProvider[T]) Create(ctx context.Context, obj T) (T, error) {
	rel := resourcePathOfCreate(obj)
	var got T
	requestURL := &url.URL{
		Scheme: p.baseURL.Scheme,
		Host:   p.baseURL.Host,
		Path:   rel,
	}
	return got, p.do(ctx, http.MethodPost, requestURL, obj, &got)
}

func (p *StateManagerProvider[T]) Update(ctx context.Context, obj T) (T, error) {
	rel := resourcePathOf(obj)
	var got T
	requestURL := &url.URL{
		Scheme: p.baseURL.Scheme,
		Host:   p.baseURL.Host,
		Path:   rel,
	}
	return got, p.do(ctx, http.MethodPut, requestURL, obj, &got)
}

func (p *StateManagerProvider[T]) UpdateStatus(ctx context.Context, obj T) (T, error) {
	rel := resourcePathOf(obj) + "/status"
	var got T
	requestURL := &url.URL{
		Scheme: p.baseURL.Scheme,
		Host:   p.baseURL.Host,
		Path:   rel,
	}
	return got, p.do(ctx, http.MethodPut, requestURL, obj, &got)
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
	requestURL := &url.URL{
		Scheme: p.baseURL.Scheme,
		Host:   p.baseURL.Host,
		Path:   rel,
	}
	return p.do(ctx, http.MethodDelete, requestURL, nil, nil)
}
