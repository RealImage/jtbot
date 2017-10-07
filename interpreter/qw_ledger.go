package interpreter

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"strings"

	"bytes"

	"github.com/maknahar/go-utils"
)

type QWCompanyTransactions struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Data      struct {
		Order      string `json:"order"`
		Action     string `json:"action"`
		Amount     int    `json:"amount"`
		Company    string `json:"company"`
		Product    string `json:"product"`
		ClientData struct {
			Type      []string `json:"type"`
			DcpAmount int      `json:"dcp_amount"`
		} `json:"client_data"`
	} `json:"data"`
	Lines []struct {
		Account string `json:"account"`
		Delta   int    `json:"delta"`
	} `json:"lines"`
}

type QWLedgerCompanyTransactionsRequestDTO struct {
	Query struct {
		Must struct {
			Terms []Terms `json:"terms"`
		} `json:"must"`
	} `json:"query"`
}

type Terms struct {
	Company string `json:"company"`
}

func GetQWCompanyTransactions(msg string) ([]*QWCompanyTransactions, error) {
	id := ""
	for _, v := range strings.Split(msg, " ") {
		if go_utils.IsValidUUIDV4(v) {
			id = v
			break
		}
	}

	payload := new(QWLedgerCompanyTransactionsRequestDTO)
	payload.Query.Must.Terms = append(payload.Query.Must.Terms, Terms{Company: id})
	pd, _ := json.Marshal(payload)

	url := os.Getenv("QW_LEDGER_URL")
	res, err := http.Post(url+"/v1/transactions/_search", "appliacation/json", bytes.NewBuffer(pd))
	if err != nil {
		return nil, fmt.Errorf("Error in getting company data %v", err)
	}

	d, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("Error in reading company data %v", err)
	}

	order := make([]*QWCompanyTransactions, 0)
	err = json.Unmarshal(d, &order)
	if err != nil {
		if res.StatusCode == http.StatusServiceUnavailable {
			return nil, errors.New("Service is unavailable")
		}
		log.Println("Error in decoding report response", err, go_utils.JsonPrettyPrint(string(d),
			"", "\t"))
		return nil, errors.New("Error in decoding response")
	}

	return order, nil
}

func CreateCSVOfTransactions(txns []*QWCompanyTransactions) string {
	// Create a csv file
	name := "transaction_report_" + txns[0].ID + time.Now().String() + ".csv"
	f, err := os.Create(name)
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()

	// Write Unmarshaled json data to CSV file
	w := csv.NewWriter(f)
	for _, obj := range txns {
		var record []string
		//Company ID
		record = append(record, obj.ID)

		//Time Stamp
		record = append(record, obj.Timestamp.String())

		w.Write(record)
	}
	w.Flush()
	return name
}
