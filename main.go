package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/pkg/errors"
)

const (
	V2Endpoint     string = "https://qiita.com/api/v2"
	DefaultPerPage int    = 2
)

type Client struct {
	URL        *url.URL
	HTTPClient *http.Client
}

type Tag struct {
	FollowersCount int    `json:"followers_count"`
	IconURL        string `json:"icon_url"`
	ID             string `json:"id"`
	ItemsCount     int    `json:"items_count"`
}

func main() {

	client, err := NewClient(V2Endpoint)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	tags, err := client.listTags(ctx)
	if err != nil {
		log.Fatal(err)
	}

	for _, tag := range tags {
		fmt.Println(fmt.Sprintf("%s, %d", tag.ID, tag.FollowersCount))
	}
}

func NewClient(urlStr string) (*Client, error) {
	parsedURL, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse url: %s", urlStr)
	}
	return &Client{URL: parsedURL, HTTPClient: http.DefaultClient}, nil
}

func (c *Client) newRequest(ctx context.Context, method, spath string, body io.Reader) (*http.Request, error) {
	u := *c.URL
	u.Path = path.Join(c.URL.Path, spath)

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return req, nil
}

func (c *Client) listTags(ctx context.Context) ([]Tag, error) {
	req, err := c.newRequest(ctx, "GET", "/tags", nil)
	if err != nil {
		return nil, err
	}
	q := url.Values{
		"page":     []string{"1"},
		"per_page": []string{strconv.Itoa(DefaultPerPage)},
		"sort":     []string{"count"},
	}
	req.URL.RawQuery = q.Encode()

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	fmt.Println(res.StatusCode)
	if res.StatusCode != 200 {
		return nil, nil
	}

	var tagList []Tag
	if err := decodeBody(res, &tagList); err != nil {
		return nil, err
	}

	//totalCount, _ := strconv.Atoi(res.Header.Get("Total-Count"))
	totalCount := 10
	maxPage := int(totalCount / DefaultPerPage)
	fmt.Println(fmt.Sprintf("%d/%d", totalCount, DefaultPerPage))
	fmt.Println(maxPage)

	for page := 2; page <= maxPage; page++ {
		req, err := c.newRequest(ctx, "GET", "/tags", nil)
		if err != nil {
			return nil, err
		}

		q := url.Values{
			"page":     []string{strconv.Itoa(page)},
			"per_page": []string{strconv.Itoa(DefaultPerPage)},
			"sort":     []string{"count"},
		}
		req.URL.RawQuery = q.Encode()

		res, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}

		if res.StatusCode != 200 {
			fmt.Println("break!!")
			break
		}

		var tags []Tag
		if err := decodeBody(res, &tags); err != nil {
			return nil, err
		}
		tagList = append(tagList, tags...)
	}

	return tagList, nil
}

func decodeBody(resp *http.Response, out interface{}) error {
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	return decoder.Decode(out)
}
