package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/rs/zerolog/log"
)

var (
	InterparkURL  = "https://api-ticketfront.interpark.com/v1/goods/%s/playSeq/PlaySeq/%03d/REMAINSEAT"
	TicketLinkURL = "https://mapi.ticketlink.co.kr/mapi/schedule/%s/grades?productClassCode=CONCERT&productId=%s"
)

var (
	config *ConfigStruct
)

type ConfigStruct struct {
	InterparkGoodsID      string `env:"INTERPARK_GOODS_ID,required"`
	InterparkPlaySeqCount int    `env:"INTERPARK_PLAY_SEQ_COUNT" envDefault:"1"`

	TicketLinkProductId   string   `env:"TICKETLINK_PRODUCT_ID,required"`
	TicketLinkScheduleIds []string `env:"TICKETLINK_SCHEDULE_IDS,required" envSeparator:","`

	TelegramToken   string  `env:"TELEGRAM_TOKEN,required"`
	TelegramChatIds []int64 `env:"TELEGRAM_CHAT_IDS,required" envSeparator:","`

	SleepDuration time.Duration `env:"SLEEP_DURATION" envDefault:"5s"`
	RenotifyAfter time.Duration `env:"RENOTIFY_AFTER" envDefault:"30s"`
}

type InterparkSeatResponse struct {
	Data InterparkSeatData `json:"data"`
}

type InterparkSeatData struct {
	RemainSeat []InterparkSeat `json:"remainSeat"`
}

type InterparkSeat struct {
	PlaySeq       string `json:"playSeq"`
	RemainCnt     int    `json:"remainCnt"`
	SeatGrade     string `json:"seatGrade"`
	SeatGradeName string `json:"seatGradeName"`
}

type TicketLinkSeatResponse struct {
	SeatData []TicketLinkSeat `json:"data"`
}

type TicketLinkSeat struct {
	Name       string `json:"name"`
	RemainCnt  int    `json:"remainCnt"`
	ScheduleID string `json:"scheduleId,omitempty`
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

	// only send to first chat id
	if _, err := bot.SendMessage(tu.Message(tu.ID(config.TelegramChatIds[0]), "Start checking...")); err != nil {
		log.Printf("Failed to send message: %v\n", err)
	}

	interparkSeatDataCache := make(map[InterparkSeat]time.Time)
	ticketLinkSeatDataCache := make(map[TicketLinkSeat]time.Time)
	for {
		var wg sync.WaitGroup
		wg.Add(2)
		// interpark
		go func() {
			defer wg.Done()
			log.Printf("Checking interpark...\n")
			seatData, err := checkInterparkSeat()
			if err != nil {
				log.Printf("Failed to check seat: %v\n", err)
				return
			}

			noAlarm := false
			for _, seat := range seatData {
				for _, s := range seat.RemainSeat {
					now := time.Now()
					if t, ok := interparkSeatDataCache[s]; ok {
						if now.Sub(t) < config.RenotifyAfter {
							noAlarm = true
						} else {
							interparkSeatDataCache[s] = now
						}
					} else {
						interparkSeatDataCache[s] = now
					}
					if s.RemainCnt > 0 && !noAlarm {
						msg := fmt.Sprintf("<Interpark> Seat Found!!!: PlaySeq: %s, RemainCnt: %d, SeatGradeName: %s\n", s.PlaySeq, s.RemainCnt, s.SeatGradeName)
						if err := sendTelegramMessage(bot, msg); err != nil {
							log.Printf("Failed to send message: %v\n", err)
						}
					}
					log.Printf("<Interpark> PlaySeq: %s, RemainCnt: %d, SeatGrade: %s, SeatGradeName: %s\n", s.PlaySeq, s.RemainCnt, s.SeatGrade, s.SeatGradeName)
				}
			}
		}()
		// ticketlink
		go func() {
			defer wg.Done()
			log.Printf("Checking ticketlink...\n")
			seats, err := checkTicketLinkSeat()
			if err != nil {
				log.Printf("Failed to check seat: %v\n", err)
				return
			}

			noAlarm := false
			for _, seat := range seats {
				now := time.Now()
				if t, ok := ticketLinkSeatDataCache[*seat]; ok {
					if now.Sub(t) < config.RenotifyAfter {
						noAlarm = true
					} else {
						ticketLinkSeatDataCache[*seat] = now
					}
				} else {
					ticketLinkSeatDataCache[*seat] = now
				}
				if seat.RemainCnt > 0 && !noAlarm {
					msg := fmt.Sprintf("<TicketLink> Seat Found!!!: Name: %s, RemainCnt: %d, ScheduleID: %s\n", seat.Name, seat.RemainCnt, seat.ScheduleID)
					if err := sendTelegramMessage(bot, msg); err != nil {
						log.Printf("Failed to send message: %v\n", err)
					}
				}
				log.Printf("<TicketLink> Name: %s, RemainCnt: %d, ScheduleID: %s\n", seat.Name, seat.RemainCnt, seat.ScheduleID)
			}
		}()

		wg.Wait()

		time.Sleep(config.SleepDuration)
	}

}

func checkInterparkSeat() ([]*InterparkSeatData, error) {
	var seatData []*InterparkSeatData
	for i := 1; i <= config.InterparkPlaySeqCount; i++ {
		urlStr := fmt.Sprintf(InterparkURL, config.InterparkGoodsID, i)
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

		var seatResp InterparkSeatResponse
		if err := json.NewDecoder(resp.Body).Decode(&seatResp); err != nil {
			return nil, err
		}

		seatData = append(seatData, &seatResp.Data)
	}

	return seatData, nil
}

func checkTicketLinkSeat() ([]*TicketLinkSeat, error) {
	var seats []*TicketLinkSeat
	for _, scheduleID := range config.TicketLinkScheduleIds {
		urlStr := fmt.Sprintf(TicketLinkURL, scheduleID, config.TicketLinkProductId)
		u, err := url.Parse(urlStr)
		if err != nil {
			return nil, err
		}

		headers := http.Header{
			"User-Agent":      []string{"Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0"},
			"Accept":          []string{"application/json, text/plain, */*"},
			"Accept-Language": []string{"ko-KR,en-US;q=0.7,en;q=0.3"},
			"Accept-Encoding": []string{"gzip, deflate, br, zstd"},
			"Origin":          []string{"https://www.ticketlink.co.kr"},
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

		var seatResp TicketLinkSeatResponse
		if err := json.NewDecoder(resp.Body).Decode(&seatResp); err != nil {
			return nil, err
		}

		for _, seat := range seatResp.SeatData {
			seat.ScheduleID = scheduleID
			seats = append(seats, &seat)
		}
	}

	return seats, nil
}

func sendTelegramMessage(bot *telego.Bot, msg string) error {
	for _, chatID := range config.TelegramChatIds {
		if _, err := bot.SendMessage(tu.Message(tu.ID(chatID), msg)); err != nil {
			return err
		}
	}
	return nil
}
