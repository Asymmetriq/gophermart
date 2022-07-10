package gophermart

// import (
// 	"bytes"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"io/ioutil"
// 	"net/http"
// 	"net/http/httptest"
// 	"strings"
// 	"testing"

// 	"github.com/Asymmetriq/gophermart/internal/config"
// 	"github.com/Asymmetriq/gophermart/internal/cookie"
// 	"github.com/golang/mock/gomock"
// 	"github.com/stretchr/testify/require"
// )

// var (
// 	defaultUserID, _ = cookie.GetSignedUserID()
// 	defaultShortURL  = "/short-url-mock"
// )

// type reqParams struct {
// 	method string
// 	path   string
// 	value  io.Reader
// }

// type want struct {
// 	contentType string
// 	statusCode  int
// 	value       string
// }

// type testCase struct {
// 	name   string
// 	params reqParams
// 	want   want
// }

// func TestPositive_getHandler(t *testing.T) {
// 	tests := []testCase{
// 		{
// 			name: "positive 1: redirect case",
// 			params: reqParams{
// 				method: http.MethodGet,
// 				path:   defaultShortURL,
// 			},
// 			want: want{
// 				contentType: "text/html; charset=ISO-8859-1",
// 				statusCode:  http.StatusOK,
// 				value:       "", // тело не проверяем, так как происходит редирект
// 			},
// 		},
// 	}
// 	for _, tt := range tests {
// 		ctrl := gomock.NewController(t)
// 		repo := mock.NewMockRepository(ctrl)

// 		repo.EXPECT().GetURL(gomock.Any(), "short-url-mock").Return("https://www.google.com", nil)

// 		ts := httptest.NewServer(Newgophermart(repo, config.InitConfig()))
// 		defer ts.Close()

// 		t.Run(tt.name, func(t *testing.T) {
// 			response, respBody := testRequest(t, ts.URL, tt.params.method, tt.params.path, tt.params.value)
// 			checkResults(t, tt, response.StatusCode, ts.URL+respBody, response.Header.Get("Content-Type"), "")
// 			response.Body.Close()
// 		})
// 	}
// }

// func TestNegative_getHandler(t *testing.T) {
// 	tests := []testCase{
// 		{
// 			name: "negative 1: unknown short url",
// 			params: reqParams{
// 				method: http.MethodGet,
// 				path:   "/wow-url",
// 			},
// 			want: want{
// 				contentType: "text/plain; charset=utf-8",
// 				statusCode:  http.StatusBadRequest,
// 				value:       fmt.Sprintf("no original url found with shortcut %q\n", "wow-url"),
// 			},
// 		},
// 	}
// 	for _, tt := range tests {
// 		ctrl := gomock.NewController(t)
// 		repo := mock.NewMockRepository(ctrl)

// 		repo.EXPECT().GetURL(gomock.Any(), gomock.Any()).Return("", fmt.Errorf("no original url found with shortcut %q", "wow-url"))

// 		ts := httptest.NewServer(Newgophermart(repo, config.InitConfig()))
// 		defer ts.Close()

// 		t.Run(tt.name, func(t *testing.T) {
// 			response, respBody := testRequest(t, ts.URL, tt.params.method, tt.params.path, tt.params.value)
// 			checkResults(t, tt, response.StatusCode, respBody, response.Header.Get("Content-Type"), "")
// 			response.Body.Close()
// 		})
// 	}
// }

// func TestPositive_postHandler(t *testing.T) {
// 	tests := []testCase{
// 		{
// 			name: "test positive 1",
// 			params: reqParams{
// 				method: http.MethodPost,
// 				path:   "/",
// 				value:  strings.NewReader("https://www.google.com"),
// 			},
// 			want: want{
// 				contentType: "application/text",
// 				statusCode:  http.StatusCreated,
// 				value:       "Cc4PMy5iGtEuaJH3KtPwePaTGHVqNMSKBtUrWhyTjo4t",
// 			},
// 		},
// 		{
// 			name: "test positive 2",
// 			params: reqParams{
// 				method: http.MethodPost,
// 				path:   "/",
// 				value:  strings.NewReader("FKLSDFKLSDFKLSDFKSD"),
// 			},
// 			want: want{
// 				contentType: "application/text",
// 				statusCode:  http.StatusCreated,
// 				value:       "4D71kVApBvoxqSyojJxysa1c67g2DWUfJuR4oMPRSmQy",
// 			},
// 		},
// 	}
// 	for _, tt := range tests {
// 		ctrl := gomock.NewController(t)
// 		repo := mock.NewMockRepository(ctrl)
// 		repo.EXPECT().SetURL(gomock.Any(), gomock.Any()).Return(nil)

// 		ts := httptest.NewServer(Newgophermart(repo, config.InitConfig()))
// 		defer ts.Close()

// 		t.Run(tt.name, func(t *testing.T) {
// 			response, respBody := testRequest(t, ts.URL, tt.params.method, tt.params.path, tt.params.value)
// 			checkResults(t, tt, response.StatusCode, respBody, response.Header.Get("Content-Type"), ts.URL)
// 			response.Body.Close()
// 		})
// 	}
// }

// func TestNegative_postHandler(t *testing.T) {
// 	// TODO
// 	t.Skip()
// 	tests := []testCase{
// 		{
// 			name: "test negative 1",
// 			params: reqParams{
// 				method: http.MethodPost,
// 				path:   "/",
// 				value:  strings.NewReader(""),
// 			},
// 			want: want{
// 				contentType: "text/plain; charset=utf-8",
// 				statusCode:  http.StatusBadRequest,
// 				value:       "no request body\n",
// 			},
// 		},
// 	}
// 	for _, tt := range tests {
// 		ctrl := gomock.NewController(t)
// 		repo := mock.NewMockRepository(ctrl)

// 		ts := httptest.NewServer(Newgophermart(repo, config.InitConfig()))
// 		defer ts.Close()

// 		t.Run(tt.name, func(t *testing.T) {
// 			response, respBody := testRequest(t, ts.URL, tt.params.method, tt.params.path, tt.params.value)
// 			checkResults(t, tt, response.StatusCode, respBody, response.Header.Get("Content-Type"), "")
// 			response.Body.Close()
// 		})
// 	}
// }

// func TestPositive_jsonHandler(t *testing.T) {
// 	// TODO
// 	t.Skip()
// 	tests := []testCase{
// 		{
// 			name: "test positive 1",
// 			params: reqParams{
// 				method: http.MethodPost,
// 				path:   "/api/shorten",
// 				value:  newJSONBody("https://www.google.com"),
// 			},
// 			want: want{
// 				contentType: "application/json",
// 				statusCode:  http.StatusCreated,
// 				value:       "/short-url-mock",
// 			},
// 		},
// 		{
// 			name: "test positive 2",
// 			params: reqParams{
// 				method: http.MethodPost,
// 				path:   "/api/shorten",
// 				value:  newJSONBody("FKLSDFKLSDFKLSDFKSD"),
// 			},
// 			want: want{
// 				contentType: "application/json",
// 				statusCode:  http.StatusCreated,
// 				value:       "/short-url-mock",
// 			},
// 		},
// 	}
// 	for _, tt := range tests {
// 		ctrl := gomock.NewController(t)
// 		repo := mock.NewMockRepository(ctrl)
// 		repo.EXPECT().SetURL(gomock.Any(), gomock.Any()).Return(nil)

// 		ts := httptest.NewServer(Newgophermart(repo, config.InitConfig()))
// 		defer ts.Close()

// 		m, err := json.Marshal(struct {
// 			Result string `json:"result"`
// 		}{Result: (tt.want.value)})
// 		if err != nil {
// 			t.Fatal(err)
// 		}
// 		tt.want.value = string(m)
// 		t.Run(tt.name, func(t *testing.T) {
// 			response, respBody := testRequest(t, ts.URL, tt.params.method, tt.params.path, tt.params.value)
// 			checkResults(t, tt, response.StatusCode, respBody, response.Header.Get("Content-Type"), ts.URL)
// 			response.Body.Close()
// 		})
// 	}
// }

// func testRequest(t *testing.T, serverURL string, method, path string, value io.Reader) (*http.Response, string) {
// 	req, err := http.NewRequest(method, serverURL+path, value)
// 	require.NoError(t, err)

// 	req.AddCookie(&http.Cookie{
// 		Name:  string(cookie.Name),
// 		Value: defaultUserID,
// 	})

// 	resp, err := http.DefaultClient.Do(req)
// 	require.NoError(t, err)

// 	respBody, err := ioutil.ReadAll(resp.Body)
// 	require.NoError(t, err)

// 	defer resp.Body.Close()

// 	return resp, string(respBody)
// }

// func checkResults(t *testing.T, tt testCase, code int, value, contentType, host string) {
// 	require.Equal(t, tt.want.statusCode, code, "status codes don't match")
// 	require.Equal(t, tt.want.contentType, contentType)
// 	if tt.want.value != "" {
// 		fullURL := tt.want.value
// 		if host != "" {
// 			fullURL = host + "/" + tt.want.value
// 		}
// 		require.Equal(t, fullURL, value, "shortened urls don't match")
// 	}

// }

// func newJSONBody(url string) io.Reader {
// 	result, _ := json.Marshal(struct {
// 		URL string `json:"url"`
// 	}{URL: url})
// 	return bytes.NewBuffer(result)
// }
