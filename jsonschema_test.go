package jsonschema

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
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
	if _, err := p(context.Background(), &proxy.Request{Body: ioutil.NopCloser(bytes.NewBufferString(""))}); err != errExpected {
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
	if _, err := p(context.Background(), &proxy.Request{Body: ioutil.NopCloser(bytes.NewBufferString(""))}); err != errExpected {
		t.Errorf("unexpected error %v", err)
	}
}

func TestProxyFactory_validationFail(t *testing.T) {
	errExpected := "- (root): Invalid type. Expected:"
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
		_, err = p(context.Background(), &proxy.Request{Body: ioutil.NopCloser(bytes.NewBufferString("{}"))})
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
		_, err = p(context.Background(), &proxy.Request{Body: ioutil.NopCloser(bytes.NewBufferString("{}"))})
		if err == nil {
			t.Error("expecting error")
			return
		}
		if err != errExpected {
			t.Errorf("unexpected error %s", err.Error())
		}
	}
}

func TestProxyFactory_emptyRequestBody(t *testing.T) {
	pf := ProxyFactory(logging.NoOp, proxy.FactoryFunc(func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		return func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
			t.Error("proxy called!")
			return nil, nil
		}, nil
	}))

	cfg := map[string]interface{}{}
	if err := json.Unmarshal([]byte(`{"type": "integer"}`), &cfg); err != nil {
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
	if err != ErrEmptyBody {
		t.Error("expecting error empty body")
	}
}
