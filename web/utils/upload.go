package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/antonholmquist/jason"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile, err := tokenCacheFile()
	if err != nil {
		log.Fatalf("Unable to get path to cached credential file. %v", err)
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(cacheFile, tok)
	}
	return config.Client(ctx, tok)
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir,
		url.QueryEscape("google-drive-golang.json")), err
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.Create(file)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func randStr(strSize int, randType string) string {

	var dictionary string

	if randType == "alphanum" {
		dictionary = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	}

	if randType == "alpha" {
		dictionary = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	}

	if randType == "number" {
		dictionary = "0123456789"
	}

	var bytes = make([]byte, strSize)
	rand.Read(bytes)
	for k, v := range bytes {
		bytes[k] = dictionary[v%byte(len(dictionary))]
	}
	return string(bytes)
}

func main() {

	ctx := context.Background()

	// process the credential file
	credential, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(credential, drive.DriveScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	client := getClient(ctx, config)

	// get our token
	cacheFile, err := tokenCacheFile()
	if err != nil {
		log.Fatalf("Unable to get path to cached credential file. %v", err)
	}

	token, err := tokenFromFile(cacheFile)
	if err != nil {
		log.Fatalf("Unable to get token from file. %v", err)
	}

	//
	// Multipart upload method
	// see https://developers.google.com/drive/v3/web/manage-uploads
	fileName := "test.zip"
	fileBytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Fatalf("Unable to read file for upload: %v", err)
	}

	fileMIMEType := http.DetectContentType(fileBytes)

	// Simple upload method will cause the filename to be named "Untitled"
	postURL := "https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart"

	// extract auth or access token from Token file
	// see https://godoc.org/golang.org/x/oauth2#Token
	authToken := token.AccessToken

	boundary := randStr(32, "alphanum")

	uploadData := []byte("\n" +
		"--" + boundary + "\n" +
		"Content-Type: application/json; charset=" + string('"') + "UTF-8" + string('"') + "\n\n" +
		"{ \n" +
		string('"') + "name" + string('"') + ":" + string('"') + fileName + string('"') + "\n" +
		"} \n\n" +
		"--" + boundary + "\n" +
		"Content-Type:" + fileMIMEType + "\n\n" +
		string(fileBytes) + "\n" +

		"--" + boundary + "--")

	// post to Drive with RESTful method
	request, _ := http.NewRequest("POST", postURL, strings.NewReader(string(uploadData)))
	request.Header.Add("Host", "www.googleapis.com")
	request.Header.Add("Authorization", "Bearer "+authToken)
	request.Header.Add("Content-Type", "multipart/related; boundary="+string('"')+boundary+string('"'))
	request.Header.Add("Content-Length", strconv.FormatInt(request.ContentLength, 10))

	// debug
	//fmt.Println(request)

	response, err := client.Do(request)
	if err != nil {
		log.Fatalf("Unable to be post to Google API: %v", err)
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		log.Fatalf("Unable to read Google API response: %v", err)
	}

	// output the response from Drive API
	fmt.Println(string(body))

	// we need to extract the uploaded file ID to execute Update command
	jsonAPIreply, _ := jason.NewObjectFromBytes(body)

	uploadedFileID, _ := jsonAPIreply.GetString("id")
	fmt.Println("Uploaded file ID : ", uploadedFileID)
}
