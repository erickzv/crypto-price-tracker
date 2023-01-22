package utils

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"os"
	"sync"
	"text/template"
	"time"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
)

const filePath = "./src/index.html"

var greenDimPrintln = ColorPrintln(Green + Dim)

type Fetch struct {
	Data     []coinGeko
	Template *template.Template
}

// NewFetch creates a Fetch instance
func NewFetch() Fetch {
	// TODO validate the existance of /dist/styles.css
	greenDimPrintln("> Fetch Instancing")

	data := make([]coinGeko, 0)
	tmpl := template.New("index")
	initialize(tmpl, &data)

	greenDimPrintln("> Fetch Instanced\n")

	return Fetch{Data: data, Template: tmpl}
}

func initialize(tmpl *template.Template, data *[]coinGeko) (err error) {
	wg := sync.WaitGroup{}
	wg.Add(3)

	go func() {
		file := loadHTMLFile()
		tmpl, err = tmpl.Parse(file)
		wg.Done()
		greenDimPrintln("Fetch > index.html parsed successfully")
	}()

	go func() {
		*data, err = getJSON()
		wg.Done()
		greenDimPrintln("Fetch > Initial data getted")
	}()

	go func() {
		minifyJavaScript()
		wg.Done()
		greenDimPrintln("Fetch > Javascript minified")
	}()

	wg.Wait()
	return err
}

func (f *Fetch) Refresh() {
	for {
		data, err := getJSON()
		if err == nil {
			f.Data = data
		}

		time.Sleep(1 * time.Minute)
	}
}

func getJSON() ([]coinGeko, error) {
	url := "https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=100&page=1&sparkline=false"

	res, err := http.Get(url)
	if err != nil {
		return make([]coinGeko, 0), errors.New("unable to load page")
	}

	var cg []coinGeko
	err = json.NewDecoder(res.Body).Decode(&cg)
	if err != nil {
		return make([]coinGeko, 0), err
	}

	parseJSON(&cg)

	return cg, nil
}

func parseJSON(cg *[]coinGeko) {
	jsonData := *cg
	for i := range jsonData {
		price := jsonData[i].CurrentPrice
		change := jsonData[i].PriceChange24H

		jsonData[i].CurrentPrice = roundFloat(price, 4)
		jsonData[i].PriceChangePercentage24H = roundFloat(change, 2)
	}
}

// roundFloat rounds the val to the precision's decimal place
func roundFloat(val, precision float64) float64 {
	ratio := math.Pow(10, precision)
	return math.Round(val*ratio) / ratio
}

func loadHTMLFile() string {
	b, err := os.ReadFile(filePath)
	if err != nil {
		panic("unable to load file: " + filePath)
	}

	b = minifyHTML(b)
	return string(b)
}

func minifyHTML(b []byte) []byte {
	const mediaType = "text/html"

	m := minify.New()
	m.AddFunc(mediaType, html.Minify)

	b, err := m.Bytes(mediaType, b)
	if err != nil {
		panic("unable to minify: " + filePath)
	}

	return b
}

func minifyJavaScript() {
	const mediaType = "application/javascript"

	m := minify.New()
	m.AddFunc(mediaType, html.Minify)

	b, err := os.ReadFile("./src/script.js")
	if err != nil {
		panic("unable to find: /src/script.js")
	}

	b, err = m.Bytes(mediaType, b)
	if err != nil {
		panic("unable to minify: /src/script.js")
	}

	err = os.WriteFile("./dist/script.js", b, 0777)
	if err != nil {
		panic("unable to create /dist/script.js")
	}
}

type coinGeko struct {
	Name                     string  `json:"name"`
	Image                    string  `json:"image"`
	Symbol                   string  `json:"symbol"`
	CurrentPrice             float64 `json:"current_price"`
	PriceChange24H           float64 `json:"price_change_24h"`
	PriceChangePercentage24H float64 `json:"price_change_percentage_24h"`
}
