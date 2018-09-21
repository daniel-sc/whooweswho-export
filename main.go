package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const indexSplits = 4 // first column index for splits
var idToNameMap = map[int]string{}
var idList []int
var myClient = &http.Client{Timeout: 10 * time.Second}

// Item is an expense entry
type Item struct {
	Description string    `json:"description"`
	Amount      float32   `json:"amount"`
	Payer       int       `json:"payor_id"`
	Split       []Split   `json:"split"`
	Time        time.Time `json:"ctime"`
}

// Split of Item
type Split struct {
	ID    int     `json:"id"`
	Value float32 `json:"value"`
}

var outputFile = flag.String("output", "expenses.csv", "csv output file")
var startURL = flag.String("url", "", `url of sheet, e.g. "https://www.whooweswho.net/session#/sheets/1234/6789/expenses"`)
var nameMapParam = flag.String("names", "", `names to replace IDs, e.g. "123456->Arnold,987654->Schwarz"`)
var verbose = flag.Bool("v", false, "verbose output")
var skipHeader = flag.Bool("skip-header", false, "skip header line in csv")
var additionalHeaders = flag.String("headers", "", `additional request headers, e.g. "Cookie:session_cookie123,X-My-Header:42"`)

func main() {
	flag.Parse()

	// TODO fetch default split from https://www.whooweswho.net/api/Book/123456/Sheet/789789

	idToNameRegexp := regexp.MustCompile(`(\d+)->([^,]+),?`)
	idToNameMatches := idToNameRegexp.FindAllStringSubmatch(*nameMapParam, -1)
	for _, match := range idToNameMatches {
		id, err := strconv.Atoi(match[1])
		checkErr(err)
		idToNameMap[id] = match[2]
		if *verbose {
			log.Printf(`registered name "%s" for id "%d"`, match[2], id)
		}
	}

	bookAndSheet := regexp.MustCompile(`/([0-9]+)(?:/Sheet)?/([0-9]+)/`) // matches both browser url and api url
	bookAndSheetMatches := bookAndSheet.FindStringSubmatch(*startURL)
	if bookAndSheetMatches == nil {
		log.Fatalf(`Could not extract book/sheet from url: "%s" (provide via param "url")`, *startURL)
	}
	book := bookAndSheetMatches[1]
	sheet := bookAndSheetMatches[2]
	url := fmt.Sprintf("https://www.whooweswho.net/api/Book/%s/Sheet/%s/Row?order=-ctime", book, sheet)
	if *verbose {
		log.Println("query book=" + book + " and sheet=" + sheet + " via url: " + url)
	}

	items := make([]Item, 0)
	parseErr := getJSON(url, &items)
	checkErr(parseErr)

	if *verbose {
		for _, item := range items {
			log.Println(item)
		}
	}

	for _, item := range items {
		if _, present := idToNameMap[item.Payer]; !present {
			idToNameMap[item.Payer] = toName(item.Payer)
		}
		if item.Split != nil {
			for _, split := range item.Split {
				if _, present := idToNameMap[split.ID]; !present {
					idToNameMap[split.ID] = toName(split.ID)
				}
			}
		}
	}
	idList = make([]int, len(idToNameMap)) // ordered ids
	i := 0
	for id := range idToNameMap {
		idList[i] = id
		i = i + 1
	}
	if *verbose {
		log.Printf("Found the following involved persons: %v", idToNameMap)
	}

	fileWriter, fileErr := os.Create(*outputFile)
	checkErr(fileErr)
	defer fileWriter.Close()

	w := csv.NewWriter(fileWriter)

	if !*skipHeader {
		firstLine := make([]string, indexSplits+len(idToNameMap))
		firstLine[0] = "Time"
		firstLine[1] = "Amount"
		firstLine[2] = "Description"
		firstLine[3] = "Payer"
		for i, id := range idList {
			firstLine[indexSplits+i] = "Split " + toName(id)
		}
		w.Write(firstLine)
	}

	for _, item := range items {
		checkErr(w.Write(item.toCsv()))
	}
	w.Flush()
	checkErr(w.Error())

	log.Printf("Exported %d rows to file %s", len(items), *outputFile)
}

func getJSON(url string, target interface{}) error {

	req, err := http.NewRequest("GET", url, nil)
	checkErr(err)

	for _, h := range strings.Split(*additionalHeaders, ",") {
		nameAndValue := strings.Split(h, ":")
		req.Header.Add(nameAndValue[0], nameAndValue[1])
		if *verbose {
			log.Printf("Added requet header %s=%s", nameAndValue[0], nameAndValue[1])
		}
	}

	r, err := myClient.Do(req)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func toName(id int) string {
	if name, ok := idToNameMap[id]; ok {
		return name
	} else {
		return fmt.Sprintf("%d", id)
	}
}

func (item Item) toCsv() []string {
	result := make([]string, indexSplits+len(idList))

	result[0] = fmt.Sprintf("%s", item.Time)
	result[1] = fmt.Sprintf("%.2f", item.Amount)
	result[2] = strings.TrimSpace(item.Description)
	result[3] = toName(item.Payer)

	for i, id := range idList {
		if item.Split != nil {
			splitValue := ""
			for _, split := range item.Split {
				if split.ID == id {
					splitValue = fmt.Sprintf("%.2f", split.Value)
				}
			}
			result[i+indexSplits] = splitValue
		} else {
			result[i+indexSplits] = ""
		}
	}

	return result
}

func (item Item) String() string {
	return fmt.Sprintf("{Time: %s, Payer: %d, Amount: %5.2f, Description: %10s, Split: %v}", item.Time, item.Payer, item.Amount, item.Description, item.Split)
}

func (s Split) String() string {
	return fmt.Sprintf("{Person: %5s, Value: %3.1f}", toName(s.ID), s.Value)
}
