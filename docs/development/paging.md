# Paging on List Operations

All List endpoints return the list of entities page by page as defined in [specification](https://github.com/Peripli/specification/blob/2eaba9aa0baa877b57385925ad32faca4ce393f3/api.md#paging-parameters).

## Page response

```
{  
  "token": <base64 encoded paging_sequence of the last entity in items list>,
  "num_items": <total number of items in the result set>,
  "items": [
    {
      "id": "a62b83e8-1604-427d-b079-200ae9247b60",
      ...
    },
    ...
  ]
}
```

The `num_items` value **MAY NOT** be accurate the next time the client retrieves the result set or the next page in the result set.


## Token
Each entity that can be listed has an additional serial column `paging_sequence`.
The token parameter is the `paging_sequence` of the last entity in the page `base64` encoded.
Next page is requested when you provide this token in a subsequent request. On the Service Manager side
the token is decoded and `max_items` number of entities with `paging_sequence` > `paging_sequence from token` are returned
in the next page ordered by `paging_sequence` (which is auto-increment field and implies creation order). 
Token is generated from the `paging_sequence` of the last entity if there are more entities for the next page.
First page is requested with empty token or no token provided.

