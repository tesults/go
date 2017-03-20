package tesults

import (
    "net/http"
    "bytes"
    "io/ioutil"
    "encoding/json"
)

func Results(data map[string]interface{}) map[string]interface{} {
    url := "https://www.tesults.com/results"
    
    jsonString, err := json.Marshal(data)
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonString))
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    body, _ := ioutil.ReadAll(resp.Body)
    
    var res map[string]interface{}
    err = json.Unmarshal([]byte(body), &res)
    if err != nil {
        panic(err)
    }
    
    var success = false
    var message = ""
    
    if val, ok := res["error"]; ok {
        success = false
        message = val.(map[string]interface{})["message"].(string)
    }
    
    if val, ok := res["data"]; ok {
        success = true
        message = val.(map[string]interface{})["message"].(string)
    }
    
    return map[string]interface{}{
                "success": success,
                "message": message,
            }
}