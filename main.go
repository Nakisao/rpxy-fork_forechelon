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
var proxyKey = os.Getenv("KEY")
var allowedPlaceID = os.Getenv("ALLOWED_PLACE_ID")

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
	// Health check endpoint.
	if string(ctx.Path()) == "/health" {
		ctx.SetStatusCode(200)
		ctx.SetBody([]byte("ok"))
		return
	}

	// Validate PROXYKEY header if KEY is set
	if proxyKey != "" && string(ctx.Request.Header.Peek("PROXYKEY")) != proxyKey {
		ctx.SetStatusCode(407)
		ctx.SetBody([]byte("Missing or invalid PROXYKEY header."))
		return
	}

	// Validate Place ID if ALLOWED_PLACE_ID is set
	if allowedPlaceID != "" {
		placeID := string(ctx.Request.Header.Peek("Roblox-Id"))
		if placeID == "" {
			ctx.SetStatusCode(403)
			ctx.SetBody([]byte("Missing Roblox-Id header."))
			return
		}
		if placeID != allowedPlaceID {
			ctx.SetStatusCode(403)
			ctx.SetBody([]byte("Unauthorized Place ID. Request rejected."))
			return
		}
	}

	if len(strings.SplitN(string(ctx.Request.Header.RequestURI())[1:], "/", 2)) < 2 {
		ctx.SetStatusCode(400)
		ctx.SetBody([]byte("URL format invalid."))
		return
	}

	response := makeRequest(ctx, 1)

	defer fasthttp.ReleaseResponse(response)

	body := response.Body()
	ctx.SetBody(body)
	ctx.SetStatusCode(response.StatusCode())
	response.Header.VisitAll(func (key, value []byte) {
		ctx.Response.Header.Set(string(key), string(value))
	})
}

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
	url := strings.SplitN(string(ctx.Request.Header.RequestURI())[1:], "/", 2)
	req.SetRequestURI("https://" + url[0] + ".roblox.com/" + url[1])
	req.SetBody(ctx.Request.Body())
	ctx.Request.Header.VisitAll(func (key, value []byte) {
		req.Header.Set(string(key), string(value))
	})
	req.Header.Set("User-Agent", "RoProxy")
	req.Header.Del("Roblox-Id")
	req.Header.Del("PROXYKEY")
	resp := fasthttp.AcquireResponse()

	err := client.Do(req, resp)

    if err != nil {
		fasthttp.ReleaseResponse(resp)
        return makeRequest(ctx, attempt + 1)
    } else {
		return resp
	}
}