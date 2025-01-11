package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

var timeout, _ = strconv.Atoi(os.Getenv("TIMEOUT"))
var retries, _ = strconv.Atoi(os.Getenv("RETRIES"))
var port = os.Getenv("PORT")

var client *fasthttp.Client

// Allowed paths
var allowedPaths = []string{
	"/illegal-content-reporting",
	"/ca-1393-report",
}

func main() {
	h := requestHandler

	client = &fasthttp.Client{
		ReadTimeout:        time.Duration(timeout) * time.Second,
		MaxIdleConnDuration: 60 * time.Second,
	}

	if err := fasthttp.ListenAndServe(":"+port, h); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	// Validate the PROXYKEY header
	val, ok := os.LookupEnv("KEY")
	if ok && string(ctx.Request.Header.Peek("PROXYKEY")) != val {
		ctx.SetStatusCode(407)
		ctx.SetBody([]byte("Missing or invalid PROXYKEY header."))
		return
	}

	// Ensure the URL path is valid
	path := string(ctx.Path())
	if !isAllowedURL(path) {
		ctx.SetStatusCode(403) // Forbidden
		ctx.SetBody([]byte("Forbidden: Invalid URL path"))
		return
	}

	// Make the request to the target Roblox URL
	response := makeRequest(ctx, 1)
	defer fasthttp.ReleaseResponse(response)

	// Set the response body and status code
	body := response.Body()
	ctx.SetBody(body)
	ctx.SetStatusCode(response.StatusCode())

	// Copy response headers
	response.Header.VisitAll(func(key, value []byte) {
		ctx.Response.Header.Set(string(key), string(value))
	})
}

// Function to check if the requested URL is allowed
func isAllowedURL(urlPath string) bool {
	for _, allowedPath := range allowedPaths {
		if strings.HasPrefix(urlPath, allowedPath) {
			return true
		}
	}
	return false
}

// Function to make the actual request to Roblox servers
func makeRequest(ctx *fasthttp.RequestCtx, attempt int) *fasthttp.Response {
	if attempt > retries {
		resp := fasthttp.AcquireResponse()
		resp.SetBody([]byte("Proxy failed to connect. Please try again."))
		resp.SetStatusCode(500)
		return resp
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	// Construct the request URI for Roblox
	urlParts := strings.SplitN(string(ctx.Request.URI().Path()), "/", 2)
	if len(urlParts) < 2 {
		resp := fasthttp.AcquireResponse()
		resp.SetBody([]byte("URL format invalid."))
		resp.SetStatusCode(400) // Bad Request
		return resp
	}

	// Set the request URI to the corresponding Roblox URL
	req.SetRequestURI("https://" + urlParts[0] + ".roblox.com/" + urlParts[1])
	req.SetBody(ctx.Request.Body())

	// Copy headers from the original request
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		req.Header.Set(string(key), string(value))
	})

	req.Header.Set("User-Agent", "RoProxy")
	req.Header.Del("Roblox-Id")

	resp := fasthttp.AcquireResponse()
	err := client.Do(req, resp)
	if err != nil {
		fasthttp.ReleaseResponse(resp)
		return makeRequest(ctx, attempt+1) // Retry on failure
	}
	return resp
}
