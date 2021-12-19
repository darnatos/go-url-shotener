package handler

import (
	"encoding/json"
	"fmt"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	"log"
	"net/http"
	"net/url"
	"time"
	"url-shortener/storage"
)

type response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"shortUrl"`
}

type handler struct {
	schema  string
	host    string
	storage storage.Service
}

func New(schema, host string, storage storage.Service) *router.Router {
	r := router.New()

	h := handler{schema, host, storage}
	r.POST("/encode/", responseHandler(h.encode))
	r.GET("/{shortLink}", h.redirect)
	r.GET("/{shortLink}/info", responseHandler(h.decode))

	return r
}

func responseHandler(h func(ctx *fasthttp.RequestCtx) (interface{}, int, error)) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		data, status, err := h(ctx)
		if err != nil {
			data = err.Error()
		}
		ctx.Response.Header.Set("Content-Type", "application/json")
		ctx.Response.SetStatusCode(status)
		err = json.NewEncoder(ctx.Response.BodyWriter()).Encode(response{Data: data, Success: err == nil})
		if err != nil {
			log.Printf("could not encode response to output: %v", err)
		}
	}
}

func (h handler) encode(ctx *fasthttp.RequestCtx) (interface{}, int, error) {
	var input struct {
		URL     string `json:"url"`
		Expires string `json:"expires"`
	}

	if err := json.Unmarshal(ctx.PostBody(), &input); err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("unable to decode JSON request body: %v", err)
	}

	uri, err := url.ParseRequestURI(input.URL)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid url")
	}

	layoutISO := "2006-01-02 15:04:05"
	expires, err := time.Parse(layoutISO, input.Expires)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid expiration date")
	}

	c, err := h.storage.Save(uri.String(), expires)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("could not store in database: %v", err)
	}

	u := url.URL{
		Scheme: h.schema,
		Host:   h.host,
		Path:   c,
	}

	fmt.Printf("Generate link: %v \n", u.String())
	return u.String(), http.StatusCreated, nil
}

func (h handler) decode(ctx *fasthttp.RequestCtx) (interface{}, int, error) {
	code := ctx.UserValue("shortLink").(string)

	model, err := h.storage.LoadInfo(code)
	if err != nil {
		return nil, http.StatusNotFound, fmt.Errorf("URL not found")
	}
	return model, http.StatusOK, nil
}

func (h handler) redirect(ctx *fasthttp.RequestCtx) {
	code := ctx.UserValue("shortLink").(string)

	uri, err := h.storage.Load(code)
	if err != nil {
		ctx.Response.Header.Set("Content-Type", "application/json")
		ctx.Response.SetStatusCode(http.StatusNotFound)
		return
	}
	ctx.Redirect(uri, http.StatusMovedPermanently)
}
