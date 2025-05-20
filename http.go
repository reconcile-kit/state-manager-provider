package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/reconcile-kit/api/resource"
	"io"
	"net/http"
	"net/url"
)

var (
	ErrBadInput  = errors.New("state-manager: bad input")
	ErrServerErr = errors.New("state-manager: internal error")
)

var jsonIter = jsoniter.Config{
	EscapeHTML:             false,
	SortMapKeys:            false,
	ValidateJsonRawMessage: true,
}.Froze()

func (p *StateManagerProvider[T]) do(
	ctx context.Context,
	method string,
	rel *url.URL,
	body any,
	out any,
) error {

	u := rel

	var rdr io.Reader
	if body != nil {
		j, err := jsonIter.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(j)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", p.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := p.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// 2xx → распарсим в out, если есть
	if res.StatusCode/100 == 2 {
		if out == nil || res.StatusCode == http.StatusNoContent {
			return nil
		}
		return jsonIter.NewDecoder(res.Body).Decode(out)
	}
	errorBody, _ := io.ReadAll(res.Body)
	switch res.StatusCode {
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", ErrBadInput, errorBody)
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", resource.NotFoundError, errorBody)
	case http.StatusConflict:
		return fmt.Errorf("%w: %s", resource.ConflictError, errorBody)
	default:
		return fmt.Errorf("%w: %s (code %d)", ErrServerErr, errorBody, res.StatusCode)
	}
}

// формирует canonical URL ресурса из самого объекта
func resourcePathOf[T resource.Object[T]](obj T) string {
	meta := obj.GetName()
	gk := obj.GetGK()
	return fmt.Sprintf("/api/v1/groups/%s/namespaces/%s/kinds/%s/resources/%s",
		url.PathEscape(gk.Group),
		url.PathEscape(meta.Namespace),
		url.PathEscape(gk.Kind),
		url.PathEscape(meta.Name),
	)
}

func resourcePathOfCreate[T resource.Object[T]](obj T) string {
	meta := obj.GetName()
	gk := obj.GetGK()
	return fmt.Sprintf("/api/v1/groups/%s/namespaces/%s/kinds/%s/resources",
		url.PathEscape(gk.Group),
		url.PathEscape(meta.Namespace),
		url.PathEscape(gk.Kind),
	)
}

func addQuery(p string, q url.Values) string {
	if len(q) == 0 {
		return p
	}
	return p + "?" + q.Encode()
}
