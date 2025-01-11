package main

import (
	"log"
	"time"
	"os"
	"github.com/valyala/fasthttp"
	"strconv"
	"strings"
)

var timeout, _ = strconv.Atoi(os.Getenv("TIMEOUT"))
var retries, _ = strconv.Atoi(os.Getenv("RETRIES"))
var port = os.Getenv("PORT")
var host = os.Getenv("HOST") // Set this to the base URL of your proxy

var client *fasthttp.Client

func main() {
	h := requestHandler
	
	client = &fasthttp.Client{
		ReadTimeout: time.Duration(timeout) * time.Second,
		MaxIdleConnDuration: 60 * time.Second,
	}

	if err := fasthttp.ListenAndServe(":" + port, h); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	val, ok := os.LookupEnv("KEY")

	if ok && string(ctx.Request.Header.Peek("PROXYKEY")) != val {
		ctx.SetStatusCode(407)
		ctx.SetBody([]byte("Missing or invalid PROXYKEY header."))
		return
	}

	// Check for special URLs (do not redirect to roblox.com)
	if isSpecialURL(ctx) {
		// Handle special cases (no redirection, just handle like subdomains)
		response := makeRequest(ctx, 1, true)
		defer fasthttp.ReleaseResponse(response)

		body := response.Body()
		ctx.SetBody(body)
		ctx.SetStatusCode(response.StatusCode())
		response.Header.VisitAll(func (key, value []byte) {
			ctx.Response.Header.Set(string(key), string(value))
		})
		return
	}

	// Default request handling
	if len(strings.SplitN(string(ctx.Request.Header.RequestURI())[1:], "/", 2)) < 2 {
		ctx.SetStatusCode(400)
		ctx.SetBody([]byte("URL format invalid."))
		return
	}

	// Default request handling for other requests
	response := makeRequest(ctx, 1, false)

	defer fasthttp.ReleaseResponse(response)

	body := response.Body()
	ctx.SetBody(body)
	ctx.SetStatusCode(response.StatusCode())
	response.Header.VisitAll(func (key, value []byte) {
		ctx.Response.Header.Set(string(key), string(value))
	})
}

// Check if the URL matches special cases for "ca-1394-report" or "illegal-content-reporting"
func isSpecialURL(ctx *fasthttp.RequestCtx) bool {
	path := string(ctx.Request.URI().Path())
	return strings.HasPrefix(path, "/ca-1394-report") || strings.HasPrefix(path, "/illegal-content-reporting")
}

func makeRequest(ctx *fasthttp.RequestCtx, attempt int, isSpecial bool) *fasthttp.Response {
	if attempt > retries {
		resp := fasthttp.AcquireResponse()
		resp.SetBody([]byte("Proxy failed to connect. Please try again."))
		resp.SetStatusCode(500)

		return resp
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.Header.SetMethod(string(ctx.Method()))

	// For special URLs like ca-1394-report or illegal-content-reporting, don't redirect to main site.
	url := strings.SplitN(string(ctx.Request.Header.RequestURI())[1:], "/", 2)

	if isSpecial {
		// For special paths, handle them on the proxy site
		req.SetRequestURI("https://" + host + "/" + url[0] + "/" + url[1])
	} else {
		// Default behavior for other paths (redirect to roblox.com)
		req.SetRequestURI("https://" + url[0] + ".roblox.com/" + url[1])
	}

	req.SetBody(ctx.Request.Body())
	ctx.Request.Header.VisitAll(func (key, value []byte) {
		req.Header.Set(string(key), string(value))
	})
	req.Header.Set("User-Agent", "RoProxy")
	req.Header.Del("Roblox-Id")
	resp := fasthttp.AcquireResponse()

	err := client.Do(req, resp)

    if err != nil {
		fasthttp.ReleaseResponse(resp)
        return makeRequest(ctx, attempt + 1, isSpecial)
    } else {
		return resp
	}
}
