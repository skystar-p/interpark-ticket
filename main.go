package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
)

var (
	URL        = "https://api-ticketfront.interpark.com/v1/goods/%s/playSeq/PlaySeq/%03d/REMAINSEAT"
	GOODSID    = "24011105"
	PlaySeqCnt = 2
)

type SeatResponse struct {
	Data SeatData `json:"data"`
}

type SeatData struct {
	RemainSeat []Seat `json:"remainSeat"`
}

type Seat struct {
	PlaySeq       string `json:"playSeq"`
	RemainCnt     int    `json:"remainCnt"`
	SeatGrade     string `json:"seatGrade"`
	SeatGradeName string `json:"seatGradeName"`
}

func main() {
	for i := 1; i <= PlaySeqCnt; i++ {
		urlStr := fmt.Sprintf(URL, GOODSID, i)
		u, err := url.Parse(urlStr)
		if err != nil {
			log.Fatal(err)
		}

		headers := http.Header{
			"User-Agent":      []string{"Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0"},
			"Accept":          []string{"application/json, text/plain, */*"},
			"Accept-Language": []string{"ko-KR,en-US;q=0.7,en;q=0.3"},
			"Accept-Encoding": []string{"gzip, deflate, br, zstd"},
			"Origin":          []string{"https://tickets.interpark.com"},
		}

		req := &http.Request{
			Method: "GET",
			URL:    u,
			Header: headers,
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		var seatResp SeatResponse
		if err := json.NewDecoder(resp.Body).Decode(&seatResp); err != nil {
			log.Fatal(err)
		}

		for _, seat := range seatResp.Data.RemainSeat {
			fmt.Printf("PlaySeq: %s, RemainCnt: %d, SeatGrade: %s, SeatGradeName: %s\n", seat.PlaySeq, seat.RemainCnt, seat.SeatGrade, seat.SeatGradeName)
		}
	}

}
