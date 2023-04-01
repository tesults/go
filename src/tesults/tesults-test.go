package tesultstest

import (
	"fmt"
	"strconv"

	"github.com/tesults/go/src/tesults/tesults"
)

func main() {
	data := map[string]interface{}{
		"target": "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpZCI6Ijg4MjY2Y2RiLWE1YzItNGMyNi1iYzg4LTBlZWViMmIzM2M4MS0xNDkyNzg3MTE4OTA1IiwiZXhwIjo0MTAyNDQ0ODAwMDAwLCJ2ZXIiOiIwIiwic2VzIjoiMjA5ZWU2NDAtNmIzNy00YWVmLWI3YjItMWMwNzE2ZjljYThlIiwidHlwZSI6InQifQ.2QSmHLpzTLaWwckKonaztEp8QClF8Z0grTHx8Q0tHw0",
		"results": map[string]interface{}{
			"cases": []interface{}{
				map[string]interface{}{
					"name":   "Test 1",
					"desc":   "Test 1 description",
					"suite":  "Suite A",
					"result": "pass",
					"files":  []string{"C:\\Users\\gbdhaliaj\\Desktop\\log.txt", "C:\\Users\\gbdhaliaj\\Desktop\\test1.png"},
				},
				map[string]interface{}{
					"name":   "Test 2",
					"desc":   "Test 2 description",
					"suite":  "Suite B",
					"result": "pass",
					"files":  []string{"C:\\Users\\gbdhaliaj\\Desktop\\test2.png"},
				},
				map[string]interface{}{
					"name":   "Test 3",
					"desc":   "Test 3 description",
					"suite":  "Suite A",
					"result": "fail",
					"reason": "Assert fail in line 203 of example.go",
					"files":  []string{"C:\\Users\\gbdhaliaj\\Desktop\\test3.png"},
				},
			},
		},
	}

	res := tesults.Results(data)

	fmt.Println("Success: ", res["success"])
	fmt.Println("Message: ", res["message"])
	fmt.Println("Warnings: ", strconv.FormatInt(int64(len(res["warnings"].([]string))), 10))
	fmt.Println("Errors: ", strconv.FormatInt(int64(len(res["errors"].([]string))), 10))

	if len(res["errors"].([]string)) > 0 {
		fmt.Println("First error: " + res["errors"].([]string)[0])
	}

	if len(res["warnings"].([]string)) > 0 {
		fmt.Println("First warning: " + res["warnings"].([]string)[0])
	}
}
