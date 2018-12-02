package server

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/vwxyzjn/portwarden/web"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	drive "google.golang.org/api/drive/v2"
)

// GetClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func GetClient(ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile, err := tokenCacheFile()
	if err != nil {
		log.Fatalf("Unable to get path to cached credential file. %v", err)
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		tok = GetTokenFromWeb(config)
		SaveToken(cacheFile, tok)
	}
	return config.Client(ctx, tok)
}

// GetTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func GetTokenFromWeb(config *oauth2.Config) *oauth2.Token {
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
func SaveToken(file string, token *oauth2.Token) {
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

// UploadFile upload the fileBytes to Google Drive's portwarden folder
// https://gist.github.com/tzmartin/f5732091783752660b671c20479f519a
func UploadFile(fileBytes []byte, token *oauth2.Token) error {
	client := web.GoogleDriveAppConfig.Client(oauth2.NoContext, token)
	srv, err := drive.New(client)
	if err != nil {
		return err
	}
	mimeType := http.DetectContentType(fileBytes)

	parentId, err := GetOrCreateFolder(srv, "portwarden_backup")
	if err != nil {
		return err
	}

	fmt.Println("Start upload")
	f := &drive.File{Title: time.Now().Format("01-02-2006") + ".portwarden", MimeType: mimeType}
	if parentId != "" {
		p := &drive.ParentReference{Id: parentId}
		f.Parents = []*drive.ParentReference{p}
	}

	_, err = srv.Files.Insert(f).Media(bytes.NewReader(fileBytes)).Do()
	if err != nil {
		return err
	}

	return nil
}

func GetOrCreateFolder(srv *drive.Service, folderName string) (string, error) {
	folderId := ""
	if folderName == "" {
		return "", nil
	}
	q := fmt.Sprintf("title=\"%s\" and mimeType=\"application/vnd.google-apps.folder\"", folderName)

	r, err := srv.Files.List().Q(q).MaxResults(1).Do()
	if err != nil {
		return "", err
	}

	if len(r.Items) > 0 {
		folderId = r.Items[0].Id
	} else {
		// no folder found create new
		f := &drive.File{Title: folderName, Description: "Auto Create by gdrive-upload", MimeType: "application/vnd.google-apps.folder"}
		r, err := srv.Files.Insert(f).Do()
		if err != nil {
			return "", err
		}
		folderId = r.Id
	}
	return folderId, nil
}

/*
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

	client := GetClient(ctx, config)

	fileBytes := []byte("xixix")
	if err != nil {
		log.Fatalf("Unable to read file for upload: %v", err)
	}
	//TODO: Call uploadFile with appropriate params: []byte of encrypted data and client instance

}
*/
