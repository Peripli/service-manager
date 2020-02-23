package util

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
)

// ListIterator lists the objects from the given url by loading one page at a time
type ListIterator struct {
	// DoRequest function executes the HTTP request, it is responsible for authentication
	DoRequest DoRequestOsbFunc
	// URL is the address of the resource to list
	URL string

	next string
	done bool
}

type listResponse struct {
	Token    string      `json:"token"`
	NumItems int64       `json:"num_items"`
	Items    interface{} `json:"items"`
}

// Next loads the next page of items
// items should be a pointer to a slice, that will be populated with the items from the current page,
// if nil, only the total number is returned
// maxItems is the maximum numbe of items to load with the next page, -1 - use server default, 0 - just get the count
// more is true if there are more items
// count is the total number of items, -1 if not available
func (li *ListIterator) Next(ctx context.Context, items interface{}, maxItems int) (more bool, count int64, err error) {
	itemsType := reflect.TypeOf(items)
	if itemsType != nil && (itemsType.Kind() != reflect.Ptr || itemsType.Elem().Kind() != reflect.Slice) {
		return false, -1, fmt.Errorf("items should be nil or a pointer to a slice, but got %v", itemsType)
	}

	if li.done {
		return false, -1, errors.New("iteration already complete")
	}

	params := map[string]string{}
	if maxItems >= 0 {
		params["max_items"] = strconv.Itoa(maxItems)
	}
	if li.next != "" {
		params["token"] = li.next
	}

	method := http.MethodGet
	url := li.URL
	response, err := SendRequest(ctx, li.DoRequest, method, url, params, nil, http.DefaultClient)
	if err != nil {
		return false, -1, fmt.Errorf("Error sending request %s %s: %s", method, url, err)
	}
	if response.Request != nil {
		url = response.Request.URL.String() // should include also the query params
	}
	if response.StatusCode != http.StatusOK {
		return false, -1, HandleResponseError(response)
	}
	body, err := BodyToBytes(response.Body)
	if err != nil {
		return false, -1, fmt.Errorf("error reading response body of request %s %s: %s",
			method, url, err)
	}
	responseBody := listResponse{Items: items}
	if err = json.Unmarshal(body, &responseBody); err != nil {
		return false, -1, fmt.Errorf("error parsing response body of request %s %s: %s",
			method, url, err)
	}

	li.next = responseBody.Token
	li.done = li.next == ""
	return !li.done, responseBody.NumItems, nil
}

// ListAll retrieves all the objects from the given url by loading all the pages
// items should be a pointer to a slice, that will be populated with all the items
// doRequest function executes the HTTP request, it is responsible for authentication
func ListAll(ctx context.Context, doRequest DoRequestOsbFunc, url string, items interface{}) error {
	itemsType := reflect.TypeOf(items)
	if itemsType.Kind() != reflect.Ptr || itemsType.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("items should be a pointer to a slice, but got %v", itemsType)
	}

	allItems := reflect.MakeSlice(itemsType.Elem(), 0, 0)
	iter := ListIterator{
		URL:       url,
		DoRequest: doRequest,
	}
	more := true
	for more {
		var err error
		pageSlice := reflect.New(itemsType.Elem())
		more, _, err = iter.Next(ctx, pageSlice.Interface(), -1)
		if err != nil {
			return err
		}
		allItems = reflect.AppendSlice(allItems, pageSlice.Elem())
	}
	reflect.ValueOf(items).Elem().Set(allItems)
	return nil
}
