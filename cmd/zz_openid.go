/*
Copyright © 2020 Alessandro Segala (@ItalyPaleAle)

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/

package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/statiko-dev/stkcli/utils"
)

func openIDAuthCommand(method string) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		baseURL, client := getURLClient()

		// Invoke the /info endpoint to see what's the authentication method
		var rInfo infoResponseModel
		err := utils.RequestJSON(utils.RequestOpts{
			Client: client,
			Target: &rInfo,
			URL:    baseURL + "/info",
		})
		if err != nil {
			utils.ExitWithError(utils.ErrorNode, "Request failed", err)
			return
		}

		// Ensure the node supports authentication with the requested method
		var openIdConfig *infoResponseModelOpenID
		extraQs := ""
		name := ""
		switch method {
		case "auth0":
			if !utils.SliceContainsString(rInfo.AuthMethods, "auth0") || rInfo.Auth0 == nil {
				utils.ExitWithError(utils.ErrorUser, "This node does not support authenticating with Auth0", nil)
				return
			}
			name = "Auth0"
			openIdConfig = rInfo.Auth0
		case "azuread":
			if !utils.SliceContainsString(rInfo.AuthMethods, "azureAD") || rInfo.AzureAD == nil {
				utils.ExitWithError(utils.ErrorUser, "This node does not support authenticating with an Azure AD account", nil)
				return
			}
			name = "Azure AD"
			extraQs = "&domain_hint=organizations"
			openIdConfig = rInfo.AzureAD
		default:
			utils.ExitWithError(utils.ErrorApp, "Invalid OpenID provider: "+method, nil)
			return
		}

		// Redirect users to the authentication URL
		state := time.Now().Unix()
		authorizeURL := fmt.Sprintf("%s?client_id=%s&response_type=code&redirect_uri=%s&response_mode=query&scope=openid+offline_access&state=%d%s", openIdConfig.AuthorizeURL, openIdConfig.ClientID, url.QueryEscape("http://localhost:3993"), state, extraQs)
		utils.LaunchBrowser(authorizeURL)

		// Start a web server to listen to authorization codes
		authCode := ""
		ctx, ctxCancel := context.WithCancel(context.Background())
		defer ctxCancel()
		mux := http.NewServeMux()
		server := &http.Server{
			Addr:           "127.0.0.1:3993",
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
			Handler:        mux,
		}
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Ensure we have the code in the response
			query := r.URL.Query()
			if query != nil && query.Get("code") != "" {
				if query.Get("state") == strconv.FormatInt(state, 10) {
					authCode = query.Get("code")
					fmt.Fprintf(w, "Authenticated with %s. You can close this window.", name)
					ctxCancel()
				} else {
					fmt.Fprintf(w, "Error: invalid state in response")
				}
			} else {
				fmt.Fprintf(w, "Error: response did not contain an authorization code")
			}
		})
		go func() {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				panic(err)
			}
		}()
		select {
		// Shutdown the server when the context is canceled
		case <-ctx.Done():
			server.Shutdown(ctx)
		}

		// Exchange the authorization code for a token
		body := url.Values{}
		// No client_secret because this is a client-side app
		body.Set("client_id", openIdConfig.ClientID)
		body.Set("code", authCode)
		body.Set("grant_type", "authorization_code")
		body.Set("redirect_uri", "http://localhost:3993")
		body.Set("scope", "openid offline_access")

		// Request
		var rToken struct {
			ExpiresIn    int    `json:"expires_in"`
			IDToken      string `json:"id_token"`
			RefreshToken string `json:"refresh_token"`
		}
		err = utils.RequestJSON(utils.RequestOpts{
			Body:            strings.NewReader(body.Encode()),
			BodyContentType: "application/x-www-form-urlencoded",
			Method:          utils.RequestPOST,
			Target:          &rToken,
			URL:             openIdConfig.TokenURL,
		})
		if err != nil {
			utils.ExitWithError(utils.ErrorNode, "Request failed", err)
			return
		}

		if rToken.IDToken == "" || rToken.RefreshToken == "" {
			utils.ExitWithError(utils.ErrorNode, "Response did not contain an id_token or a refresh_token", nil)
			return
		}

		// Test the auth token by requesting the node's site list, invoking the /site endpoint
		// We're not requesting anything from the response
		err = utils.RequestJSON(utils.RequestOpts{
			Authorization: rToken.IDToken,
			Client:        client,
			URL:           baseURL + "/site",
		})
		if err != nil {
			// Check if the error is a 401
			if strings.HasPrefix(err.Error(), "invalid response status code: 401") {
				utils.ExitWithError(utils.ErrorUser, "Node did not accept the token provided by "+name, nil)
			} else {
				utils.ExitWithError(utils.ErrorNode, "Request failed", err)
			}
			return
		}

		// Store the key in the node store
		if err := nodeStore.StoreAuthToken(optAddress, rToken.IDToken, rToken.RefreshToken, openIdConfig.ClientID, openIdConfig.TokenURL); err != nil {
			utils.ExitWithError(utils.ErrorApp, "Error while storing the token", err)
			return
		}

		fmt.Println("Success! You're authenticated")
	}
}
