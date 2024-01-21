package main

import (
    "encoding/json"
    "errors"
    "fmt"
    "log"
    "math"
    "net/http"
    "net/url"
    "os"
    "strconv"

    "github.com/go-chi/chi/v5"
    "gopkg.in/yaml.v2"
)

// CFGFILE - path to config file
const CFGFILE string = "config.yml"

// Config struct
type Config struct {
    ListenPort int32              `yaml:"listen_port"`
    BeelineAPI string             `yaml:"beeline_api"`
    Accounts   map[string]Account `yaml:"accounts"`
    DebugMode  bool               `yaml:"debug"`
}

// Account struct
type Account struct {
    Login    int64  `yaml:"login"`
    Password string `yaml:"password"`
    Token    string `yaml:"token"`
}

// Summary struct
type Summary struct {
    Name      string
    Number    int64
    Status    string
    Gigabytes float64
    Minutes   float64
    SMS       float64
    Balance   float64
}

// CFG - config object
var CFG Config

// ColorReset - ANSI color
const ColorReset string = "\033[0m"

// ColorRed - ANSI color
const ColorRed string = "\033[31m"

// ColorGreen - ANSI color
const ColorGreen string = "\033[32m"

// ColorYellow - ANSI color
const ColorYellow string = "\033[33m"

// ColorBlue - ANSI color
const ColorBlue string = "\033[34m"

// ColorPurple - ANSI color
const ColorPurple string = "\033[35m"

// ColorCyan - ANSI color
const ColorCyan string = "\033[36m"

// ColorWhite - ANSI color
const ColorWhite string = "\033[37m"

// debug log
func logDebug(msg string) {
    if CFG.DebugMode {
        log.Printf("[%sDEBUG%s] %s", ColorCyan, ColorReset, msg)
    }
}

// info log
func logInfo(msg string) {
    log.Printf("[%sINFO%s] %s", ColorGreen, ColorReset, msg)
}

// warning log
func logWarning(msg string) {
    log.Printf("[%sWARNING%s] %s", ColorYellow, ColorReset, msg)
}

// error log
func logError(msg string) {
    log.Printf("[%sERROR%s] %s", ColorRed, ColorReset, msg)
}

// read config from yaml
func readYML(cfg interface{}, filename string) error {
    data, err := os.ReadFile(filename)
    if err != nil {
        logError(fmt.Sprintf("[config] Read file failed: %v", err))
        return err
    }
    err = yaml.Unmarshal(data, cfg)
    if err != nil {
        logError(fmt.Sprintf("[config] Parse yaml failed: %v", err))
        return err
    }
    logInfo(fmt.Sprintf("[config] Loaded %s", filename))
    return nil
}

// write config to yaml
func writeYML(cfg interface{}, filename string) error {
    // encode to yaml
    data, err := yaml.Marshal(cfg)
    if err != nil {
        logError(fmt.Sprintf("[config] YAML marshal failed: %v", err))
        return err
    }
    // attach document start and end strings
    data = append([]byte("---\n"), data...)
    data = append(data, []byte("...\n")...)
    err = os.WriteFile(filename, data, 0644)
    if err != nil {
        logError(fmt.Sprintf("[config] Write file failed: %v", err))
        return err
    }
    logInfo(fmt.Sprintf("[config] Saved %s", filename))
    return nil
}

// init configuration
func initConfig() error {
    // load main config
    err := readYML(&CFG, CFGFILE)
    if err != nil {
        return err
    }
    return nil
}

// get request from api
func apiGet(endpoint string, params map[string]string) (map[string]interface{}, error) {
    var res map[string]interface{}
    method := "GET"
    req, err := http.NewRequest("GET", CFG.BeelineAPI+endpoint, nil)
    if err != nil {
        return res, err
    }
    q := url.Values{}
    for k, v := range params {
        q.Add(k, v)
    }
    req.URL.RawQuery = q.Encode()
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        logError(fmt.Sprintf("[API %s] Request failed: %v, url: %s", method, err, req.URL.String()))
        return res, errors.New("API request failed")
    }
    defer resp.Body.Close()

    // parse response
    err = json.NewDecoder(resp.Body).Decode(&res)
    if err != nil {
        logError(fmt.Sprintf("[API %s] Response json decode failed: %v, endpoint: %s", method, err, endpoint))
        return res, errors.New("API response decode failed")
    }
    // if we have no http errors - check result
    if resp.StatusCode == 200 {
        logDebug(fmt.Sprintf("[API %s] Response: %+v", method, res))
        // parse errors from api
        meta := res["meta"].(map[string]interface{})
        if meta["status"] != "OK" {
            logWarning(fmt.Sprintf("[API %s] Returned %v error: %v, endpoint: %s", method, meta["code"], meta["message"], endpoint))
            return res, errors.New(meta["message"].(string))
        }
        return res, nil
    }
    logError(fmt.Sprintf("[API %s] Returned %d error, raw response: %#v, endpoint: %s", method, resp.StatusCode, res, endpoint))
    return res, fmt.Errorf("%d", resp.StatusCode)
}

// login and update token
func updateToken(accName string) error {
    acc := CFG.Accounts[accName]
    res, err := apiGet("auth", map[string]string{"login": strconv.FormatInt(acc.Login, 10), "password": acc.Password})
    if err != nil {
        return err
    }
    acc.Token = res["token"].(string)
    CFG.Accounts[accName] = acc
    writeYML(&CFG, CFGFILE)
    return nil
}

// get account counters
func getSummary(accName string) (Summary, error) {
    var resp map[string]interface{}
    var res Summary
    var err error
    acc := CFG.Accounts[accName]
    res.Name = accName
    res.Number = acc.Login
    // check token
    if acc.Token == "" {
        err = updateToken(accName)
        if err != nil {
            return res, err
        }
        // update value
        acc = CFG.Accounts[accName]
    }
    // check status
    resp, err = apiGet("info/status", map[string]string{"ctn": strconv.FormatInt(acc.Login, 10), "token": acc.Token})
    // if token expired
    if err != nil {
        if err.Error() == "TOKEN_NOT_FOUND" || err.Error() == "TOKEN_EXPIRED" {
            err = updateToken(accName)
            if err != nil {
                return res, err
            }
            // run again with updated token
            return getSummary(accName)
        }
        return res, err
    }
    res.Status = resp["status"].(string)

    // get counters
    resp, err = apiGet("info/prepaidAddBalance", map[string]string{"ctn": strconv.FormatInt(acc.Login, 10), "token": acc.Token})
    if err == nil {
        if resp["balanceTime"] != nil {
            tmp := resp["balanceTime"].([]interface{})[0]
            res.Minutes = tmp.(map[string]interface{})["value"].(float64) / 60
        }
        if resp["balanceSMS"] != nil {
            tmp := resp["balanceSMS"].([]interface{})[0]
            res.SMS = tmp.(map[string]interface{})["value"].(float64)
        }
        if resp["balanceData"] != nil {
            tmp := resp["balanceData"].([]interface{})[0]
            res.Gigabytes = tmp.(map[string]interface{})["value"].(float64) / (1024 * 1024 * 1024)
        }
    }

    // check plan and add counters from different endpoint
    resp, err = apiGet("info/pricePlan", map[string]string{"ctn": strconv.FormatInt(acc.Login, 10), "token": acc.Token})
    if err == nil && resp["pricePlanInfo"].(map[string]interface{})["name"].(string) == "VYOUNG" {
        resp, err = apiGet("info/accumulators", map[string]string{"ctn": strconv.FormatInt(acc.Login, 10), "token": acc.Token})
        if err == nil {
            for _, v := range resp["accumulators"].([]interface{}) {
                tmp := v.(map[string]interface{})
                if tmp["soc"] == "VYOUNG" {
                    res.Gigabytes = tmp["rest"].(float64) / (1024 * 1024)
                }
            }
        }
    }

    // round Gigabytes
    res.Gigabytes = math.Round(res.Gigabytes*100) / 100

    // get total balance
    resp, err = apiGet("info/prepaidBalance", map[string]string{"ctn": strconv.FormatInt(acc.Login, 10), "token": acc.Token})
    if err == nil {
        res.Balance = resp["balance"].(float64)
    }

    return res, err
}

// endpoint /accounts
func summaryHandler(w http.ResponseWriter, r *http.Request) {
    logDebug(fmt.Sprintf("[server] [all] Get rquest from %s", r.Header.Get("X-Forwarded-For")))
    var res []Summary
    for a := range CFG.Accounts {
        acc, _ := getSummary(a)
        res = append(res, acc)
    }
    json.NewEncoder(w).Encode(res)
}

// endpoint /accounts/{accName}
func accHandler(w http.ResponseWriter, r *http.Request) {
    accName := chi.URLParam(r, "accName")
    logDebug(fmt.Sprintf("[server] [%s] Get rquest from %s", accName, r.Header.Get("X-Forwarded-For")))
    res, err := getSummary(accName)
    if err != nil {
        http.Error(w, err.Error(), 500)
    } else {
        json.NewEncoder(w).Encode(res)
    }
}

// MAIN APP
func main() {
    err := initConfig()
    if err != nil {
        log.Fatal(err)
    }
    logDebug(fmt.Sprintf("[CFG] %+v", CFG))
    r := chi.NewRouter()
    r.Get("/accounts", summaryHandler)
    r.Get("/accounts/{accName}", accHandler)
    logInfo(fmt.Sprintf("Starting server on port %d", CFG.ListenPort))
    log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", CFG.ListenPort), r))
}
