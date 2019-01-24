# krakend-jsonschema
a JSONschema validator for the KrakenD API Gateway

## Usage
Include in the `krakend.json` the configuration for this middleware with your schema definition inside:

```
"extra_config": {
    "github.com/devopsfaith/krakend-jsonschema": {
       YOUR SCHEMA HERE
    }
}
```

Examples of schema can be found [here](http://json-schema.org/learn/miscellaneous-examples.html)
