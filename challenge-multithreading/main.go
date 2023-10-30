package main

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/maphash"
	"math/rand"
	"net/http"
	"time"
)

var ceps = []string{
	"77006-130",
	"88330-765",
	"31710-550",
	"57307-125",
	"87300-142",
	"53425-460",
	"76824-600",
	"69151-702",
	"59124-690",
	"72880-455",
}

type CepService struct {
	Name     string
	Url      string
	Response map[string]interface{}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// get a random cep to avoid being blocked by the api services
	random := rand.New(rand.NewSource(int64(new(maphash.Hash).Sum64())))
	cep := ceps[random.Intn(len(ceps))]
	fmt.Println("CEP:", cep)

	services := []CepService{
		{
			Name: "apicep",
			Url:  "https://cdn.apicep.com/file/apicep/" + cep + ".json",
		},
		{
			Name: "viacep",
			Url:  "http://viacep.com.br/ws/" + cep + "/json/",
		},
	}

	ch := make(chan CepService, 1)

	for _, service := range services {
		go fetch(ctx, service, ch)
	}

	select {
	case <-ctx.Done():
		fmt.Println("Request timeout and was canceled by the context")
		close(ch)
	case result := <-ch:
		fmt.Println("Faster Service:", result.Name)
		s, _ := json.MarshalIndent(result.Response, "", "\t")
		fmt.Print(string(s))
	}
}

func fetch(ctx context.Context, service CepService, ch chan<- CepService) {
	// Force a delay to test the timeout
	// if service.Name == "viacep" {
	// 	time.Sleep(3 * time.Second)
	// }
	client := http.Client{}

	// create a new http request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, service.Url, nil)
	if err != nil {
		fmt.Println("Error building request:", err)
		return
	}

	// send the request
	res, err := client.Do(req)
	if res.StatusCode != http.StatusOK {
		fmt.Println("Error doing request:", err)
		return
	}

	if err != nil {
		fmt.Println(err)
		return
	}

	response := make(map[string]interface{})
	err = json.NewDecoder(res.Body).Decode(&response)
	service.Response = response

	if err != nil {
		fmt.Println("Error decoding response:", err)
		return
	}

	ch <- service
}
