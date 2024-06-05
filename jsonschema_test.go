package jsonschema

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/luraproject/lura/v2/config"
	"github.com/luraproject/lura/v2/logging"
	"github.com/luraproject/lura/v2/proxy"
)

func TestProxyFactory_erroredNext(t *testing.T) {
	errExpected := errors.New("proxy factory called")
	pf := ProxyFactory(logging.NoOp, proxy.FactoryFunc(func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		return func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
			t.Error("proxy called")
			return nil, errors.New("proxy called")
		}, errExpected
	}))

	_, err := pf.New(&config.EndpointConfig{})
	if err == nil {
		t.Error("error expected")
		return
	}
	if err != errExpected {
		t.Errorf("unexpected error %s", err.Error())
	}
}

func TestProxyFactory_bypass(t *testing.T) {
	errExpected := errors.New("proxy called")
	pf := ProxyFactory(logging.NoOp, proxy.FactoryFunc(func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		return func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
			return nil, errExpected
		}, nil
	}))
	p, err := pf.New(&config.EndpointConfig{})
	if err != nil {
		t.Errorf("unexpected error %s", err.Error())
		return
	}
	if _, err := p(context.Background(), &proxy.Request{Body: io.NopCloser(bytes.NewBufferString(""))}); err != errExpected {
		t.Errorf("unexpected error %v", err)
	}
}

func TestProxyFactory_schemaInvalidBypass(t *testing.T) {
	errExpected := errors.New("proxy called")
	pf := ProxyFactory(logging.NoOp, proxy.FactoryFunc(func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		return func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
			return nil, errExpected
		}, nil
	}))

	tc := `{"type": "not a valid type"}`
	cfg := map[string]interface{}{}
	if err := json.Unmarshal([]byte(tc), &cfg); err != nil {
		t.Error(err)
		return
	}
	p, err := pf.New(&config.EndpointConfig{
		ExtraConfig: map[string]interface{}{
			Namespace: cfg,
		},
	})
	if err != nil {
		t.Errorf("unexpected error %s", err.Error())
		return
	}
	if _, err := p(context.Background(), &proxy.Request{Body: io.NopCloser(bytes.NewBufferString(""))}); err != errExpected {
		t.Errorf("unexpected error %v", err)
	}
}

func TestProxyFactory_validationFail(t *testing.T) {
	errExpected := "jsonschema validation failed with"
	pf := ProxyFactory(logging.NoOp, proxy.FactoryFunc(func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		return func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
			t.Error("proxy called!")
			return nil, nil
		}, nil
	}))

	for _, tc := range []string{
		`{"type": "string"}`,
		`{"type": "array"}`,
		`{"type": "boolean"}`,
		`{"type": "number"}`,
		`{"type": "integer"}`,
	} {
		cfg := map[string]interface{}{}
		if err := json.Unmarshal([]byte(tc), &cfg); err != nil {
			t.Error(err)
			return
		}
		p, err := pf.New(&config.EndpointConfig{
			ExtraConfig: map[string]interface{}{
				Namespace: cfg,
			},
		})
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
			return
		}
		_, err = p(context.Background(), &proxy.Request{Body: io.NopCloser(bytes.NewBufferString("{}"))})
		if err == nil {
			t.Error("expecting error")
			return
		}
		if !strings.Contains(err.Error(), errExpected) {
			t.Errorf("unexpected error %s", err.Error())
		}
	}
}

func TestProxyFactory_validationOK(t *testing.T) {
	errExpected := errors.New("proxy called")
	pf := ProxyFactory(logging.NoOp, proxy.FactoryFunc(func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		return func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
			return nil, errExpected
		}, nil
	}))

	for _, tc := range []string{
		`{"type": "object"}`,
	} {
		cfg := map[string]interface{}{}
		if err := json.Unmarshal([]byte(tc), &cfg); err != nil {
			t.Error(err)
			return
		}
		p, err := pf.New(&config.EndpointConfig{
			ExtraConfig: map[string]interface{}{
				Namespace: cfg,
			},
		})
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
			return
		}
		_, err = p(context.Background(), &proxy.Request{Body: io.NopCloser(bytes.NewBufferString("{}"))})
		if err == nil {
			t.Error("expecting error")
			return
		}
		if err != errExpected {
			t.Errorf("unexpected error %s", err.Error())
		}
	}
}

func TestBakcnedFactory_validationOK(t *testing.T) {
	errExpected := errors.New("proxy called")
	bf := BackendFactory(logging.NoOp, func(cfg *config.Backend) proxy.Proxy {
		return func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
			return nil, errExpected
		}
	})

	for _, tc := range []string{
		`{"type": "object"}`,
	} {
		cfg := map[string]interface{}{}
		if err := json.Unmarshal([]byte(tc), &cfg); err != nil {
			t.Error(err)
			return
		}
		p := bf(&config.Backend{
			ExtraConfig: map[string]interface{}{
				Namespace: cfg,
			},
		})

		_, err := p(context.Background(), &proxy.Request{Body: io.NopCloser(bytes.NewBufferString("{}"))})
		if err == nil {
			t.Error("expecting error")
			return
		}
		if err != errExpected {
			t.Errorf("unexpected error %s", err.Error())
		}
	}
}

func TestProxyFactory_invalidJSON(t *testing.T) {
	errExpected := "invalid character '\\n' in string literal"
	statusExpected := http.StatusBadRequest
	pf := ProxyFactory(logging.NoOp, proxy.FactoryFunc(func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		return func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
			t.Error("proxy called!")
			return nil, nil
		}, nil
	}))

	for _, tc := range []string{
		`{"type": "object"}`,
	} {
		cfg := map[string]interface{}{}
		if err := json.Unmarshal([]byte(tc), &cfg); err != nil {
			t.Error(err)
			return
		}
		p, err := pf.New(&config.EndpointConfig{
			ExtraConfig: map[string]interface{}{
				Namespace: cfg,
			},
		})
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
			return
		}
		body := []byte(`{
			"name": "John 
Doe"
		}`)
		_, err = p(context.Background(), &proxy.Request{Body: io.NopCloser(bytes.NewReader(body))})
		if err == nil {
			t.Error("expecting error")
			return
		}
		if !strings.Contains(err.Error(), errExpected) {
			t.Errorf("unexpected error %s", err.Error())
			return
		}
		statusCodeErr, ok := err.(statusCodeError)
		if !ok {
			t.Errorf("unexpected error: %+v (%T)", err, err)
			return
		}
		if sc := statusCodeErr.StatusCode(); sc != statusExpected {
			t.Errorf("unexpected status code: %d", sc)
			return
		}
	}
}

func TestProxyFactory_emptyBody(t *testing.T) {
	errExpected := "could not validate an empty body"
	statusExpected := http.StatusBadRequest
	pf := ProxyFactory(logging.NoOp, proxy.FactoryFunc(func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		return func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
			t.Error("proxy called!")
			return nil, nil
		}, nil
	}))

	for _, tc := range []string{
		`{"type": "object"}`,
	} {
		cfg := map[string]interface{}{}
		if err := json.Unmarshal([]byte(tc), &cfg); err != nil {
			t.Error(err)
			return
		}
		p, err := pf.New(&config.EndpointConfig{
			ExtraConfig: map[string]interface{}{
				Namespace: cfg,
			},
		})
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
			return
		}
		_, err = p(context.Background(), &proxy.Request{Body: http.NoBody})
		if err == nil {
			t.Error("expecting error")
			return
		}
		if !strings.Contains(err.Error(), errExpected) {
			t.Errorf("unexpected error %s", err.Error())
			return
		}
		statusCodeErr, ok := err.(statusCodeError)
		if !ok {
			t.Errorf("unexpected error: %+v (%T)", err, err)
			return
		}
		if sc := statusCodeErr.StatusCode(); sc != statusExpected {
			t.Errorf("unexpected status code: %d", sc)
			return
		}
	}
}

type statusCodeError interface {
	error
	StatusCode() int
}
