# Gravity Presenter for Restful API

This gravity presenter is useful to generate restful API with no code.

## API Customization

Gravity Presenter provides a easy way to customize API, you can just write simple JSON configurations to design API and its behaviors.

### API Definition

First of all, you have to define API with the following settings:

```json
{
	"method": "post",
	"uri": "/v1/user/getPreTransferInfo",
	"query": {
		"table": "accounts",
		"condition": {
			"name": "phone",
			"value": "body.ReqBody.MobilePhone",
			"operator": "="
		}
	},
	"response": {
		"state": {
			"no_results": {
				"contentType": "application/json",
				"code": 404,
				"template": "getPreTransferInfo.no_results.tmpl"
			},
			"success": {
				"contentType": "application/json",
				"code": 200,
				"template": "getPreTransferInfo.success.tmpl"
			}
		}
	}
```

With above settings, a restful API `/v1/user/getPreTransferInfo` will be generated and exposed. When this API is getting called, it can execute query from `accounts` table by using column `phone` with `ReqBody.MobilePhone` of HTTP request body (assume Content-Type is JSON).

In the example, there is definition for `no_results` and `success` states to determine response of the API. You can set `contentType` and `code` to define necessary API behaviors and render content by using specific template.

### Content Template

Template can be customized to present data for API response:

```json
{
	"RCode": "4001",
	"AccountInfo": {
		"BankCode": "013",
		"MobilePhone": "{{ (index .Records 0).phone }}",
		"AccountType": "{{ (index .Records 0).type }}",
		"AccountName": "{{ (index .Records 0).name }}"
	}
}
```

## License

Licensed under the MIT License

## Authors

Copyright(c) 2020 Fred Chien <<fred@brobridge.com>>
