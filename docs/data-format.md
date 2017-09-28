Data Format
===========

The basic structure of the JSON returned from a request to /index is:

``` json
{
	'Registry': '<url, resolved relative to the document>',
	'Results": [
		{
			"Name": "some/repository",
			"Images": [
				{
					"Tags": ["<tag>", "<tag>"],
					"Digest": "<digest>",
					"MediaType": "<media type>",
					"OS": "<os>",
					"Arch": "<arch>",
					"Annotations": {
						"org.example.annotations.x": "<value>",
						"label:com.redhat.component": "<value>",
					}
				},
				{
					"<another image>"
				},
			],
			"Lists": [
				{
					"Tags": ["<tag>", "<tag>"],
					"Digest": "<digest>",
					"MediaType": "<media type>",
					"Images": [
						{ "<image data, no tags field>" },
						{ "<image data, no tags field>" },
					],
					"Annotations": {
						"org.example.annotations.y": "<value>"
					}
				}
			]
		},
		{
			"<another repository>"
		},
	]
}
```

