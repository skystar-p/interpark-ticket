package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

var (
	URL        = "https://api-ticketfront.interpark.com/v1/goods/%s/playSeq/PlaySeq/%03d/REMAINSEAT"
	GOODSID    = "24011105"
	PlaySeqCnt = 2
)

var (
	config *ConfigStruct
)

type ConfigStruct struct {
	TelegramToken  string `env:"TELEGRAM_TOKEN,required"`
	TelegramChatID int64  `env:"TELEGRAM_CHAT_ID,required"`

	SleepDuration time.Duration `env:"SLEEP_DURATION" envDefault:"5s"`
}

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
	config = new(ConfigStruct)
	if err := env.Parse(config); err != nil {
		log.Fatal(err)
	}

	bot, err := telego.NewBot(config.TelegramToken)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Start checking...\n")

	if _, err := bot.SendMessage(tu.Message(tu.ID(config.TelegramChatID), "Start checking...")); err != nil {
		log.Printf("Failed to send message: %v\n", err)
	}

	for {
		seatData, err := checkSeat()
		if err != nil {
			log.Printf("Failed to check seat: %v\n", err)
			time.Sleep(config.SleepDuration)
			continue
		}

		for _, seat := range seatData {
			for _, s := range seat.RemainSeat {
				if s.RemainCnt > 0 {
					msg := fmt.Sprintf("Seat Found!!!: PlaySeq: %s, RemainCnt: %d, SeatGradeName: %s\n", s.PlaySeq, s.RemainCnt, s.SeatGradeName)
					if _, err := bot.SendMessage(tu.Message(tu.ID(config.TelegramChatID), msg)); err != nil {
						log.Printf("Failed to send message: %v\n", err)
					}
				}
				fmt.Printf("PlaySeq: %s, RemainCnt: %d, SeatGrade: %s, SeatGradeName: %s\n", s.PlaySeq, s.RemainCnt, s.SeatGrade, s.SeatGradeName)
			}
		}

		time.Sleep(config.SleepDuration)
	}

}

func checkSeat() ([]*SeatData, error) {
	var seatData []*SeatData
	for i := 1; i <= PlaySeqCnt; i++ {
		urlStr := fmt.Sprintf(URL, GOODSID, i)
		u, err := url.Parse(urlStr)
		if err != nil {
			return nil, err
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
			return nil, err
		}

		var seatResp SeatResponse
		if err := json.NewDecoder(resp.Body).Decode(&seatResp); err != nil {
			return nil, err
		}

		/*
			for _, seat := range seatResp.Data.RemainSeat {
				fmt.Printf("PlaySeq: %s, RemainCnt: %d, SeatGrade: %s, SeatGradeName: %s\n", seat.PlaySeq, seat.RemainCnt, seat.SeatGrade, seat.SeatGradeName)
			}
		*/

		seatData = append(seatData, &seatResp.Data)
	}

	return seatData, nil
}
