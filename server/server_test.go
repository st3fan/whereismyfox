package main

import "encoding/json"
import "net/http"
import "net/http/httptest"
import "testing"
import "github.com/emicklei/go-restful"

type MockPersona struct {
	LoggedIn bool
}

func (self MockPersona) IsLoggedIn(r *http.Request) bool {
	return self.LoggedIn
}

func (self MockPersona) GetLoginName(r *http.Request) string {
	if self.LoggedIn {
		return "ggp@mozilla.com"
	}

	return ""
}

func (self MockPersona) Logout(w http.ResponseWriter, r *http.Request) {
	panic("Logout should not have been called!")
}

func (self MockPersona) Login(verifierURL string, w http.ResponseWriter, r *http.Request) error {
	panic("Login should not have been called!")
}

func initTestingServer(t *testing.T) func() {
	db, cleanup := initTestDatabase(t)

	gDB = db
	gServerConfig = ServerConfig{}
	gPersona = MockPersona{LoggedIn: true}

	restful.Add(createDeviceWebService())
	return func() {
		cleanup()
		gDB = nil
		gServerConfig = ServerConfig{}
		gPersona = nil
	}
}

func doRequest(method, url string) *httptest.ResponseRecorder {
	request, _ := http.NewRequest(method, url, nil)
	response := httptest.NewRecorder()
	restful.DefaultDispatch(response, request)
	return response
}

func TestUnauthorizedAccess(t *testing.T) {
	cleanup := initTestingServer(t)
	defer cleanup()

	gPersona = MockPersona{LoggedIn: false}

	unauthorized := []string{"/device/", "/device/1"}
	for _, url := range unauthorized {
		response := doRequest("GET", url)
		if response.Code != http.StatusUnauthorized {
			t.Errorf("Unexpected response code: %d", response.Code)
		}
	}
}

func TestServeDevicesByUser(t *testing.T) {
	cleanup := initTestingServer(t)
	defer cleanup()

	response := doRequest("GET", "/device")
	if response.Code != http.StatusOK {
		t.Errorf("Unexpected response code: %d", response.Code)
	}

	expected := []string{"/device/1", "/device/2"}
	result := []string{}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Error("Failed to unmarshal response: " + err.Error())
	}

	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("Got unexpected reply from the server: %#v", result)
		}
	}
}

func TestServeDevice(t *testing.T) {
	cleanup := initTestingServer(t)
	defer cleanup()

	response := doRequest("GET", "/device/1")
	result := Device{}

	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Error("Failed to unmarshal response: " + err.Error())
	}

	expected := gTestDevices[0]
	if result != expected {
		t.Error("Mismatch in device response: %#v != %#v", result, expected)
	}
}
