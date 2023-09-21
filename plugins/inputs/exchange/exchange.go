//go:generate ../../../tools/readme_config_includer/generator
package exchange

import (
	_ "embed"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
)

//go:embed sample.conf
var sampleConfig string

const (
	currencyApiEndpoint = "https://api.apilayer.com"
	currencyApiResource = "/currency_data/live"
)

type Exchange struct {
	APIKey           string   `toml:"apikey"`
	BaseCurrency     string   `toml:"base_currency"`
	TargetCurrencies []string `toml:"target_currencies"`

	parserFunc  telegraf.ParserFunc
	fullApiLink string
}

func (*Exchange) SampleConfig() string {
	return sampleConfig
}

// Init is for setup, and validating config.
func (e *Exchange) Init() error {
	// We cannot access API without token.
	if e.APIKey == "" {
		return fmt.Errorf("'api_token' cannot be blank")
	}

	// We cannot access API without base_currency.
	if e.BaseCurrency == "" {
		return fmt.Errorf("'base_currency' cannot be blank")
	}

	return nil
}

// Gather takes in an accumulator and adds the metrics that the Input
// gathers. This is called every agent.interval
func (e *Exchange) Gather(acc telegraf.Accumulator) error {
	res, err := e.makeApiRequest()

	// Checks if request was successful.
	if err != nil {
		return fmt.Errorf("[url=%s]: %w", e.fullApiLink, err)
	}

	// Checks if body was parsed successfully.
	if err = parseResponse(res, e, acc); err != nil {
		return fmt.Errorf("[url=%s]: %w", e.fullApiLink, err)
	}

	return nil
}

// Init value
func init() {
	inputs.Add("exchange", func() telegraf.Input { return &Exchange{} })
}

// SetParserFunc takes the data_format from the config and finds the right parser for that format
func (e *Exchange) SetParserFunc(fn telegraf.ParserFunc) {
	e.parserFunc = fn
}

// makeApiRequest makes request and return response.
func (e *Exchange) makeApiRequest() (*http.Response, error) {
	// Prepare http client
	client, req, err := e.prepareHttpClient()
	if err != nil {
		return nil, err
	}

	// Make request
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// Check response status
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("%v", res.Status)
	}

	return res, nil
}

// prepareHttpClient prepares request params and instance of http client.
func (e *Exchange) prepareHttpClient() (*http.Client, *http.Request, error) {
	// Create HTTP client.
	client := &http.Client{}

	// Prepare request params
	params := url.Values{}
	params.Add("source", e.BaseCurrency)
	params.Add("currencies", strings.Join(e.TargetCurrencies, ","))

	// Prepare URI.
	u, _ := url.ParseRequestURI(currencyApiEndpoint)
	u.Path = currencyApiResource
	u.RawQuery = params.Encode()
	// "http://example.com/path?param1=value1&param2=value2"
	e.fullApiLink = fmt.Sprintf("%v", u)

	req, err := http.NewRequest("GET", e.fullApiLink, nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("apikey", e.APIKey)

	return client, req, nil
}

// parseResponse parses response from the endpoint and adds fields to the accumulator.
func parseResponse(res *http.Response, e *Exchange, acc telegraf.Accumulator) error {
	defer res.Body.Close()

	// Read from body to slice of bytes.
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	// Instantiate a new parser for the new data.
	parser, err := e.parserFunc()
	if err != nil {
		return err
	}

	// Parse response body.
	metrics, err := parser.Parse(body)
	if err != nil {
		return err
	}

	// Write data to influxdb.
	for _, metric := range metrics {
		for _, field := range metric.FieldList() {

			if strings.Contains(field.Key, "quotes") {
				newName := strings.Replace(field.Key, "quotes_", "", -1)
				acc.AddFields(newName, map[string]interface{}{newName: field.Value}, nil)
			}
		}
	}

	return nil
}
