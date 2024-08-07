package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/rs/zerolog/log"
)

var (
	URL = "https://api-ticketfront.interpark.com/v1/goods/%s/playSeq/PlaySeq/%03d/REMAINSEAT"
)

var (
	config *ConfigStruct
)

type ConfigStruct struct {
	GoodsID      string `env:"GOODS_ID,required"`
	PlaySeqCount int    `env:"PLAY_SEQ_COUNT" envDefault:"1"`

	TelegramToken   string  `env:"TELEGRAM_TOKEN,required"`
	TelegramChatIds []int64 `env:"TELEGRAM_CHAT_IDS,required"`

	SleepDuration time.Duration `env:"SLEEP_DURATION" envDefault:"5s"`
	RenotifyAfter time.Duration `env:"RENOTIFY_AFTER" envDefault:"1m"`
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
		log.Fatal().Err(err)
	}

	bot, err := telego.NewBot(config.TelegramToken)
	if err != nil {
		log.Fatal().Err(err)
	}

	log.Printf("Start checking...\n")

	for _, chatId := range config.TelegramChatIds {
		if _, err := bot.SendMessage(tu.Message(tu.ID(chatId), "Start checking...")); err != nil {
			log.Printf("Failed to send message: %v\n", err)
		}
	}

	seatDataCache := make(map[Seat]time.Time)
	for {
		seatData, err := checkSeat()
		if err != nil {
			log.Printf("Failed to check seat: %v\n", err)
			time.Sleep(config.SleepDuration)
			continue
		}

		for _, seat := range seatData {
			for _, s := range seat.RemainSeat {
				now := time.Now()
				if t, ok := seatDataCache[s]; ok {
					if now.Sub(t) < config.RenotifyAfter {
						continue
					} else {
						seatDataCache[s] = now
					}
				} else {
					seatDataCache[s] = now
				}
				if s.RemainCnt > 0 {
					msg := fmt.Sprintf("Seat Found!!!: PlaySeq: %s, RemainCnt: %d, SeatGradeName: %s\n", s.PlaySeq, s.RemainCnt, s.SeatGradeName)
					for _, chatId := range config.TelegramChatIds {
						if _, err := bot.SendMessage(tu.Message(tu.ID(chatId), msg)); err != nil {
							log.Printf("Failed to send message: %v\n", err)
						}
					}
				}
				log.Printf("PlaySeq: %s, RemainCnt: %d, SeatGrade: %s, SeatGradeName: %s\n", s.PlaySeq, s.RemainCnt, s.SeatGrade, s.SeatGradeName)
			}
		}

		time.Sleep(config.SleepDuration)
	}

}

func checkSeat() ([]*SeatData, error) {
	var seatData []*SeatData
	for i := 1; i <= config.PlaySeqCount; i++ {
		urlStr := fmt.Sprintf(URL, config.GoodsID, i)
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

		seatData = append(seatData, &seatResp.Data)
	}

	return seatData, nil
}
