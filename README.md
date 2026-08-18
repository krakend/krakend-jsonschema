# krakend-jsonschema
A JSON schema validator for the KrakenD API Gateway

## Usage
Include in your `krakend.json` the JSON Schema configuration associated to every `endpoint` needing it. For instance:

```
{
	"version": "2",
	"endpoints": [
		{
			"endpoint": "/foo",
			"extra_config": {
				"validation/json-schema": {
					YOUR SCHEMA HERE
				}
			}
		}  
	]
}
```
The configuration key `"validation/json-schema"` takes directly as value the schema definition. 
Examples of schema can be found [here](http://json-schema.org/learn/miscellaneous-examples.html)
