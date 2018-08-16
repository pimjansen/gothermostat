package nest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/marcofranssen/gothermostat/config"
)

type apiError struct {
	Error       string `json:"error,omitempty"`
	Description string `json:"error_description,omitempty"`
	InstanceID  string `json:"instance_id,omitempty"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token,omitempty"`
	ExpiresIn   int    `json:"expires_in,omitempty"`
}

type nest struct {
	config *config.Config
}

type Nest interface {
	Authenticate() error
	All(combined *Combined) error
	Devices(devices *Devices) error
}

func New(config *config.Config) Nest {
	return &nest{config: config}
}

func (n *nest) Authenticate() error {
	if len(n.config.AuthCode) <= 0 {
		fmt.Printf("Go to %s and get a authCode and put it in your config file.\n", n.config.AuthURL)
	}

	if len(n.config.AccessToken) <= 0 {
		tokenResp, err := getAccessToken(n.config)
		if err != nil {
			return err
		}

		n.config.AccessToken = tokenResp.AccessToken

		fmt.Println(tokenResp)
	}
	return nil
}

func (n *nest) Devices(devices *Devices) error {
	return n.get("/devices", devices)
}

func (n *nest) All(combined *Combined) error {
	return n.get("/", combined)
}

func (n *nest) get(path string, response interface{}) error {
	client := http.Client{
		CheckRedirect: checkRedirect,
	}

	url := fmt.Sprintf("https://developer-api.nest.com%s", path)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", n.config.AccessToken))

	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("Api request failed: %s\n%s", resp.Status, body)
	}

	err = json.Unmarshal(body, response)
	return err
}

func checkRedirect(redirRequest *http.Request, via []*http.Request) error {
	// Go's http.DefaultClient does not forward headers when a redirect 3xx
	// response is received. Thus, the header (which in this case contains the
	// Authorization token) needs to be passed forward to the redirect
	// destinations.
	redirRequest.Header = via[0].Header

	// Go's http.DefaultClient allows 10 redirects before returning an
	// an error. We have mimicked this default behavior.s
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	return nil
}

func getAccessToken(cfg *config.Config) (tokenResponse, error) {
	var tokenResp tokenResponse
	authURL := fmt.Sprintf(cfg.TokenURL+"?client_id=%s&client_secret=%s&code=%s&grant_type=authorization_code", cfg.ClientID, cfg.ClientSecret, cfg.AuthCode)
	resp, err := http.Post(authURL, "x-www-form-urlencoded", nil)
	if err != nil {
		return tokenResp, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return tokenResp, err
	}

	if resp.StatusCode != 200 {
		return tokenResp, fmt.Errorf("accesstoken failed: %s\n%s", resp.Status, body)
	}

	err = json.Unmarshal(body, &tokenResp)
	return tokenResp, err
}
