package tesults

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func refreshCredentials(target string, key string) map[string]interface{} {
	data := map[string]interface{}{
		"target": target,
		"key":    key,
	}

	url := "https://www.tesults.com/permitupload"

	jsonString, err := json.Marshal(data)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonString))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": "Error requesting refresh credentials.",
		}
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var res map[string]interface{}
	err = json.Unmarshal([]byte(body), &res)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": "Error processing refresh credentials response.",
		}
	}

	if val, ok := res["error"]; ok {
		return map[string]interface{}{
			"success": false,
			"message": val.(map[string]interface{})["message"].(string),
		}
	}

	if val, ok := res["data"]; ok {
		return map[string]interface{}{
			"success": true,
			"message": val.(map[string]interface{})["message"].(string),
			"upload":  val.(map[string]interface{})["upload"],
		}
	}

	return map[string]interface{}{
		"success": false,
		"message": "Unexpected response from refresh credentials.",
	}
}

func createS3Client(auth map[string]interface{}) *s3manager.Uploader {
	creds := credentials.NewStaticCredentials(auth["AccessKeyId"].(string), auth["SecretAccessKey"].(string), auth["SessionToken"].(string))
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"),
		Credentials: creds,
	}))
	uploader := s3manager.NewUploader(sess)
	return uploader
}

func filesUpload(files []map[string]interface{}, keyPrefix string, auth map[string]interface{}, target string) map[string]interface{} {
	expireBuffer := int64(30) // 30 seconds
	filesUploaded := int64(0)
	bytesUploaded := int64(0)
	expiration := int64(auth["Expiration"].(float64))
	//expiration, _ := strconv.ParseInt(expirationString, 10, 64)

	warnings := make([]string, 0)
	uploading := int(0)
	maxActiveUploads := 10 // Upload at most 10 files simultaneously to avoid hogging the client machine.
	s3Client := createS3Client(auth)

	for len(files) != 0 || uploading != 0 {
		if uploading < maxActiveUploads && len(files) != 0 {
			// Check if new credentials required.
			now := int64(time.Now().Unix())
			if now+expireBuffer > expiration { // Check within 30 seconds of expiry.
				// Refresh credentials.
				if uploading == 0 {
					// Wait for all current transfers to complete so we can get a new S3 client.

					response := refreshCredentials(target, keyPrefix)
					if response["success"].(bool) != true {
						// Must stop upload due to failure to be permitted for new credentials.
						warnings = append(warnings, response["message"].(string))
						break
					} else {
						upload := response["upload"].(map[string]interface{})
						keyPrefix = upload["key"].(string)
						uploadMessage := upload["message"].(string)
						permit := upload["permit"].(bool)
						auth = upload["auth"].(map[string]interface{})
						if permit != true {
							// Must stop upload due to failure to be permitted for new credentials.
							warnings = append(warnings, uploadMessage)
							break
						} else {
							// Upload permitted.
							expiration = int64(auth["Expiration"].(float64))
							//expiration, _ = strconv.ParseInt(expirationString, 10, 64)
							s3Client = createS3Client(auth)
						}
					}
				}
			}

			if now+expireBuffer < expiration {
				// Load new file for upload.

				batchNumber := maxActiveUploads - uploading
				if len(files) < batchNumber {
					batchNumber = len(files)
				}

				var wg sync.WaitGroup
				for i := 0; i < batchNumber; i++ {
					file := files[0]
					files = append(files[:0], files[1:]...)
					fNum := int64(file["num"].(int))
					fFile := file["file"].(string)

					if _, err := os.Stat(fFile); os.IsNotExist(err) {
						// file does not exist
						warnings = append(warnings, "File not found: "+fFile)
					} else {
						// Perform an upload.
						key := keyPrefix + "/" + strconv.FormatInt(fNum, 10) + "/" + filepath.Base(fFile)

						file, err := os.Open(fFile)
						if err != nil {
							warnings = append(warnings, "Unable to read file: "+fFile)
						} else {

							fi, e := os.Stat(fFile)
							if e != nil {
								warnings = append(warnings, "Unable to access file: "+fFile)
							} else {
								fileSize := fi.Size()

								body := bufio.NewReader(file)

								var bucket = "tesults-results"
								params := &s3manager.UploadInput{
									Bucket: &bucket,
									Key:    &key,
									Body:   body,
								}

								uploading++
								wg.Add(1)
								go func(key string, size int64) {
									_, err := s3Client.Upload(params)
									if err != nil {
										warnings = append(warnings, "Unable to upload file: "+fFile)
									} else {
										filesUploaded++
										bytesUploaded += size
									}
									uploading--
									defer wg.Done()
								}(key, fileSize)
							}
						}
					}
				}
				wg.Wait()
			}
		}

		// Check if existing upload complete - handled by each transfer.
	}

	return map[string]interface{}{
		"message":  "Success. " + strconv.FormatInt(filesUploaded, 10) + " files uploaded. " + strconv.FormatInt(bytesUploaded, 10) + " bytes uploaded.",
		"warnings": warnings,
	}
}

func filesInTestCases(data map[string]interface{}) []map[string]interface{} {
	results := data["results"].(map[string]interface{})
	cases := results["cases"].([]interface{})
	files := make([](map[string]interface{}), 0)
	num := 0
	for _, c := range cases {
		if caseFiles, ok := c.(map[string]interface{})["files"]; ok {
			for _, caseFile := range caseFiles.([]string) {
				files = append(files, map[string]interface{}{
					"num":  num,
					"file": caseFile,
				})
			}
		}
		num++
	}
	return files
}

func validateInput(data map[string]interface{}) map[string]interface{} {
	type1 := map[string]interface{}{}
	type2 := make([]string, 0)
	type3 := []interface{}{}
	type4 := "string"

	valid := map[string]interface{}{
		"valid":   true,
		"message": "",
	}

	if reflect.TypeOf(data).Kind() != reflect.TypeOf(type1).Kind() {
		valid["valid"] = false
		valid["message"] = "test results input must be of type map[string]interface{}"
		return valid
	}

	target := data["target"]

	if reflect.TypeOf(target).Kind() != reflect.TypeOf(type4).Kind() {
		valid["valid"] = false
		valid["message"] = "target must be of type string."
		return valid
	}

	results := data["results"]

	if reflect.TypeOf(results).Kind() != reflect.TypeOf(type1).Kind() {
		valid["valid"] = false
		valid["message"] = "results must be of type map[string]interface{}"
		return valid
	}

	cases := results.(map[string]interface{})["cases"]

	if reflect.TypeOf(cases).Kind() != reflect.TypeOf(type3).Kind() {
		valid["valid"] = false
		valid["message"] = "cases must be of type []interface{}"
		return valid
	}

	for _, c := range cases.([]interface{}) {
		if reflect.TypeOf(c).Kind() != reflect.TypeOf(type1).Kind() {
			valid["valid"] = false
			valid["message"] = "each case must be of type map[string]interface{}"
			return valid
		}

		files := c.(map[string]interface{})["files"]

		if files != nil {
			if reflect.TypeOf(files).Kind() != reflect.TypeOf(type2).Kind() {
				valid["valid"] = false
				valid["message"] = "each case must be of type []string"
				return valid
			}

			for _, f := range files.([]string) {
				if reflect.TypeOf(f).Kind() != reflect.TypeOf(type4).Kind() {
					valid["valid"] = false
					valid["message"] = "each file must be of type string"
					return valid
				}
			}
		}
	}

	return valid
}

// Results uploads test results data to Tesults.
func Results(data map[string]interface{}) map[string]interface{} {
	validate := validateInput(data)
	valid := validate["valid"].(bool)
	validMessage := validate["message"].(string)

	if valid != true {
		invalidMessage := "Invalid input: " + validMessage
		invalidWarnings := make([]string, 0)
		invalidErrors := make([]string, 0)
		invalidErrors = append(invalidErrors, invalidMessage)

		return map[string]interface{}{
			"success":  false,
			"message":  invalidMessage,
			"warnings": invalidWarnings,
			"errors":   invalidErrors,
		}
	}

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
	warnings := make([]string, 0)
	errors := make([]string, 0)

	if val, ok := res["error"]; ok {
		success = false
		message = val.(map[string]interface{})["message"].(string)
		errors = append(errors, message)
	}

	if val, ok := res["data"]; ok {
		success = true
		message = val.(map[string]interface{})["message"].(string)

		if upload, ok := val.(map[string]interface{})["upload"].(map[string]interface{}); ok {
			target := data["target"]
			files := filesInTestCases(data)

			key := upload["key"].(string)
			uploadMessage := upload["message"].(string)
			permit := upload["permit"].(bool)
			auth := upload["auth"].(map[string]interface{})

			if permit != true {
				warnings = append(warnings, uploadMessage)
			} else {
				// Upload required and permitted.
				fileUploadReturn := filesUpload(files, key, auth, target.(string)) // This can take a while.
				message = fileUploadReturn["message"].(string)
				warnings = fileUploadReturn["warnings"].([]string)
			}
		}
	}

	return map[string]interface{}{
		"success":  success,
		"message":  message,
		"warnings": warnings,
		"errors":   errors,
	}
}
