package interpreter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"

	"time"

	"github.com/nlopes/slack"
)

var patternMap map[*regexp.Regexp]Message

func init() {
	databaseFile := os.Getenv("DB_FILE")
	if databaseFile != "" {
		d, err := ioutil.ReadFile(databaseFile)
		if err != nil {
			log.Println("Error in reading database file")
			panic(err)
		}
		var databaseFile struct {
			Messages []Message `json:"messages"`
		}

		err = json.Unmarshal(d, &databaseFile)
		if err != nil {
			log.Println("Error in decoding database file")
			panic(err)
		}
		patternMap = make(map[*regexp.Regexp]Message)
		for _, v := range databaseFile.Messages {
			patternMap[v.GetRegex()] = v
		}
	}
}

func ProcessQuery(q string, api *slack.Client, msg *slack.MessageEvent) slack.PostMessageParameters {
	params := GetSlackMessage()
	attachment := &params.Attachments[0]

	for k, v := range patternMap {
		fmt.Println(k, q, k.MatchString(q))
		if k.MatchString(q) {
			if v.Response != "" {
				attachment.Pretext = v.Response
				return params
			}
			switch v.Category {
			case "Show Qube Wire Transaction Report of company":
				r, err := GetQWCompanyTransactions(q)
				if len(r) == 0 {
					attachment.Pretext = "No transactions found"
					return params
				}
				if err != nil {
					log.Println("Error:", err)
					attachment.Pretext = err.Error()
					return params
				}

				f := CreateCSVOfTransactions(r)

				_, err = api.UploadFile(slack.FileUploadParameters{
					File:           f,
					Filename:       f,
					Filetype:       "csv",
					Title:          "Company Transaction Report",
					Channels:       []string{msg.Channel},
					InitialComment: "Transaction Report for company with id `" + r[0].Data.Company + "` as of `" + time.Now().UTC().String() + "`",
				})

				if err != nil {
					log.Println("Error:", err)
					attachment.Pretext = err.Error()
					return params
				}
				attachment.Pretext = "Report has been uploaded"
				return params

			case "Show Justickets Order":
				order, err := GetOrder(q)
				if err != nil {
					log.Println("Error:", err)
					attachment.Pretext = err.Error()
					return params
				}
				order.FormatSlackMessage(attachment)
				return params

			case "Show Justickets Bill":
				order, err := GetOrder(q)
				if err != nil {
					log.Println("Error:", err)
					attachment.Pretext = err.Error()
					return params
				}
				order.FormatSlackMessageForBill(attachment)
				return params

			case "Staging Report is Down":
				r, err := GetReportStatus(true)
				if err != nil {
					log.Println("Error:", err)
					attachment.Pretext = err.Error()
					return params
				}
				r.FormatSlackMessage(attachment)
				return params

			case "Report is Down":
				r, err := GetReportStatus(false)
				if err != nil {
					log.Println("Error:", err)
					attachment.Pretext = err.Error()
					return params
				}
				r.FormatSlackMessage(attachment)
				return params
			default:

			}
		}
	}
	return params
}

func FormatSlackMessageReport(attachment *slack.Attachment) {
	attachment.Pretext = "Staging report is down"
}
