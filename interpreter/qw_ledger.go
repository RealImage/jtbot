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

	"strconv"

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
		Delta   int64  `json:"delta"`
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
	w.Write(GetHeader())

	for _, obj := range txns {
		lines := make(map[string]int64, 0)
		for _, v := range obj.Lines {
			lines[v.Account] = v.Delta
		}

		var record []string
		//Txn ID
		record = append(record, obj.ID)

		//Time Stamp
		record = append(record, obj.Timestamp.String())

		//Order ID
		record = append(record, obj.Data.Order)

		//Action
		record = append(record, obj.Data.Action)

		//Debit
		record = append(record, strconv.FormatInt(lines[obj.Data.Company+"DEBIT"], 10))

		//Credit
		record = append(record, strconv.FormatInt(lines[obj.Data.Company+"CREDIT"], 10))

		//QW.WIP
		record = append(record, strconv.FormatInt(lines["QW.WIP"], 10))

		//QW.Revenue
		record = append(record, strconv.FormatInt(lines["QW.REVENUE"], 10))

		//Stripe
		record = append(record, strconv.FormatInt(lines["STRIPE"], 10))

		//Credit Given
		record = append(record, strconv.FormatInt(lines["CREDITSGIVEN"], 10))

		w.Write(record)
	}
	w.Flush()
	return name
}

func GetHeader() []string {
	var record []string
	//Txn ID
	record = append(record, "Txn ID")

	//Time Stamp
	record = append(record, "Timestamp")

	//Order ID
	record = append(record, "Order ID")

	//Action
	record = append(record, "Action")

	//Debit
	record = append(record, "Debit")

	//Credit
	record = append(record, "Credit")

	//QW.WIP
	record = append(record, "QW.WIP")

	//QW.Revenue
	record = append(record, "QW.REVENUE")

	//Stripe
	record = append(record, "STRIPE")

	//Credit Given
	record = append(record, "CREDITSGIVEN")
	return record
}
