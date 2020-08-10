package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"github.com/gorilla/websocket"
	"github.com/kelseyhightower/envconfig"
)

// Application Configuration
type appConfiguration struct {
	Hostname string   `default:"localhost"`
	Port     string   `default:"8080"`
	Symbols  []string `required:"true"`
}

// Currency information that will be served by this program
type currencyInfo struct {
	ID          string `json:"id"`
	FullName    string `json:"fullName"`
	Ask         string `json:"ask"`
	Bid         string `json:"bid"`
	Last        string `json:"last"`
	Open        string `json:"open"`
	Low         string `json:"low"`
	High        string `json:"high"`
	FeeCurrency string `json:"feeCurrency"`
}

// Concurrency control for currency information
var currencyInfoLock []sync.Mutex

// Currency information array
var currencies []currencyInfo

// Map from currency symbol to index in currencies array
var symbolToIndex map[string]int

var appConf appConfiguration

// Configure app based on environment
// Default hostname: localhost
// Default port: 8080
// Default symbols to include: BTCUSD, ETHBTC
func doConfig() {
	err := envconfig.Process("dchoweller_crypto", &appConf)
	if err != nil { // If error reading environment, use default values
		appConf.Hostname = "localhost"
		appConf.Port = "8080"
		appConf.Symbols = make([]string, 2)
		appConf.Symbols[0] = "BTCUSD"
		appConf.Symbols[1] = "ETHBTC"
	}
	// Allocate space for currencies array
	currencies = make([]currencyInfo, len(appConf.Symbols))
	// Allocate space for map
	symbolToIndex = make(map[string]int)

	// Initialize concurrency lock for each currencies entry
	currencyInfoLock = make([]sync.Mutex, len(appConf.Symbols))

	// Initialize symbol to index map
	for i := range appConf.Symbols {
		symbolToIndex[appConf.Symbols[i]] = i
	}
}

// strcuture retreived by ticker update websocket API
type tickerUpdate struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  struct {
		Ask         string    `json:"ask"`
		Bid         string    `json:"bid"`
		Last        string    `json:"last"`
		Open        string    `json:"open"`
		Low         string    `json:"low"`
		High        string    `json:"high"`
		Volume      string    `json:"volume"`
		VolumeQuote string    `json:"volumeQuote"`
		Timestamp   time.Time `json:"timestamp"`
		Symbol      string    `json:"symbol"`
	} `json:"params"`
}

var addr = flag.String("addr", "api.hitbtc.com", "http service address")

type symbolParam struct {
	Symbol string `json:"symbol"`
}

type getSymbolCommand struct {
	Method string      `json:"method"`
	Params symbolParam `json:"params"`
	ID     int         `json:"id"`
}

type getSymbolResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		ID                   string `json:"id"`
		BaseCurrency         string `json:"baseCurrency"`
		QuoteCurrency        string `json:"quoteCurrency"`
		QuantityIncrement    string `json:"quantityIncrement"`
		TickSize             string `json:"tickSize"`
		TakeLiquidityRate    string `json:"takeLiquidityRate"`
		ProvideLiquidityRate string `json:"provideLiquidityRate"`
		FeeCurrency          string `json:"feeCurrency"`
	} `json:"result"`
	ID int `json:"id"`
}

func getSymbol(c *websocket.Conn, symbol string) (*getSymbolResponse, error) {
	commandStruct := getSymbolCommand{
		Method: "getSymbol",
		Params: symbolParam{
			Symbol: symbol,
		},
		ID: 123,
	}
	commandString, _ := json.Marshal(&commandStruct)
	errWrite := c.WriteMessage(websocket.TextMessage, commandString)
	if errWrite != nil {
		log.Println("Failed to send getSymbol command", errWrite)
		return nil, errWrite
	}
	_, message, errRead := c.ReadMessage()
	if errRead != nil {
		log.Println("Failed to read response to getSymbol command", errRead)
		return nil, errRead
	}
	gs := getSymbolResponse{}
	if errUnmarshall := json.Unmarshal(message, &gs); errUnmarshall != nil {
		log.Println("Failed to unmarshal response to getSymbol command", errUnmarshall)
		return nil, errUnmarshall
	}
	return &gs, nil
}

type subscribeTickerCommand struct {
	Method string      `json:"method"`
	Params symbolParam `json:"params"`
	ID     int         `json:"id"`
}

func subscribeToTicker(c *websocket.Conn, symbol string) (err error) {
	commandStruct := subscribeTickerCommand{
		Method: "subscribeTicker",
		Params: symbolParam{
			Symbol: symbol,
		},
		ID: 123,
	}
	commandString, _ := json.Marshal(&commandStruct)

	return c.WriteMessage(websocket.TextMessage, commandString)
}

type getCurrencyParam struct {
	Currency string `json:"currency"`
}

type getCurrencyCommand struct {
	Method string           `json:"method"`
	Params getCurrencyParam `json:"params"`
	ID     int              `json:"id"`
}

type getCurrencyResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		ID                  string `json:"id"`
		FullName            string `json:"fullName"`
		Crypto              bool   `json:"crypto"`
		PayinEnabled        bool   `json:"payinEnabled"`
		PayinPaymentID      bool   `json:"payinPaymentId"`
		PayinConfirmations  int    `json:"payinConfirmations"`
		PayoutEnabled       bool   `json:"payoutEnabled"`
		PayoutIsPaymentID   bool   `json:"payoutIsPaymentId"`
		TransferEnabled     bool   `json:"transferEnabled"`
		Delisted            bool   `json:"delisted"`
		PayoutFee           string `json:"payoutFee"`
		PayoutMinimalAmount string `json:"payoutMinimalAmount"`
		PrecisionPayout     int    `json:"precisionPayout"`
		PrecisionTransfer   int    `json:"precisionTransfer"`
	} `json:"result"`
	ID int `json:"id"`
}

func getCurrency(c *websocket.Conn, currency string) (*getCurrencyResponse, error) {
	commandStruct := getCurrencyCommand{
		Method: "getCurrency",
		Params: getCurrencyParam{
			Currency: currency,
		},
		ID: 123,
	}
	commandString, _ := json.Marshal(&commandStruct)
	errWrite := c.WriteMessage(websocket.TextMessage, commandString)
	if errWrite != nil {
		log.Println("Failed to send getCurrency command", errWrite)
		return nil, errWrite
	}
	_, message, errRead := c.ReadMessage()
	if errRead != nil {
		log.Println("Failed to read response to getCurrency command", errRead)
		return nil, errRead
	}
	gcs := getCurrencyResponse{}
	if errUnmarshall := json.Unmarshal(message, &gcs); errUnmarshall != nil {
		log.Println("Failed to unmarshal response to getSymbol command", errUnmarshall)
		return nil, errUnmarshall
	}
	return &gcs, nil
}

func connectToAPI() *websocket.Conn {
	u := url.URL{Scheme: "wss", Host: *addr, Path: "/api/2/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	return c
}

func initializeCurrencyInfo(c *websocket.Conn) error {
	for i := range appConf.Symbols {
		gsResponse, gsErr := getSymbol(c, appConf.Symbols[i])
		if gsErr != nil {
			log.Printf("getSymbol %v failed: %v", appConf.Symbols[i], gsErr)
			return gsErr
		}
		currencies[i].FeeCurrency = gsResponse.Result.FeeCurrency
		currencies[i].ID = gsResponse.Result.BaseCurrency
		gcs, gcsErr := getCurrency(c, currencies[i].ID)
		if gcsErr != nil {
			log.Printf("getCurrency %v failed: %v", appConf.Symbols[i], gcsErr)
			return gcsErr
		}
		currencies[i].FullName = gcs.Result.FullName

	}
	return nil
}

func setCurrencyInfo(tu *tickerUpdate) {
	currencyIndex := symbolToIndex[tu.Params.Symbol]
	currencyInfoLock[currencyIndex].Lock()
	currencies[currencyIndex].Ask = tu.Params.Ask
	currencies[currencyIndex].Bid = tu.Params.Bid
	currencies[currencyIndex].Last = tu.Params.Last
	currencies[currencyIndex].Open = tu.Params.Open
	currencies[currencyIndex].Low = tu.Params.Low
	currencies[currencyIndex].High = tu.Params.High
	currencyInfoLock[currencyIndex].Unlock()

}

func getSingleCurrency(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	symbol := vars["symbol"]
	currencyIndex, ok := symbolToIndex[symbol]
	if !ok {
		fmt.Fprintf(w, "Unsupported symbol %v!  Use one of:\n", symbol)
		for i := range appConf.Symbols {
			fmt.Fprintf(w, "%v\n", appConf.Symbols[i])
		}
		return
	}
	currencyInfoLock[currencyIndex].Lock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(currencies[currencyIndex])
	currencyInfoLock[currencyIndex].Unlock()
}

type getAllCurrenciesResponse struct {
	Currencies []currencyInfo `json:"currencies"`
}

func getAllCurrencies(w http.ResponseWriter, r *http.Request) {
	numCurrencies := len(currencies)
	var result []currencyInfo
	result = make([]currencyInfo, numCurrencies)
	for i := range currencies {
		currencyInfoLock[i].Lock()
		result[i] = currencies[i]
		currencyInfoLock[i].Unlock()
	}
	var resp = getAllCurrenciesResponse{
		Currencies: result,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

}

func setupRoutes() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/currency/all", getAllCurrencies)
	r.HandleFunc("/currency/{symbol}", getSingleCurrency)
	return r
}

func main() {
	doConfig()
	flag.Parse()
	log.SetFlags(0)

	c := connectToAPI()
	defer c.Close()

	initializeCurrencyInfo(c)

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			tu := tickerUpdate{}
			if err = json.Unmarshal(message, &tu); err != nil {
				panic(err)
			}
			setCurrencyInfo(&tu)

		}
	}()

	for i := range appConf.Symbols {
		if err := subscribeToTicker(c, appConf.Symbols[i]); err != nil {
			log.Println("write:", err)
			return
		}

	}

	server := &http.Server{Addr: appConf.Hostname + ":" + appConf.Port, Handler: setupRoutes()}
	log.Printf("Server listening on host %v, port %v...", appConf.Hostname, appConf.Port)

	go func() {
		log.Fatal(server.ListenAndServe())
	}()

	// Allow server to be interrupted
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	<-interrupt

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Wait with timeout for server to close connection
	server.Shutdown(ctx)

}
