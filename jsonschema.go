package jsonschema

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/luraproject/lura/v3/config"
	"github.com/luraproject/lura/v3/logging"
	"github.com/luraproject/lura/v3/proxy"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const Namespace = "validation/json-schema"

var ErrEmptyBody = &malformedError{err: errors.New("could not validate an empty body")}

// ProxyFactory creates an proxy factory over the injected one adding a JSON Schema
// validator middleware to the pipe when required
func ProxyFactory(logger logging.Logger, pf proxy.Factory) proxy.FactoryFunc {
	return proxy.FactoryFunc(func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		logPrefix := fmt.Sprintf("[ENDPOINT: %s %s][JSONSchema]", cfg.Method, cfg.Endpoint)
		next, err := pf.New(cfg)
		if err != nil {
			return proxy.NoopProxy, err
		}
		jschema := configGetter(cfg.ExtraConfig)
		if jschema == nil {
			return next, nil
		}

		c := jsonschema.NewCompiler()
		c.AddResource("./schema.json", jschema)
		s, err := c.Compile("./schema.json")
		if err != nil {
			logger.Error(logPrefix + " Parsing the definition:" + err.Error())
			return next, nil
		}
		logger.Debug(logPrefix + " Validator enabled")
		return newProxy(s, next), nil
	})
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

		b, err := jsonschema.UnmarshalJSON(bytes.NewReader(body))
		if err != nil {
			return nil, &malformedError{err: err}
		}

		err = schema.Validate(b)
		if err != nil {
			return nil, &validationError{error: err}
		}

		return next(ctx, r)
	}
}

func configGetter(cfg config.ExtraConfig) interface{} {
	v, ok := cfg[Namespace]
	if !ok {
		return nil
	}
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(v); err != nil {
		return nil
	}
	schema, err := jsonschema.UnmarshalJSON(buf)
	if err != nil {
		return nil
	}
	return schema
}

type validationError struct {
	error
}

func (v *validationError) Error() string {
	s := v.error.Error()
	if strings.HasPrefix(s, "jsonschema validation failed with") {
		if ss := strings.SplitN(s, "\n", 2); len(ss) == 2 {
			return ss[1]
		}
	}
	return s
}

func (*validationError) StatusCode() int {
	return http.StatusBadRequest
}

type malformedError struct {
	err error
}

func (m *malformedError) Error() string {
	return m.err.Error()
}

func (*malformedError) StatusCode() int {
	return http.StatusBadRequest
}
