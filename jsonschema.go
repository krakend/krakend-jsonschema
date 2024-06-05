package jsonschema

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/luraproject/lura/v2/config"
	"github.com/luraproject/lura/v2/logging"
	"github.com/luraproject/lura/v2/proxy"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const Namespace = "github.com/devopsfaith/krakend-jsonschema"

var (
	ErrEmptyBody = &validationError{err: errors.New("could not validate an empty body")}
	ErrNoConfig  = errors.New("no config found")
)

// ProxyFactory creates an proxy factory over the injected one adding a JSON Schema
// validator middleware to the pipe when required
func ProxyFactory(logger logging.Logger, pf proxy.Factory) proxy.FactoryFunc {
	return proxy.FactoryFunc(func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		next, err := pf.New(cfg)
		if err != nil {
			return proxy.NoopProxy, err
		}
		schemaCfg, err := configGetter(cfg.ExtraConfig)
		if err != nil {
			if err != ErrNoConfig {
				return next, err
			}
			return next, nil
		}

		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("schema", schemaCfg); err != nil {
			logger.Error("[ENDPOINT: " + cfg.Endpoint + "][JSONSchema] Parsing the definition:" + err.Error())
			return next, nil
		}
		schema, err := compiler.Compile("schema")
		if err != nil {
			logger.Error("[ENDPOINT: " + cfg.Endpoint + "][JSONSchema] Parsing the definition:" + err.Error())
			return next, nil
		}

		logger.Debug("[ENDPOINT: " + cfg.Endpoint + "][JSONSchema] Validator enabled")
		return newProxy(schema, next), nil
	})
}

func BackendFactory(logger logging.Logger, bf proxy.BackendFactory) proxy.BackendFactory {
	return func(remote *config.Backend) proxy.Proxy {
		logPrefix := fmt.Sprintf("[BACKEND: %s][%s]", remote.URLPattern, Namespace)
		p := bf(remote)
		schemaCfg, err := configGetter(remote.ExtraConfig)
		if err != nil {
			if err != ErrNoConfig {
				logger.Error(logPrefix + " Parsing the definition:" + err.Error())
			}
			return p
		}
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("schema", schemaCfg); err != nil {
			logger.Error(logPrefix + " Parsing the definition:" + err.Error())
			return p
		}
		schema, err := compiler.Compile("schema")
		if err != nil {
			logger.Error(logPrefix + " Parsing the definition:" + err.Error())
			return p
		}

		logger.Debug(logPrefix, "Validator enabled")
		return newProxy(schema, p)
	}
}

func newProxy(schema *jsonschema.Schema, next proxy.Proxy) proxy.Proxy {
	return func(ctx context.Context, r *proxy.Request) (*proxy.Response, error) {
		if r.Body == nil {
			return nil, ErrEmptyBody
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		r.Body.Close()
		if len(body) == 0 {
			return nil, ErrEmptyBody
		}
		r.Body = io.NopCloser(bytes.NewBuffer(body))

		var doc interface{}
		if err := json.Unmarshal(body, &doc); err != nil {
			return nil, &validationError{err: err}
		}
		if err := schema.Validate(doc); err != nil {
			return nil, &validationError{err: err}
		}

		return next(ctx, r)
	}
}

func configGetter(cfg config.ExtraConfig) (interface{}, error) {
	v, ok := cfg[Namespace]
	if !ok {
		return nil, ErrNoConfig
	}
	return v, nil
}

type validationError struct {
	err error
}

func (m *validationError) Error() string {
	return m.err.Error()
}

func (*validationError) StatusCode() int {
	return http.StatusBadRequest
}
