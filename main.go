func makeRequest(ctx *fasthttp.RequestCtx, attempt int) *fasthttp.Response {
	if attempt > retries {
		resp := fasthttp.AcquireResponse()
		resp.SetBody([]byte("Proxy failed to connect. Please try again."))
		resp.SetStatusCode(500)
		return resp
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.Header.SetMethod(string(ctx.Method()))

	// Get the path of the request URI
	urlPath := strings.SplitN(string(ctx.Request.Header.RequestURI())[1:], "/", 2)

	// Check if the URL matches the specific paths
	if strings.HasPrefix(urlPath[0], "ca-1394-report") || strings.HasPrefix(urlPath[0], "illegal-content-reporting") {
		// Handle these special cases directly without redirecting or modifying them
		req.SetRequestURI("https://roblox.com/" + urlPath[0])
	} else {
		// Otherwise, continue as usual for other subdomains
		req.SetRequestURI("https://" + urlPath[0] + ".roblox.com/" + urlPath[1])
	}

	req.SetBody(ctx.Request.Body())
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		req.Header.Set(string(key), string(value))
	})
	req.Header.Set("User-Agent", "RoProxy")
	req.Header.Del("Roblox-Id")

	resp := fasthttp.AcquireResponse()

	err := client.Do(req, resp)

	if err != nil {
		fasthttp.ReleaseResponse(resp)
		return makeRequest(ctx, attempt+1)
	} else {
		return resp
	}
}
