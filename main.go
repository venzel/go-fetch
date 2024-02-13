/**
 * @author Enéas Almeida <eneas.eng@yahoo.com>
 * @description O exemplo a seguir realiza um fetch de 100 fotos de um servidor remoto e armazena em um map.
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Response struct {
	Message string
	Body    []byte
	Time    float64
}

type Node struct {
	ID       int
	Attempts int
}

type Photo struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Url       string `json:"url"`
	Thumbnail string `json:"thumbnailUrl"`
}

func fetch(node *Node, chr chan *Response, chn chan *Node, wg *sync.WaitGroup) {
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

	defer cancel()

	client := &http.Client{}

	endpoint := "https://jsonplaceholder.typicode.com/photos/" + strconv.Itoa(node.ID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)

	if err != nil {
		chn <- node
		wg.Done()
		return
	}

	res, err := client.Do(req)

	if err != nil || res.StatusCode != http.StatusOK {
		chn <- node
		wg.Done()
		return
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)

	if err != nil {
		chn <- node
		wg.Done()
		return
	}

	chr <- &Response{
		Message: "Success",
		Body:    body,
		Time:    time.Since(start).Seconds(),
	}

	wg.Done()
}

func parse(body *[]byte) (*Photo, error) {
	photo := &Photo{}

	err := json.Unmarshal(*body, photo)

	if err != nil {
		return nil, err
	}

	return photo, nil
}

func print(photos *map[int]*Photo, start *time.Time) {
	fmt.Println("Photos:")

	for i := 0; i < len(*photos); i++ {
		photo, exists := (*photos)[i+1]

		if !exists {
			continue
		}

		fmt.Println(photo.ID, photo.Thumbnail)
	}

	fmt.Println("Total:", len(*photos))

	fmt.Printf("Tempo de execução: %.2fs\n", time.Since(*start).Seconds())
}

func main() {
	start := time.Now()

	ids := []int{}
	requests := 4500

	for i := 0; i < requests; i++ {
		ids = append(ids, i+1)
	}

	chr := make(chan *Response)
	chn := make(chan *Node)
	wg := sync.WaitGroup{}

	for i := 0; i < requests; i++ {
		wg.Add(1)
		go fetch(&Node{ID: ids[i], Attempts: 0}, chr, chn, &wg)
	}

	photos := make(map[int]*Photo)

	go func() {
		for {
			select {
			case res := <-chr:
				photo, err := parse(&res.Body)
				if err != nil {
					continue
				}
				photos[photo.ID] = photo
			case node := <-chn:
				node.Attempts++
				if node.Attempts <= 3 {
					fmt.Println(node.ID, "Retrying...", node.Attempts)
					wg.Add(1)
					go fetch(node, chr, chn, &wg)
				}
			default:
				continue
			}
		}
	}()

	wg.Wait()

	print(&photos, &start)
}
