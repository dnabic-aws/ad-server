package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
)

type adInfo struct {
	Id     string
	Result string
}

// export ADSERVER_ADRECOMMENDERURL="http://localhost:8085"
type config struct {
	AdRecommenderURL string `default:"http://localhost:8080"`
}

type client struct {
	HTTPClient *http.Client
	conf       config
}

func main() {
	var conf config
	err := envconfig.Process("adserver", &conf)
	if err != nil {
		log.Fatal(err.Error())
	}

	var client client
	client.conf = conf
	client.HTTPClient = &http.Client{
		Timeout: time.Second * 2,
	}

	r := mux.NewRouter()
	r.HandleFunc("/ad", client.ServeAd).Methods("GET")
	fmt.Println("Starting up on 8081")
	log.Fatal(http.ListenAndServe(":8081", r))
}

func (c *client) ServeAd(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	id := req.URL.Query().Get("id")
	if len(id) == 0 {
		fmt.Fprintln(w, "default")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Millisecond*2000))
	defer cancel()

	info, err := c.getAdInfo(ctx, id)
	if err != nil {
		log.Println(err)
	}
	fmt.Fprintln(w, info)
}

func (c *client) getAdInfo(ctx context.Context, id string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/find?id=%s", c.conf.AdRecommenderURL, id), nil)
	if err != nil {
		return "default", err
	}

	req = req.WithContext(ctx)

	return c.sendRequest(req)
}

func (c *client) sendRequest(req *http.Request) (string, error) {
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return "default", err
	}

	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMovedPermanently {
		return "default", fmt.Errorf("unknown error, status code: %d", res.StatusCode)
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return "default", readErr
	}

	var adInfo adInfo

	jsonErr := json.Unmarshal(body, &adInfo)
	if jsonErr != nil {
		return "default", jsonErr
	}

	if len(adInfo.Result) == 0 {
		return "default", jsonErr
	}
	return adInfo.Result, nil
}
