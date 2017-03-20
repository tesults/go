package main

import "./tesults"
import "fmt"

func main() {
    data := map[string]interface{}{
        "target": "token",
        "results": map[string]interface{}{
            "cases": []interface{}{
                map[string]interface{}{
                    "name": "Test 1",
                    "desc": "Test 1 description",
                    "suite": "Suite A",
                    "result": "pass",
                },
                map[string]interface{}{
                    "name": "Test 2",
                    "desc": "Test 2 description",
                    "suite": "Suite B",
                    "result": "pass",
                },
                map[string]interface{}{
                    "name": "Test 3",
                    "desc": "Test 3 description",
                    "suite": "Suite A",
                    "result": "fail",
                    "reason": "Assert fail in line 203 of example.go",
                },
            },
        },
    }
    
    res := tesults.Results(data)
    
    fmt.Println("success: ", res["success"])
    fmt.Println("message: ", res["message"])
}